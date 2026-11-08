// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstallOpts sind die Auswahlmöglichkeiten der Installation.
type InstallOpts struct {
	Desktop   bool
	StartMenu bool
	Redist    bool
	SrcRoot   string // Stamm der Disc 1 (für optionale Zusatzdateien wie die Steam-Manifestdatei)
}

// exeDir liefert den Ordner der laufenden Programmdatei.
func exeDir() (string, string) {
	p, err := os.Executable()
	if err != nil {
		return "", ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Dir(p), p
}

// loadConfig liest launcher.ini aus dem angegebenen Ordner.
func loadConfig(dir string) (*Config, error) {
	f, err := os.Open(filepath.Join(dir, "launcher.ini"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseConfig(f)
}

// fsName macht aus dem Titel einen gültigen Datei-/Ordnernamen.
func fsName(s string) string {
	r := strings.NewReplacer("<", " ", ">", " ", ":", " -", "\"", "'", "/", "-", "\\", "-", "|", "-", "?", "", "*", "")
	s = strings.TrimSpace(r.Replace(s))
	s = strings.TrimRight(s, ". ")
	if s == "" {
		s = "Game"
	}
	return s
}

// DefaultDir ist der vorgeschlagene Installationsordner.
func DefaultDir(cfg *Config) string {
	name := cfg.DirName
	if name == "" {
		name = fsName(cfg.Title)
	}
	if cfg.IsSteam() {
		if libs := steamLibraries(); len(libs) > 0 {
			return filepath.Join(libs[0], "steamapps", "common", name)
		}
	}
	return filepath.Join(defaultGamesRoot(), name)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func exeFull(cfg *Config, dest string) string {
	rel, _ := safeRel(cfg.Exe)
	return filepath.Join(dest, rel)
}

// PostInstall erledigt alles nach dem Kopieren: Deinstaller, Registrierung, Verknüpfungen, Komponenten.
// Rückgabe: true, wenn mindestens eine Komponente nicht installiert werden konnte.
func PostInstall(cfg *Config, dest string, opts InstallOpts, status func(string)) (redistFailed bool) {
	lw, err := openLog(dest, cfg)
	if err != nil {
		return false
	}
	defer lw.Close()

	// Deinstaller = Kopie dieses Programms
	uninst := filepath.Join(dest, "uninstall.exe")
	if _, self := exeDir(); self != "" {
		if copyFile(self, uninst) == nil {
			lw.Add("F", "uninstall.exe")
		}
	}
	game := exeFull(cfg, dest)
	regRegister(cfg, dest, cfg.TotalSize()/1024, uninst)

	// Steam-Modus: Manifestdatei neben das Spiel legen, damit Steam es als vorhanden erkennt.
	// Entweder die mitgelieferte Datei (extras\appmanifest.acf) oder, wenn keine dabei ist, ein schlichtes selbst
	// erzeugtes Manifest. Eine bereits vorhandene Datei wird nie überschrieben.
	if cfg.IsSteam() {
		if apps, ok := steamAppsOf(dest); ok {
			dst := filepath.Join(apps, "appmanifest_"+cfg.SteamAppID+".acf")
			if _, e := os.Stat(dst); os.IsNotExist(e) {
				placed := false
				if cfg.SteamManifest != "" && opts.SrcRoot != "" {
					if rel, e := safeRel(cfg.SteamManifest); e == nil {
						if copyFile(filepath.Join(opts.SrcRoot, rel), dst) == nil {
							placed = true
						}
					}
				}
				if !placed {
					text := genManifest(cfg, filepath.Base(dest), time.Now())
					if os.WriteFile(dst, []byte(text), 0o644) == nil {
						placed = true
					}
				}
				if placed {
					lw.Add("X", dst)
				}
			}
		}
	}

	// steam_appid.txt anlegen, wenn sie fehlt und die AppID bekannt ist (wird bei der Deinstallation wieder entfernt).
	if rel := ensureSteamAppIDFile(cfg, dest); rel != "" {
		lw.Add("F", rel)
	}

	title := fsName(cfg.Title)
	if cfg.IsSteam() { // Verknüpfungen würden Steam umgehen
		opts.Desktop, opts.StartMenu = false, false
	}
	if opts.Desktop {
		if d := specialDir("desktop"); d != "" {
			l := filepath.Join(d, title+".lnk")
			if makeShortcut(l, game, filepath.Dir(game), game, cfg.Title) == nil {
				lw.Add("L", l)
			}
		}
	}
	if opts.StartMenu {
		if d := specialDir("programs"); d != "" {
			dir := filepath.Join(d, title)
			if os.MkdirAll(dir, 0o755) == nil {
				l1 := filepath.Join(dir, title+".lnk")
				if makeShortcut(l1, game, filepath.Dir(game), game, cfg.Title) == nil {
					lw.Add("L", l1)
				}
				l2 := filepath.Join(dir, T("uninstall")+" "+title+".lnk")
				if makeShortcut(l2, uninst, dest, uninst, T("uninstall")+" "+cfg.Title) == nil {
					lw.Add("L", l2)
				}
			}
		}
	}

	if opts.Redist {
		redistFailed = RunRedist(cfg, dest, status)
	}
	return redistFailed
}

// RunRedist startet die mitgelieferten Installationsprogramme für Zusatzkomponenten (Visual C++, DirectX ...)
// aus dem Spielordner. Rückgabe: true, wenn mindestens eine Komponente nicht installiert werden konnte.
func RunRedist(cfg *Config, dest string, status func(string)) (failed bool) {
	for _, r := range cfg.Redist {
		rel, err := safeRel(r.Path)
		if err != nil {
			continue
		}
		full := filepath.Join(dest, rel)
		if _, err := os.Stat(full); err != nil {
			continue
		}
		label := r.Label
		if label == "" {
			label = filepath.Base(full)
		}
		if status != nil {
			status(T("redistRun", label))
		}
		prog, args := full, r.Args
		if strings.EqualFold(filepath.Ext(full), ".msi") {
			prog, args = "msiexec.exe", `/i "`+full+`" `+r.Args
		}
		code, err := runInstaller(prog, args, filepath.Dir(full))
		// 0 = ok, 3010 = Neustart nötig, 1638 = neuere Version vorhanden, 1641 = Neustart eingeleitet
		if err != nil || !(code == 0 || code == 3010 || code == 1638 || code == 1641) {
			failed = true
		}
	}
	return failed
}

// PlayGame startet das Spiel: im Steam-Modus über Steam, sonst direkt die Programmdatei.
func PlayGame(cfg *Config, dest string) error {
	if cfg.IsSteam() {
		return launchFile("steam://rungameid/"+cfg.SteamAppID, "")
	}
	game := exeFull(cfg, dest)
	return launchFile(game, filepath.Dir(game))
}

// IsInstalled prüft anhand der Registrierung, ob das Spiel installiert ist und die Programmdatei existiert.
func IsInstalled(cfg *Config) (string, bool) {
	dest, ok := regLookup(cfg.PackageID)
	if !ok || dest == "" {
		return "", false
	}
	if _, err := os.Stat(exeFull(cfg, dest)); err != nil {
		return "", false
	}
	return dest, true
}

// RunUninstall führt die Deinstallation für den Ordner dest aus.
// askExtra wird gefragt, wenn zusätzliche Dateien vorhanden sind (true = alles löschen).
func RunUninstall(dest string, askExtra func() bool) (title string, err error) {
	info, err := ReadLog(dest)
	if err != nil {
		return "", err
	}
	removeExtra := false
	if info.CreatedDest && HasExtraFiles(dest, info) && askExtra != nil {
		removeExtra = askExtra()
	}
	skip := map[string]bool{}
	if _, self := exeDir(); self != "" {
		skip[strings.ToLower(self)] = true
	}
	pending, err := Uninstall(dest, info, removeExtra, skip)
	regUnregister(info.PackageID)
	if len(pending) > 0 {
		deleteSelfLater(pending, dest, removeExtra && info.CreatedDest)
	}
	return info.Title, err
}
