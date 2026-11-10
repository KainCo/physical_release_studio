// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseLibraryFolders(t *testing.T) {
	vdf := "\"libraryfolders\"\n{\n\t\"0\"\n\t{\n\t\t\"path\"\t\t\"C:\\\\Program Files (x86)\\\\Steam\"\n\t}\n\t\"1\"\n\t{\n\t\t\"path\"\t\t\"D:\\\\SteamLibrary\"\n\t}\n}\n"
	got := parseLibraryFolders(vdf)
	want := []string{`C:\Program Files (x86)\Steam`, `D:\SteamLibrary`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSteamAppsOf(t *testing.T) {
	if a, ok := steamAppsOf(filepath.Join("x", "steamapps", "common", "Game")); !ok || a != filepath.Join("x", "steamapps") {
		t.Fatalf("steamAppsOf: %q %v", a, ok)
	}
	if _, ok := steamAppsOf(filepath.Join("x", "Games", "Game")); ok {
		t.Fatal("kein Steam-Ordner erwartet")
	}
}

func steamIni(base string) string {
	r := strings.Replace(base, "ShowTitle=1\n", "ShowTitle=1\nMode=steam\nSteamAppID=12345\nSteamManifest=extras\\appmanifest.acf\n", 1)
	return r
}

func TestSteamModeInstallCopiesManifestAndUninstallRemovesIt(t *testing.T) {
	roots, _, ini := buildDiscs(t)
	for _, r := range roots {
		os.WriteFile(filepath.Join(r, "launcher.ini"), []byte(steamIni(ini)), 0o644)
	}
	os.MkdirAll(filepath.Join(roots[0], "extras"), 0o755)
	os.WriteFile(filepath.Join(roots[0], "extras", "appmanifest.acf"), []byte("\"AppState\"\n{\n}\n"), 0o644)
	cfg, err := loadConfig(roots[0])
	if err != nil || !cfg.IsSteam() || cfg.SteamAppID != "12345" {
		t.Fatalf("Steam-Konfiguration: %+v %v", cfg, err)
	}
	lib := t.TempDir()
	dest := filepath.Join(lib, "steamapps", "common", "Test Spiel")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	t.Setenv("PRS_SPECIAL", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{Desktop: true, StartMenu: true, SrcRoot: roots[0]}, nil)
	acf := filepath.Join(lib, "steamapps", "appmanifest_12345.acf")
	if _, err := os.Stat(acf); err != nil {
		t.Fatal("appmanifest wurde nicht kopiert")
	}
	// Im Steam-Modus entstehen keine Verknüpfungen
	if es, _ := os.ReadDir(os.Getenv("PRS_SPECIAL")); len(es) != 0 {
		t.Fatal("im Steam-Modus dürfen keine Verknüpfungen angelegt werden")
	}
	if _, err := RunUninstall(dest, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(acf); err == nil {
		t.Fatal("appmanifest sollte mit entfernt werden")
	}
}

func TestExistingManifestIsNeverOverwritten(t *testing.T) {
	roots, _, ini := buildDiscs(t)
	for _, r := range roots {
		os.WriteFile(filepath.Join(r, "launcher.ini"), []byte(steamIni(ini)), 0o644)
	}
	os.MkdirAll(filepath.Join(roots[0], "extras"), 0o755)
	os.WriteFile(filepath.Join(roots[0], "extras", "appmanifest.acf"), []byte("NEU"), 0o644)
	cfg, _ := loadConfig(roots[0])
	lib := t.TempDir()
	os.MkdirAll(filepath.Join(lib, "steamapps"), 0o755)
	acf := filepath.Join(lib, "steamapps", "appmanifest_12345.acf")
	os.WriteFile(acf, []byte("ORIGINAL"), 0o644)
	dest := filepath.Join(lib, "steamapps", "common", "Test Spiel")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	Install(context.Background(), cfg, dest, hooksFor(cfg, roots))
	PostInstall(cfg, dest, InstallOpts{SrcRoot: roots[0]}, nil)
	RunUninstall(dest, func() bool { return false })
	if b, err := os.ReadFile(acf); err != nil || string(b) != "ORIGINAL" {
		t.Fatalf("vorhandene Manifestdatei wurde verändert oder gelöscht: %q %v", b, err)
	}
}

func TestSteamConfigValidation(t *testing.T) {
	base := "[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\n[Disc]\nCount=1\n"
	bad := []string{
		base[:len("[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\n")] + "Mode=steam\n[Disc]\nCount=1\n",                   // keine AppID
		"[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\nMode=steam\nSteamAppID=abc\n[Disc]\nCount=1\n",                    // AppID keine Zahl
		"[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\nMode=bogus\n[Disc]\nCount=1\n",                                    // unbekannter Modus
		"[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\nMode=steam\nSteamAppID=1\nSteamManifest=..\\x\n[Disc]\nCount=1\n", // Pfad
	}
	for i, ini := range bad {
		if _, err := ParseConfig(strings.NewReader(ini)); err == nil {
			t.Errorf("Fall %d: Fehler erwartet", i)
		}
	}
}

func TestInstalledDetection(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	if _, ok := IsInstalled(cfg); ok {
		t.Fatal("noch nicht installiert")
	}
	dest := filepath.Join(t.TempDir(), "G")
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{}, nil)
	if d, ok := IsInstalled(cfg); !ok || d != dest {
		t.Fatalf("Installation nicht erkannt: %q %v", d, ok)
	}
	os.Remove(filepath.Join(dest, "bin", "game.exe")) // Programmdatei fehlt => gilt als nicht installiert
	if _, ok := IsInstalled(cfg); ok {
		t.Fatal("ohne Programmdatei darf es nicht als installiert gelten")
	}
}

func TestRenderBannerClampsHeight(t *testing.T) {
	mk := func(w, h int) image.Image {
		im := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				im.Set(x, y, color.RGBA{200, 100, 50, 255})
			}
		}
		return im
	}
	cases := []struct{ w, h, want int }{{920, 430, 299}, {1000, 100, bannerMinH}, {500, 1000, bannerMaxH}}
	for _, c := range cases {
		b := renderBanner(mk(c.w, c.h), 640)
		if b == nil || b.W != 640 || b.H != c.want || len(b.BGRA) != 640*c.want*4 {
			t.Fatalf("%dx%d: %+v", c.w, c.h, b)
		}
	}
}

func writeIniAll(t *testing.T, roots []string, ini string) {
	t.Helper()
	for _, r := range roots {
		if err := os.WriteFile(filepath.Join(r, "launcher.ini"), []byte(ini), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSteamModeGeneratesManifestWhenNoneProvided(t *testing.T) {
	roots, _, ini := buildDiscs(t)
	writeIniAll(t, roots, strings.Replace(ini, "ShowTitle=1\n", "ShowTitle=1\nMode=steam\nSteamAppID=12345\n", 1))
	cfg, err := loadConfig(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	lib := t.TempDir()
	dest := filepath.Join(lib, "steamapps", "common", "Test Spiel")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	t.Setenv("PRS_SPECIAL", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{SrcRoot: roots[0]}, nil)

	acf := filepath.Join(lib, "steamapps", "appmanifest_12345.acf")
	b, err := os.ReadFile(acf)
	if err != nil {
		t.Fatal("es wurde kein Manifest erzeugt")
	}
	txt := string(b)
	for _, want := range []string{"\"AppState\"", "\"appid\"\t\t\"12345\"", "\"name\"\t\t\"Test Spiel\"", "\"installdir\"\t\t\"Test Spiel\"", "\"StateFlags\"\t\t\"4\""} {
		if !strings.Contains(txt, want) {
			t.Errorf("Manifest enthält %q nicht:\n%s", want, txt)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "bin", "steam_appid.txt")); err == nil {
		t.Fatal("der Installer darf keine steam_appid.txt anlegen")
	}
	if info, err := ReadLog(dest); err != nil || !info.Steam {
		t.Fatalf("Steam-Modus fehlt im Protokoll: %+v %v", info, err)
	}
	if _, err := RunUninstall(dest, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(acf); err == nil {
		t.Fatal("erzeugtes Manifest muss bei der Deinstallation entfernt werden")
	}
}

func TestNeverCreatesSteamAppIDFile(t *testing.T) {
	roots, _, ini := buildDiscs(t)
	writeIniAll(t, roots, strings.Replace(ini, "ShowTitle=1\n", "ShowTitle=1\nSteamAppID=777\n", 1))
	cfg, err := loadConfig(roots[0])
	if err != nil || cfg.IsSteam() || cfg.SteamAppID != "777" {
		t.Fatalf("Konfiguration: %+v %v", cfg, err)
	}
	dest := filepath.Join(t.TempDir(), "Spiel")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	t.Setenv("PRS_SPECIAL", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{SrcRoot: roots[0]}, nil)
	if _, err := os.Stat(filepath.Join(dest, "bin", "steam_appid.txt")); err == nil {
		t.Fatal("der Installer darf keine steam_appid.txt anlegen")
	}
	if info, _ := ReadLog(dest); info.Steam {
		t.Fatal("im Ordner-Modus darf kein Steam-Flag gesetzt sein")
	}
	// Eine vorhandene Datei gehört zum Spiel: Sie bleibt unverändert und wird bei der Deinstallation nur entfernt, wenn sie mitkopiert wurde.
	f := filepath.Join(dest, "bin", "steam_appid.txt")
	os.WriteFile(f, []byte("ORIG"), 0o644)
	RunUninstall(dest, func() bool { return false })
	if b, err := os.ReadFile(f); err != nil || string(b) != "ORIG" {
		t.Fatalf("fremde steam_appid.txt wurde verändert oder gelöscht: %q %v", b, err)
	}
}

func TestInvalidAppIDInFolderModeRejected(t *testing.T) {
	ini := "[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\nSteamAppID=../x\n[Disc]\nCount=1\n"
	if _, err := ParseConfig(strings.NewReader(ini)); err == nil {
		t.Fatal("ungültige AppID muss abgelehnt werden")
	}
}

func TestRunRedistReportsFailure(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	dest := filepath.Join(t.TempDir(), "G")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	if RunRedist(cfg, dest, nil) {
		t.Fatal("kein Fehler erwartet")
	}
	t.Setenv("PRS_REDIST_FAIL", "1")
	if !RunRedist(cfg, dest, nil) {
		t.Fatal("Fehler erwartet")
	}
}
