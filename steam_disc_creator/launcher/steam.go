// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Hilfsfunktionen für den Steam-Modus. Es werden ausschließlich lokale Dateien und die
// Registry gelesen; es gibt keinerlei Netzwerkzugriff und keine Eingriffe in Steam selbst.

var vdfPathRe = regexp.MustCompile(`"path"\s+"([^"]*)"`)

// parseLibraryFolders liest die Bibliothekspfade aus einer libraryfolders.vdf.
func parseLibraryFolders(text string) []string {
	var out []string
	for _, m := range vdfPathRe.FindAllStringSubmatch(text, -1) {
		p := strings.ReplaceAll(m[1], `\\`, `\`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// steamLibraries liefert alle bekannten Steam-Bibliotheken (die erste ist der Steam-Ordner selbst).
func steamLibraries() []string {
	root := steamRoot()
	if root == "" {
		return nil
	}
	libs := []string{filepath.Clean(root)}
	if b, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf")); err == nil {
		seen := map[string]bool{strings.ToLower(libs[0]): true}
		for _, p := range parseLibraryFolders(string(b)) {
			p = filepath.Clean(filepath.FromSlash(p))
			if !seen[strings.ToLower(p)] {
				seen[strings.ToLower(p)] = true
				libs = append(libs, p)
			}
		}
	}
	return libs
}

// steamAppsOf liefert den steamapps-Ordner, wenn dest die Form <Bibliothek>\steamapps\common\<Spiel> hat.
func steamAppsOf(dest string) (string, bool) {
	common := filepath.Dir(filepath.Clean(dest))
	apps := filepath.Dir(common)
	if strings.EqualFold(filepath.Base(common), "common") && strings.EqualFold(filepath.Base(apps), "steamapps") {
		return apps, true
	}
	return "", false
}

// acfText bereinigt Werte für die Manifestdatei (keine Anführungszeichen, Backslashes oder Zeilenumbrüche).
func acfText(s string) string {
	r := strings.NewReplacer("\"", "'", "\\", "/", "\r", " ", "\n", " ", "\t", " ")
	return strings.TrimSpace(r.Replace(s))
}

// genManifest erzeugt ein schlichtes appmanifest_<AppID>.acf für ein vollständig installiertes Spiel.
// Der Aufbau entspricht einer von Steam selbst geschriebenen Datei, enthält aber nur die nötigsten Felder.
func genManifest(cfg *Config, installDir string, now time.Time) string {
	kv := func(k, v string) string { return "\t\"" + k + "\"\t\t\"" + v + "\"\n" }
	var b strings.Builder
	b.WriteString("\"AppState\"\n{\n")
	b.WriteString(kv("appid", cfg.SteamAppID))
	b.WriteString(kv("Universe", "1"))
	b.WriteString(kv("name", acfText(cfg.Title)))
	b.WriteString(kv("StateFlags", "4"))
	b.WriteString(kv("installdir", acfText(installDir)))
	b.WriteString(kv("LastUpdated", strconv.FormatInt(now.Unix(), 10)))
	b.WriteString(kv("SizeOnDisk", strconv.FormatInt(cfg.TotalSize(), 10)))
	b.WriteString(kv("buildid", "0"))
	b.WriteString(kv("AutoUpdateBehavior", "0"))
	b.WriteString("}\n")
	return b.String()
}
