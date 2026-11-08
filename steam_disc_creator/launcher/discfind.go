// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func discMarkerName(n int) string { return fmt.Sprintf("PHYSICAL_RELEASE_DISC_%d.txt", n) }

// checkDisc prüft, ob in root die Marker-Datei der Disc n zu diesem Paket liegt.
func checkDisc(root string, cfg *Config, n int) bool {
	b, err := os.ReadFile(filepath.Join(root, discMarkerName(n)))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "packageid=") {
			return strings.EqualFold(strings.TrimSpace(line[len("packageid="):]), cfg.PackageID)
		}
	}
	return false
}

// FindDisc sucht die Disc n. startDir ist der Ordner, aus dem das Programm gestartet wurde.
// Gesucht wird dort, in Geschwisterordnern (Ordnerausgabe: Disc1, Disc2, ...) und auf allen Laufwerken.
func FindDisc(cfg *Config, n int, startDir string, known map[int]string) (string, bool) {
	if r, ok := known[n]; ok && checkDisc(r, cfg, n) {
		return r, true
	}
	var cands []string
	if startDir != "" {
		cands = append(cands, startDir)
		parent := filepath.Dir(startDir)
		cands = append(cands, filepath.Join(parent, fmt.Sprintf("Disc%d", n)))
		if es, err := os.ReadDir(parent); err == nil {
			for _, e := range es {
				if e.IsDir() {
					cands = append(cands, filepath.Join(parent, e.Name()))
				}
			}
		}
	}
	for _, d := range driveRoots() {
		cands = append(cands, d, filepath.Join(d, fmt.Sprintf("Disc%d", n)))
	}
	seen := map[string]bool{}
	for _, c := range cands {
		key := strings.ToLower(filepath.Clean(c))
		if seen[key] {
			continue
		}
		seen[key] = true
		if checkDisc(c, cfg, n) {
			return c, true
		}
	}
	return "", false
}
