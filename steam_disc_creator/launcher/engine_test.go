// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildDiscs erzeugt ein kleines Testpaket auf 3 "Discs" (Ordner) und liefert Stämme und Originaldaten.
func buildDiscs(t *testing.T) (roots []string, orig map[string][]byte, ini string) {
	t.Helper()
	base := t.TempDir()
	for i := 1; i <= 3; i++ {
		d := filepath.Join(base, fmt.Sprintf("Disc%d", i))
		os.MkdirAll(filepath.Join(d, "Game"), 0o755)
		os.WriteFile(filepath.Join(d, discMarkerName(i)),
			[]byte(fmt.Sprintf("PackageID=abcdef0123456789\nDisc=%d\nCount=3\n", i)), 0o644)
		roots = append(roots, d)
	}
	orig = map[string][]byte{
		"bin\\game.exe":         bytes.Repeat([]byte("EXE"), 5000),
		"data\\readme.txt":      []byte("hallo welt"),
		"data\\big.bin":         bytes.Repeat([]byte("0123456789abcdef"), 40000), // 640000 Bytes, geteilt
		"_CommonRedist\\vc.exe": []byte("redist"),
		"empty.dat":             {},
	}
	crc := func(b []byte) string { return fmt.Sprintf("%08x", crc32.ChecksumIEEE(b)) }
	write := func(disc int, rel string, b []byte) {
		p := filepath.Join(roots[disc-1], strings.ReplaceAll(rel, "\\", string(filepath.Separator)))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, b, 0o644)
	}
	var sb strings.Builder
	sb.WriteString("[Game]\nTitle=Test Spiel\nExe=bin\\game.exe\nPackageID=abcdef0123456789\nShowTitle=1\n[Disc]\nCount=3\n[Files]\n")
	exe := orig["bin\\game.exe"]
	write(1, "Game\\bin\\game.exe", exe)
	fmt.Fprintf(&sb, "F=%d|bin\\game.exe\nS=1|0|%d|%s|Game\\bin\\game.exe\n", len(exe), len(exe), crc(exe))
	rd := orig["data\\readme.txt"]
	write(1, "Game\\data\\readme.txt", rd)
	fmt.Fprintf(&sb, "F=%d|data\\readme.txt\nS=1|0|%d|%s|Game\\data\\readme.txt\n", len(rd), len(rd), crc(rd))
	big := orig["data\\big.bin"]
	p1, p2, p3 := big[:300000], big[300000:500000], big[500000:]
	write(2, "Game\\data\\big.bin.part1", p1)
	write(3, "Game\\data\\big.bin.part2", p2)
	write(3, "Game\\data\\big.bin.part3", p3)
	fmt.Fprintf(&sb, "F=%d|data\\big.bin\n", len(big))
	fmt.Fprintf(&sb, "S=2|0|%d|%s|Game\\data\\big.bin.part1\n", len(p1), crc(p1))
	fmt.Fprintf(&sb, "S=3|300000|%d|%s|Game\\data\\big.bin.part2\n", len(p2), crc(p2))
	fmt.Fprintf(&sb, "S=3|500000|%d|%s|Game\\data\\big.bin.part3\n", len(p3), crc(p3))
	rv := orig["_CommonRedist\\vc.exe"]
	write(1, "Game\\_CommonRedist\\vc.exe", rv)
	fmt.Fprintf(&sb, "F=%d|_CommonRedist\\vc.exe\nS=1|0|%d|%s|Game\\_CommonRedist\\vc.exe\n", len(rv), len(rv), crc(rv))
	sb.WriteString("F=0|empty.dat\n")
	sb.WriteString("[Redist]\nR=_CommonRedist\\vc.exe|/quiet /norestart|Visual C++\n")
	ini = sb.String()
	for _, r := range roots {
		os.WriteFile(filepath.Join(r, "launcher.ini"), []byte(ini), 0o644)
	}
	return
}

func hooksFor(cfg *Config, roots []string) Hooks {
	known := map[int]string{}
	for i, r := range roots {
		known[i+1] = r
	}
	return Hooks{
		NeedDisc: func(n int) (string, error) {
			if r, ok := FindDisc(cfg, n, filepath.Dir(roots[0]), known); ok {
				return r, nil
			}
			return "", fmt.Errorf("Disc %d fehlt", n)
		},
	}
}

func TestInstallAndUninstall(t *testing.T) {
	roots, orig, _ := buildDiscs(t)
	cfg, err := loadConfig(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "Games", "Test Spiel")
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	for rel, want := range orig {
		got, err := os.ReadFile(filepath.Join(dest, strings.ReplaceAll(rel, "\\", string(filepath.Separator))))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("%s: Inhalt weicht ab (%v)", rel, err)
		}
	}
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	t.Setenv("PRS_SPECIAL", t.TempDir())
	if PostInstall(cfg, dest, InstallOpts{Desktop: true, StartMenu: true, Redist: true}, nil) {
		t.Fatal("Redist sollte nicht fehlschlagen")
	}
	if d, ok := regLookup(cfg.PackageID); !ok || d != dest {
		t.Fatalf("Registrierung fehlt: %q %v", d, ok)
	}
	if _, err := os.Stat(filepath.Join(dest, "uninstall.exe")); err != nil {
		t.Fatal("uninstall.exe fehlt")
	}
	// Zusatzdatei (Spielstand)
	os.WriteFile(filepath.Join(dest, "save.dat"), []byte("x"), 0o644)
	info, err := ReadLog(dest)
	if err != nil || !info.CreatedDest || info.Title != "Test Spiel" {
		t.Fatalf("Protokoll: %+v %v", info, err)
	}
	if !HasExtraFiles(dest, info) {
		t.Fatal("Zusatzdatei nicht erkannt")
	}
	// Deinstallation ohne Zusatzdateien zu löschen
	if _, err := RunUninstall(dest, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "save.dat")); err != nil {
		t.Fatal("Spielstand wurde fälschlich gelöscht")
	}
	if _, err := os.Stat(filepath.Join(dest, "bin", "game.exe")); err == nil {
		t.Fatal("game.exe sollte weg sein")
	}
	if _, ok := regLookup(cfg.PackageID); ok {
		t.Fatal("Registrierung sollte weg sein")
	}
}

func TestUninstallRemoveExtra(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	dest := filepath.Join(t.TempDir(), "Neu")
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{}, nil)
	os.WriteFile(filepath.Join(dest, "save.dat"), []byte("x"), 0o644)
	if _, err := RunUninstall(dest, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("Ordner sollte vollständig entfernt sein")
	}
}

func TestExistingFolderIsNeverWiped(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	dest := t.TempDir() // existiert bereits => nicht vom Installer angelegt
	os.WriteFile(filepath.Join(dest, "fremd.txt"), []byte("fremd"), 0o644)
	t.Setenv("PRS_STATE_DIR", t.TempDir())
	if err := Install(context.Background(), cfg, dest, hooksFor(cfg, roots)); err != nil {
		t.Fatal(err)
	}
	PostInstall(cfg, dest, InstallOpts{}, nil)
	RunUninstall(dest, func() bool { return true })
	if _, err := os.Stat(filepath.Join(dest, "fremd.txt")); err != nil {
		t.Fatal("fremde Datei wurde gelöscht")
	}
}

func TestCorruptSegmentDetectedAndRetry(t *testing.T) {
	roots, orig, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	part := filepath.Join(roots[2], "Game", "data", "big.bin.part2")
	good, _ := os.ReadFile(part)
	bad := append([]byte(nil), good...)
	bad[10] ^= 0xff
	os.WriteFile(part, bad, 0o644)

	dest := filepath.Join(t.TempDir(), "x")
	h := hooksFor(cfg, roots)
	err := Install(context.Background(), cfg, dest, h)
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("erwartet ErrCorrupt, bekam %v", err)
	}
	// mit Retry: Disc wird "gereinigt" (Datei repariert) und der zweite Versuch gelingt
	h.Retry = func(file string, err error) bool {
		os.WriteFile(part, good, 0o644)
		return true
	}
	dest2 := filepath.Join(t.TempDir(), "y")
	if err := Install(context.Background(), cfg, dest2, h); err != nil {
		t.Fatalf("Retry sollte gelingen: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dest2, "data", "big.bin"))
	if !bytes.Equal(got, orig["data\\big.bin"]) {
		t.Fatal("zusammengeführte Datei stimmt nicht")
	}
}

func TestCancel(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Install(ctx, cfg, filepath.Join(t.TempDir(), "z"), hooksFor(cfg, roots))
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("erwartet ErrCancelled, bekam %v", err)
	}
}

func TestMissingDisc(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	os.Remove(filepath.Join(roots[2], discMarkerName(3)))
	err := Install(context.Background(), cfg, filepath.Join(t.TempDir(), "m"), hooksFor(cfg, roots[:3]))
	if err == nil || !strings.Contains(err.Error(), "Disc 3") {
		t.Fatalf("erwartet Fehler zu Disc 3, bekam %v", err)
	}
}

func TestWrongPackageDiscIsRejected(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	os.WriteFile(filepath.Join(roots[1], discMarkerName(2)), []byte("PackageID=ffffffffffffffff\n"), 0o644)
	if checkDisc(roots[1], cfg, 2) {
		t.Fatal("Disc eines anderen Pakets darf nicht akzeptiert werden")
	}
}

func TestPathTraversalRejected(t *testing.T) {
	for _, bad := range []string{"..\\evil.exe", "a\\..\\..\\b", "C:\\x", "\\abs", "a\\\\b", "con:", "a|b", "x\\"} {
		if _, err := safeRel(bad); err == nil {
			t.Errorf("%q hätte abgelehnt werden müssen", bad)
		}
	}
	ini := "[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\n[Disc]\nCount=1\n[Files]\nF=1|..\\evil\nS=1|0|1|00000000|Game\\evil\n"
	if _, err := ParseConfig(strings.NewReader(ini)); err == nil {
		t.Fatal("Pfad mit .. muss abgelehnt werden")
	}
}

func TestConfigValidation(t *testing.T) {
	base := "[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\n[Disc]\nCount=2\n[Files]\n"
	cases := map[string]string{
		"Lücke":       base + "F=10|a\nS=1|0|4|00000000|Game\\a.1\nS=2|5|5|00000000|Game\\a.2\n",
		"Summe":       base + "F=10|a\nS=1|0|4|00000000|Game\\a\n",
		"Disc":        base + "F=4|a\nS=3|0|4|00000000|Game\\a\n",
		"ohneTeile":   base + "F=4|a\n",
		"S ohne F":    base + "S=1|0|4|00000000|Game\\a\n",
		"BOM+PkgID":   "\xef\xbb\xbf[Game]\nTitle=T\nExe=g.exe\nPackageID=XYZ\n[Disc]\nCount=1\n",
		"DiscCount 0": "[Game]\nTitle=T\nExe=g.exe\nPackageID=abcdef0123456789\n[Disc]\nCount=0\n",
	}
	for name, ini := range cases {
		if _, err := ParseConfig(strings.NewReader(ini)); err == nil {
			t.Errorf("%s: Fehler erwartet", name)
		}
	}
	ok := "\xef\xbb\xbf[Game]\nTitle=T\nExe=g.exe\nPackageID=ABCDEF0123456789\n[Disc]\nCount=1\n[Files]\nF=0|leer\n"
	if _, err := ParseConfig(strings.NewReader(ok)); err != nil {
		t.Errorf("gültige Datei abgelehnt: %v", err)
	}
}

func TestNoNetworkImports(t *testing.T) {
	// Zusicherung der Datenschutzerklärung: der Launcher baut keine Internetverbindung auf.
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, _ := os.ReadFile(f)
		for _, bad := range []string{"\"net/http\"", "\"net\"", "\"net/url\"", "WinHttp", "wininet", "ws2_32", "URLDownloadToFile"} {
			if bytes.Contains(b, []byte(bad)) {
				t.Errorf("%s enthält Netzwerkbezug %s", f, bad)
			}
		}
	}
}

func TestFindDiscSiblingFolders(t *testing.T) {
	roots, _, _ := buildDiscs(t)
	cfg, _ := loadConfig(roots[0])
	// Start aus Disc1 – Disc2/Disc3 liegen als Geschwisterordner
	for n := 1; n <= 3; n++ {
		r, ok := FindDisc(cfg, n, roots[0], map[int]string{})
		if !ok || r != roots[n-1] {
			t.Fatalf("Disc %d: %q %v", n, r, ok)
		}
	}
}
