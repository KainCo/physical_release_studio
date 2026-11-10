// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const logName = "uninstall.lst"

var (
	// ErrCancelled wird zurückgegeben, wenn der Nutzer abbricht.
	ErrCancelled = errors.New("abgebrochen")
	// ErrCorrupt wird bei Prüfsummen- oder Längenfehlern verwendet.
	ErrCorrupt = errors.New("Datei beschädigt oder unvollständig")
)

// Hooks verbindet die Engine mit der Oberfläche.
type Hooks struct {
	// NeedDisc liefert den Stamm der Disc n (blockiert, bis sie gefunden wurde).
	NeedDisc func(n int) (string, error)
	// Progress meldet den Fortschritt in Bytes.
	Progress func(done, total int64, current string)
	// Retry wird bei Lese-/Prüffehlern gefragt; true = erneut versuchen.
	Retry func(file string, err error) bool
}

// LogWriter schreibt das Protokoll für die Deinstallation.
type LogWriter struct {
	f *os.File
	w *bufio.Writer
}

func openLog(dest string, cfg *Config) (*LogWriter, error) {
	p := filepath.Join(dest, logName)
	_, statErr := os.Stat(p)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	lw := &LogWriter{f: f, w: bufio.NewWriter(f)}
	if statErr != nil { // neue Datei: Kopfzeilen
		lw.Add("H", "LST 1")
		lw.Add("T", cfg.Title)
		lw.Add("P", cfg.PackageID)
		lw.Add("E", cfg.Exe)
		if cfg.IsSteam() {
			lw.Add("M", "steam")
		}
	}
	return lw, nil
}

// Add hängt einen Eintrag an und schreibt sofort auf die Platte (absturzsicher).
func (l *LogWriter) Add(kind, val string) {
	fmt.Fprintf(l.w, "%s|%s\n", kind, strings.ReplaceAll(val, "\n", " "))
	l.w.Flush()
}

// Close schließt das Protokoll.
func (l *LogWriter) Close() { l.w.Flush(); l.f.Close() }

// mkdirAllLogged legt Ordner an und protokolliert neu erzeugte (relativ zu dest, "." = dest selbst).
func mkdirAllLogged(dest, dir string, lw *LogWriter) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		if err := mkdirAllLogged(dest, parent, lw); err != nil {
			return err
		}
	}
	if err := os.Mkdir(dir, 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	rel, err := filepath.Rel(dest, dir)
	if err == nil && !strings.HasPrefix(rel, "..") {
		lw.Add("D", rel)
	}
	return nil
}

// CheckSpace prüft den freien Platz (mit 2 % Reserve).
func CheckSpace(cfg *Config, dest string) (need, free int64, ok bool) {
	need = cfg.TotalSize()
	need += need / 50
	probe := dest
	for {
		if _, err := os.Stat(probe); err == nil {
			break
		}
		next := filepath.Dir(probe)
		if next == probe {
			break
		}
		probe = next
	}
	f, err := freeSpace(probe)
	if err != nil {
		return need, -1, true // unbekannt: nicht blockieren
	}
	return need, f, f >= need
}

type segRef struct {
	file *FileEntry
	seg  Seg
}

// Install kopiert alle Dateien von den Discs in den Zielordner und prüft sie.
func Install(ctx context.Context, cfg *Config, dest string, h Hooks) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	dest = filepath.Clean(dest)
	destExisted := true
	if _, err := os.Stat(dest); err != nil {
		destExisted = false
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	lw, err := openLog(dest, cfg)
	if err != nil {
		return err
	}
	defer lw.Close()
	if !destExisted {
		lw.Add("D", ".")
	}

	var total int64
	byDisc := make(map[int][]segRef)
	for i := range cfg.Files {
		f := &cfg.Files[i]
		for _, s := range f.Segs {
			total += s.Size
			byDisc[s.Disc] = append(byDisc[s.Disc], segRef{f, s})
		}
	}

	touched := map[string]bool{}
	openFor := func(f *FileEntry) (*os.File, error) {
		rel, _ := safeRel(f.Path)
		full := filepath.Join(dest, rel)
		if err := mkdirAllLogged(dest, filepath.Dir(full), lw); err != nil {
			return nil, err
		}
		flags := os.O_WRONLY | os.O_CREATE
		if !touched[full] {
			flags |= os.O_TRUNC
			lw.Add("F", rel)
			touched[full] = true
		}
		return os.OpenFile(full, flags, 0o644)
	}

	// leere Dateien
	for i := range cfg.Files {
		f := &cfg.Files[i]
		if f.Size == 0 {
			ff, err := openFor(f)
			if err != nil {
				return err
			}
			ff.Close()
		}
	}

	var done int64
	for disc := 1; disc <= cfg.DiscCount; disc++ {
		refs := byDisc[disc]
		if len(refs) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return ErrCancelled
		}
		root, err := h.NeedDisc(disc)
		if err != nil {
			return err
		}
		for _, r := range refs {
			for {
				if ctx.Err() != nil {
					return ErrCancelled
				}
				n, err := copySeg(ctx, r, root, openFor, func(d int64) {
					if h.Progress != nil {
						h.Progress(done+d, total, r.file.Path)
					}
				})
				_ = n
				if err == nil {
					break
				}
				if errors.Is(err, ErrCancelled) {
					return err
				}
				if h.Retry != nil && h.Retry(r.file.Path, err) {
					// Datenträger evtl. gewechselt/gereinigt: erneut suchen
					if nr, e2 := h.NeedDisc(disc); e2 == nil {
						root = nr
					}
					continue
				}
				return fmt.Errorf("%s: %w", r.file.Path, err)
			}
			done += r.seg.Size
			if h.Progress != nil {
				h.Progress(done, total, r.file.Path)
			}
		}
	}

	// Größenkontrolle
	for i := range cfg.Files {
		rel, _ := safeRel(cfg.Files[i].Path)
		st, err := os.Stat(filepath.Join(dest, rel))
		if err != nil || st.Size() != cfg.Files[i].Size {
			return fmt.Errorf("%s: %w", cfg.Files[i].Path, ErrCorrupt)
		}
	}
	return nil
}

func copySeg(ctx context.Context, r segRef, root string, openFor func(*FileEntry) (*os.File, error), prog func(int64)) (int64, error) {
	srel, _ := safeRel(r.seg.Rel)
	src, err := os.Open(filepath.Join(root, srel))
	if err != nil {
		return 0, err
	}
	defer src.Close()
	dst, err := openFor(r.file)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	buf := make([]byte, 1<<20)
	h := crc32.NewIEEE()
	var copied int64
	lr := io.LimitReader(src, r.seg.Size)
	for {
		if ctx.Err() != nil {
			return copied, ErrCancelled
		}
		n, rerr := lr.Read(buf)
		if n > 0 {
			if _, werr := dst.WriteAt(buf[:n], r.seg.Offset+copied); werr != nil {
				return copied, werr
			}
			h.Write(buf[:n])
			copied += int64(n)
			prog(copied)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return copied, rerr
		}
	}
	if copied != r.seg.Size || h.Sum32() != r.seg.CRC {
		return copied, ErrCorrupt
	}
	return copied, nil
}

// UninstallInfo enthält die aus dem Protokoll gelesenen Angaben.
type UninstallInfo struct {
	Title, PackageID, Exe string
	Files, Dirs, Links    []string
	Extra, Groups         []string // X = einzelne Datei außerhalb des Zielordners, G = Startmenü-Gruppe
	CreatedDest           bool
	Steam                 bool // Paket wurde im Steam-Modus installiert (Hinweis bei der Deinstallation)
}

// ReadLog liest das Deinstallations-Protokoll im Zielordner.
func ReadLog(dest string) (*UninstallInfo, error) {
	f, err := os.Open(filepath.Join(dest, logName))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info := &UninstallInfo{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	seen := map[string]bool{}
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, '|')
		if i < 1 {
			continue
		}
		k, v := line[:i], line[i+1:]
		switch k {
		case "T":
			info.Title = v
		case "P":
			info.PackageID = v
		case "E":
			info.Exe = v
		case "M":
			info.Steam = v == "steam"
		case "F":
			if !seen["F"+v] {
				seen["F"+v] = true
				info.Files = append(info.Files, v)
			}
		case "D":
			if v == "." {
				info.CreatedDest = true
			} else if !seen["D"+v] {
				seen["D"+v] = true
				info.Dirs = append(info.Dirs, v)
			}
		case "L":
			if !seen["L"+v] {
				seen["L"+v] = true
				info.Links = append(info.Links, v)
			}
		case "X":
			if !seen["X"+v] {
				seen["X"+v] = true
				info.Extra = append(info.Extra, v)
			}
		case "G":
			if !seen["G"+v] {
				seen["G"+v] = true
				info.Groups = append(info.Groups, v)
			}
		}
	}
	return info, sc.Err()
}

func isRootLike(p string) bool {
	p = filepath.Clean(p)
	return filepath.Dir(p) == p
}

// Uninstall entfernt alle protokollierten Dateien. skip enthält Pfade (absolut), die nicht
// gelöscht werden dürfen (laufendes Programm); sie werden in pending zurückgegeben.
// removeExtra löscht den Ordner vollständig – nur wenn der Installer ihn selbst angelegt hat.
func Uninstall(dest string, info *UninstallInfo, removeExtra bool, skip map[string]bool) (pending []string, err error) {
	dest = filepath.Clean(dest)
	for _, rel := range info.Files {
		s, e := safeRel(rel)
		if e != nil {
			continue
		}
		full := filepath.Join(dest, s)
		if skip[strings.ToLower(full)] {
			pending = append(pending, full)
			continue
		}
		if e := os.Remove(full); e != nil && !os.IsNotExist(e) {
			err = e
		}
	}
	for _, l := range info.Links {
		os.Remove(l)
	}
	for _, x := range info.Extra {
		os.Remove(x)
	}
	for _, g := range info.Groups {
		os.Remove(g) // nur wenn leer; sonst schlägt das still fehl
	}
	dirs := append([]string(nil), info.Dirs...)
	sort.Slice(dirs, func(a, b int) bool { return len(dirs[a]) > len(dirs[b]) })
	for _, rel := range dirs {
		if s, e := safeRel(rel); e == nil {
			os.Remove(filepath.Join(dest, s))
		}
	}
	if removeExtra && info.CreatedDest && !isRootLike(dest) && len(pending) == 0 {
		// Nur Dateien im Ordner selbst: erst Protokoll weg, dann alles
		os.RemoveAll(dest)
		return nil, err
	}
	if len(pending) == 0 {
		os.Remove(filepath.Join(dest, logName))
		if info.CreatedDest {
			os.Remove(dest)
		}
	}
	return pending, err
}

// HasExtraFiles meldet, ob im Ordner Dateien liegen, die nicht von der Installation stammen.
func HasExtraFiles(dest string, info *UninstallInfo) bool {
	known := map[string]bool{strings.ToLower(logName): true}
	for _, f := range info.Files {
		if s, e := safeRel(f); e == nil {
			known[strings.ToLower(s)] = true
		}
	}
	extra := false
	filepath.WalkDir(dest, func(p string, d os.DirEntry, err error) error {
		if err != nil || extra {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(dest, p)
		if e == nil && !known[strings.ToLower(rel)] {
			extra = true
		}
		return nil
	})
	return extra
}
