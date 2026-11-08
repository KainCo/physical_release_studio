// Steam Disc Creator – Launcher/Installer
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Seg ist ein zusammenhängender Teil einer Datei auf einer bestimmten Disc.
type Seg struct {
	Disc   int
	Offset int64
	Size   int64
	CRC    uint32
	Rel    string // Pfad relativ zum Disc-Stamm (Backslashes)
}

// FileEntry ist eine Datei des Spiels (relativ zum Installationsordner).
type FileEntry struct {
	Size int64
	Path string // Backslashes
	Segs []Seg
}

// RedistEntry beschreibt ein mitgeliefertes Installationsprogramm (z. B. VC++).
type RedistEntry struct {
	Path  string
	Args  string
	Label string
}

// Config ist der Inhalt der launcher.ini.
type Config struct {
	Title     string
	Exe       string
	PackageID string
	DirName   string
	Banner    string
	ShowTitle bool
	DiscCount int
	Files     []FileEntry
	Redist    []RedistEntry

	// Steam-Modus: Die Dateien werden in eine Steam-Bibliothek kopiert; gestartet wird über Steam.
	Mode          string // "" oder "folder" = normaler Ordner, "steam" = Steam-Bibliothek
	SteamAppID    string
	SteamManifest string // optionale appmanifest_<id>.acf auf Disc 1 (relativer Pfad)
}

// IsSteam meldet, ob das Paket für eine Steam-Bibliothek gedacht ist (Spiel benötigt Steam).
func (c *Config) IsSteam() bool { return c.Mode == "steam" }

var pkgIDRe = regexp.MustCompile(`^[0-9a-f]{8,32}$`)
var steamIDRe = regexp.MustCompile(`^[0-9]{1,10}$`)

// ParseConfig liest die launcher.ini. Format siehe docs/FORMAT.md.
func ParseConfig(r io.Reader) (*Config, error) {
	cfg := &Config{ShowTitle: true}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	section := ""
	var cur *FileEntry
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			line = strings.TrimPrefix(line, "\xef\xbb\xbf") // UTF-8-BOM
			first = false
		}
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key, val := strings.TrimSpace(line[:eq]), line[eq+1:]
		switch section {
		case "game":
			switch strings.ToLower(key) {
			case "title":
				cfg.Title = strings.TrimSpace(val)
			case "exe":
				cfg.Exe = strings.TrimSpace(val)
			case "packageid":
				cfg.PackageID = strings.ToLower(strings.TrimSpace(val))
			case "dirname":
				cfg.DirName = strings.TrimSpace(val)
			case "banner":
				cfg.Banner = strings.TrimSpace(val)
			case "showtitle":
				cfg.ShowTitle = strings.TrimSpace(val) != "0"
			case "mode":
				cfg.Mode = strings.ToLower(strings.TrimSpace(val))
			case "steamappid":
				cfg.SteamAppID = strings.TrimSpace(val)
			case "steammanifest":
				cfg.SteamManifest = strings.TrimSpace(val)
			}
		case "disc":
			if strings.EqualFold(key, "count") {
				n, err := strconv.Atoi(strings.TrimSpace(val))
				if err != nil {
					return nil, fmt.Errorf("Disc/Count: %w", err)
				}
				cfg.DiscCount = n
			}
		case "files":
			switch strings.ToUpper(key) {
			case "F":
				p := strings.SplitN(val, "|", 2)
				if len(p) != 2 {
					return nil, fmt.Errorf("ungültige Zeile F=%s", val)
				}
				size, err := strconv.ParseInt(p[0], 10, 64)
				if err != nil || size < 0 {
					return nil, fmt.Errorf("ungültige Größe in F=%s", val)
				}
				cfg.Files = append(cfg.Files, FileEntry{Size: size, Path: p[1]})
				cur = &cfg.Files[len(cfg.Files)-1]
			case "S":
				if cur == nil {
					return nil, errors.New("S= ohne vorangehendes F=")
				}
				p := strings.SplitN(val, "|", 5)
				if len(p) != 5 {
					return nil, fmt.Errorf("ungültige Zeile S=%s", val)
				}
				disc, e1 := strconv.Atoi(p[0])
				off, e2 := strconv.ParseInt(p[1], 10, 64)
				size, e3 := strconv.ParseInt(p[2], 10, 64)
				crc, e4 := strconv.ParseUint(p[3], 16, 32)
				if e1 != nil || e2 != nil || e3 != nil || e4 != nil || off < 0 || size < 0 {
					return nil, fmt.Errorf("ungültige Zeile S=%s", val)
				}
				cur.Segs = append(cur.Segs, Seg{Disc: disc, Offset: off, Size: size, CRC: uint32(crc), Rel: p[4]})
			}
		case "redist":
			if strings.ToUpper(key) == "R" {
				p := strings.SplitN(val, "|", 3)
				if len(p) < 2 {
					return nil, fmt.Errorf("ungültige Zeile R=%s", val)
				}
				e := RedistEntry{Path: p[0], Args: p[1]}
				if len(p) == 3 {
					e.Label = p[2]
				}
				cfg.Redist = append(cfg.Redist, e)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// safeRel prüft, dass ein relativer Pfad nicht aus dem Zielordner ausbrechen kann
// (Schutz vor manipulierten Discs) und liefert ihn im Format des Betriebssystems.
func safeRel(p string) (string, error) {
	if p == "" {
		return "", errors.New("leerer Pfad")
	}
	parts := strings.Split(strings.ReplaceAll(p, "/", "\\"), "\\")
	for _, s := range parts {
		if s == "" || s == "." || s == ".." || strings.ContainsAny(s, ":*?\"<>|") {
			return "", fmt.Errorf("unzulässiger Pfad: %q", p)
		}
		for _, r := range s {
			if r < 32 {
				return "", fmt.Errorf("unzulässiger Pfad: %q", p)
			}
		}
		if strings.HasSuffix(s, " ") || strings.HasSuffix(s, ".") {
			return "", fmt.Errorf("unzulässiger Pfad: %q", p)
		}
	}
	return filepath.Join(parts...), nil
}

// Validate prüft die Konfiguration auf Vollständigkeit und Konsistenz.
func (c *Config) Validate() error {
	if c.Title == "" {
		return errors.New("Game/Title fehlt")
	}
	if !pkgIDRe.MatchString(c.PackageID) {
		return errors.New("Game/PackageID fehlt oder ist ungültig")
	}
	if c.DiscCount < 1 || c.DiscCount > 999 {
		return errors.New("Disc/Count ungültig")
	}
	if _, err := safeRel(c.Exe); err != nil {
		return fmt.Errorf("Game/Exe: %w", err)
	}
	if c.DirName != "" {
		if _, err := safeRel(c.DirName); err != nil {
			return fmt.Errorf("Game/DirName: %w", err)
		}
	}
	switch c.Mode {
	case "", "folder":
		if c.SteamAppID != "" && !steamIDRe.MatchString(c.SteamAppID) {
			return errors.New("Game/SteamAppID ist ungültig")
		}
	case "steam":
		if !steamIDRe.MatchString(c.SteamAppID) {
			return errors.New("Game/SteamAppID fehlt oder ist ungültig")
		}
		if c.SteamManifest != "" {
			if _, err := safeRel(c.SteamManifest); err != nil {
				return fmt.Errorf("Game/SteamManifest: %w", err)
			}
		}
	default:
		return fmt.Errorf("Game/Mode: unbekannter Wert %q", c.Mode)
	}
	for i := range c.Files {
		f := &c.Files[i]
		if _, err := safeRel(f.Path); err != nil {
			return err
		}
		sort.Slice(f.Segs, func(a, b int) bool { return f.Segs[a].Offset < f.Segs[b].Offset })
		var pos int64
		for _, s := range f.Segs {
			if s.Disc < 1 || s.Disc > c.DiscCount {
				return fmt.Errorf("%s: ungültige Disc-Nummer %d", f.Path, s.Disc)
			}
			if _, err := safeRel(s.Rel); err != nil {
				return err
			}
			if s.Offset != pos {
				return fmt.Errorf("%s: Teile sind lückenhaft", f.Path)
			}
			pos += s.Size
		}
		if pos != f.Size {
			return fmt.Errorf("%s: Summe der Teile (%d) passt nicht zur Dateigröße (%d)", f.Path, pos, f.Size)
		}
		if f.Size > 0 && len(f.Segs) == 0 {
			return fmt.Errorf("%s: keine Teile angegeben", f.Path)
		}
	}
	for _, r := range c.Redist {
		if _, err := safeRel(r.Path); err != nil {
			return fmt.Errorf("Redist: %w", err)
		}
	}
	return nil
}

// TotalSize ist die Summe aller Dateigrößen.
func (c *Config) TotalSize() int64 {
	var t int64
	for _, f := range c.Files {
		t += f.Size
	}
	return t
}
