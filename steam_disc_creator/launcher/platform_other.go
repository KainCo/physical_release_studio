//go:build !windows

// Ersatzfunktionen für Nicht-Windows-Systeme. Sie dienen nur den automatischen Tests
// und dem Kommandozeilenmodus; der Launcher selbst ist ein Windows-Programm.
package main

import (
	"os"
	"path/filepath"
	"syscall"
)

func stateDir() string {
	if d := os.Getenv("PRS_STATE_DIR"); d != "" {
		return d
	}
	return filepath.Join(os.TempDir(), "prs-state")
}

func driveRoots() []string {
	if v := os.Getenv("PRS_DRIVES"); v != "" {
		return filepath.SplitList(v)
	}
	return nil
}

func defaultGamesRoot() string { return filepath.Join(os.TempDir(), "Games") }

func freeSpace(path string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}

func regLookup(pkgID string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(stateDir(), pkgID))
	if err != nil {
		return "", false
	}
	return string(b), true
}

func regRegister(cfg *Config, dest string, sizeKB int64, uninstall string) error {
	os.MkdirAll(stateDir(), 0o755)
	return os.WriteFile(filepath.Join(stateDir(), cfg.PackageID), []byte(dest), 0o644)
}

func regUnregister(pkgID string) { os.Remove(filepath.Join(stateDir(), pkgID)) }

func makeShortcut(lnk, target, workdir, icon, desc string) error {
	os.MkdirAll(filepath.Dir(lnk), 0o755)
	return os.WriteFile(lnk, []byte(target+"\n"), 0o644)
}

func specialDir(kind string) string {
	if d := os.Getenv("PRS_SPECIAL"); d != "" {
		return filepath.Join(d, kind)
	}
	return ""
}

func launchFile(path, dir string) error { return nil }

func runInstaller(path, args, dir string) (int, error) {
	if os.Getenv("PRS_REDIST_FAIL") != "" {
		return 1603, nil
	}
	return 0, nil
}

func deleteSelfLater(pending []string, dir string, removeDir bool) {
	for _, p := range pending {
		os.Remove(p)
	}
	os.Remove(filepath.Join(dir, logName))
	if removeDir {
		os.RemoveAll(dir)
	} else {
		os.Remove(dir)
	}
}

func uiLanguage() string { return "en" }

// steamRoot ist unter Windows der Steam-Ordner aus der Registry; hier nur für Tests per Umgebungsvariable.
func steamRoot() string { return os.Getenv("PRS_STEAM") }
