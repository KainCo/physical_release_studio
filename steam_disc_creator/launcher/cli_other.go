//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Kommandozeilenmodus für Tests unter Linux/macOS:
//
//	launcher install <Zielordner> <Disc-Stammordner-1> [<Disc-Stammordner-2> ...]
//	launcher uninstall <Zielordner>
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Dieses Programm ist ein Windows-Launcher. Testmodus: install|uninstall ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "install":
		dest := os.Args[2]
		roots := os.Args[3:]
		if len(roots) == 0 {
			fmt.Println("Disc-Ordner fehlen")
			os.Exit(2)
		}
		cfg, err := loadConfig(roots[0])
		if err != nil {
			fmt.Println("Fehler:", err)
			os.Exit(1)
		}
		known := map[int]string{}
		for _, r := range roots {
			for n := 1; n <= cfg.DiscCount; n++ {
				if checkDisc(r, cfg, n) {
					known[n] = r
				}
			}
		}
		h := Hooks{
			NeedDisc: func(n int) (string, error) {
				if r, ok := FindDisc(cfg, n, filepath.Dir(roots[0]), known); ok {
					return r, nil
				}
				return "", fmt.Errorf("Disc %d nicht gefunden", n)
			},
			Progress: func(done, total int64, cur string) {},
			Retry:    func(file string, err error) bool { return false },
		}
		if err := Install(context.Background(), cfg, dest, h); err != nil {
			fmt.Println("Fehler:", err)
			os.Exit(1)
		}
		src := known[1] // wie srcRoot() in der Windows-Oberfläche: Stamm der Disc 1
		if src == "" {
			src = roots[0]
		}
		if PostInstall(cfg, dest, InstallOpts{Desktop: true, StartMenu: true, Redist: true, SrcRoot: src}, func(s string) { fmt.Println(s) }) {
			fmt.Println("Hinweis: Komponenten fehlgeschlagen")
		}
		fmt.Println("OK")
	case "uninstall":
		dest := os.Args[2]
		title, err := RunUninstall(dest, func() bool { return os.Getenv("PRS_DELETE_EXTRA") != "" })
		if err != nil {
			fmt.Println("Fehler:", err)
			os.Exit(1)
		}
		fmt.Println("deinstalliert:", title)
	default:
		os.Exit(2)
	}
}
