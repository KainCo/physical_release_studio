// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const extrasIni = "[Game]\r\nTitle=T\r\nExe=g.exe\r\nPackageID=0123456789abcdef\r\n[Disc]\r\nCount=1\r\n[Files]\r\nF=1|g.exe\r\nS=1|0|1|00000000|Game\\g.exe\r\n" +
	"[Extras]\r\nX=Extras\\Manual.pdf|Handbuch\r\nX=Extras\\Missing.txt|Fehlt\r\nX=Extras\\x.txt|\r\nX=..\\evil.exe|Boese\r\n" +
	"X=Extras\\a|A\r\nX=Extras\\b|B\r\nX=Extras\\c|C\r\nX=Extras\\d|D\r\n"

func TestExtrasParseAndAvailable(t *testing.T) {
	cfg, err := ParseConfig(strings.NewReader(extrasIni))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Extras) > maxExtras {
		t.Fatalf("mehr als %d Extras: %d", maxExtras, len(cfg.Extras))
	}
	if cfg.Extras[0].Label != "Handbuch" || cfg.Extras[0].Path != `Extras\Manual.pdf` {
		t.Fatalf("falsch: %+v", cfg.Extras[0])
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "Extras"), 0o755)
	os.WriteFile(filepath.Join(root, "Extras", "Manual.pdf"), []byte("x"), 0o644)
	got := AvailableExtras(cfg, root)
	if len(got) != 1 || got[0].Label != "Handbuch" {
		t.Fatalf("erwartet nur das vorhandene Extra, bekam %+v", got)
	}
}
