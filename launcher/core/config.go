// Package core contains the platform independent logic of the launcher
// (INI parsing, disc detection rules, staging of split BIN files).
package core

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type Extra struct{ Label, Path string }

// Part is one piece of a split BIN file, stored in SplitFiles/ on a given disc.
type Part struct {
	Name string
	Disc int
	Size int64  // 0 = unknown
	Dir  string // sub folder on the disc ("SplitFiles" for split parts, "" = disc root)
}

type SplitFile struct {
	Original string
	Parts    []Part
}

type Config struct {
	Title, Installer, Cover, Language string
	MediaType, Label, PackageID       string
	MediaCount                        int
	CapacityBytes, StagingBytes       int64
	Extras                            []Extra
	Splits                            []SplitFile // forced staging: BINs larger than the medium, parts in SplitFiles/
	Bins                              []SplitFile // optional quick mode: intact BINs in the disc root
}

// ParseINI reads GOGAutoLauncher.ini. Unknown keys are ignored.
func ParseINI(r io.Reader) Config {
	var c Config
	sec := ""
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\xef\xbb\xbf"))
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			sec = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.Trim(strings.TrimSpace(v), "\"")
		switch sec {
		case "game":
			switch k {
			case "title":
				c.Title = v
			case "installer":
				c.Installer = v
			case "cover":
				c.Cover = v
			case "language":
				c.Language = strings.ToLower(v)
			}
		case "", "disc", "staging":
			switch k {
			case "installer":
				if sec == "" {
					c.Installer = v
				}
			case "mediacount":
				c.MediaCount, _ = strconv.Atoi(v)
			case "mediatype":
				c.MediaType = v
			case "label":
				c.Label = v
			case "packageid":
				c.PackageID = v
			case "capacitybytes":
				c.CapacityBytes, _ = strconv.ParseInt(v, 10, 64)
			case "stagingbytes", "requiredbytes":
				c.StagingBytes, _ = strconv.ParseInt(v, 10, 64)
			}
		case "extras":
			if strings.HasPrefix(k, "item") {
				p := strings.SplitN(v, "|", 2)
				if len(p) == 2 && strings.TrimSpace(p[1]) != "" {
					c.Extras = append(c.Extras, Extra{strings.TrimSpace(p[0]), strings.TrimSpace(p[1])})
				}
			}
		case "splitfiles", "bins":
			if !strings.HasPrefix(k, "file") {
				continue
			}
			p := strings.SplitN(v, "|", 2)
			if len(p) != 2 {
				continue
			}
			sf := SplitFile{Original: strings.TrimSpace(p[0])}
			for _, e := range strings.Split(p[1], ",") {
				e = strings.TrimSpace(e)
				at := strings.LastIndex(e, "@")
				if at <= 0 {
					continue
				}
				rest := e[at+1:]
				var size int64
				if i := strings.Index(rest, ":"); i >= 0 {
					size, _ = strconv.ParseInt(rest[i+1:], 10, 64)
					rest = rest[:i]
				}
				d, err := strconv.Atoi(rest)
				if err == nil && d > 0 {
					dir := "SplitFiles"
					if sec == "bins" {
						dir = ""
					}
					sf.Parts = append(sf.Parts, Part{Name: e[:at], Disc: d, Size: size, Dir: dir})
				}
			}
			if len(sf.Parts) > 0 {
				if sec == "bins" {
					c.Bins = append(c.Bins, sf)
				} else {
					c.Splits = append(c.Splits, sf)
				}
			}
		}
	}
	if c.Title == "" {
		c.Title = "Your GOG Game"
	}
	if c.MediaCount < 1 {
		c.MediaCount = 1
	}
	return c
}

// Forced reports whether staging is mandatory (a BIN does not fit on one medium).
func (c *Config) Forced() bool { return len(c.Splits) > 0 }

// Plan returns the file list the staging engine works through.
func (c *Config) Plan(forced bool) []SplitFile {
	if forced {
		return c.Splits
	}
	return c.Bins
}

// FirstPartOnDisc returns the first part (any list) stored on the disc.
func (c *Config) FirstPartOnDisc(disc int) (Part, bool) {
	for _, list := range [][]SplitFile{c.Splits, c.Bins} {
		for _, s := range list {
			for _, p := range s.Parts {
				if p.Disc == disc {
					return p, true
				}
			}
		}
	}
	return Part{}, false
}
