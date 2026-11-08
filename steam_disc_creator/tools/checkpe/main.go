// checkpe prüft eine Windows-EXE: Ressourcenabschnitt, Symbol und Manifest müssen vorhanden sein.
//
//	go run ./checkpe ../dist/launcher.exe
//
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Aufruf: checkpe <datei.exe>")
		os.Exit(2)
	}
	f, err := pe.Open(os.Args[1])
	if err != nil {
		fmt.Println("keine gültige PE-Datei:", err)
		os.Exit(1)
	}
	defer f.Close()
	var rs *pe.Section
	for _, s := range f.Sections {
		if s.Name == ".rsrc" {
			rs = s
		}
	}
	if rs == nil {
		fmt.Println("FEHLER: kein .rsrc-Abschnitt")
		os.Exit(1)
	}
	data, err := rs.Data()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		fmt.Println("FEHLER: keine 64-Bit-EXE")
		os.Exit(1)
	}
	le := binary.LittleEndian
	rva := rs.VirtualAddress
	found := map[uint32]int{}
	var walk func(off uint32, depth int, typ uint32)
	walk = func(off uint32, depth int, typ uint32) {
		n := int(le.Uint16(data[off+12:])) + int(le.Uint16(data[off+14:]))
		for i := 0; i < n; i++ {
			e := off + 16 + uint32(i)*8
			id, to := le.Uint32(data[e:]), le.Uint32(data[e+4:])
			if depth == 0 {
				typ = id
			}
			if to&0x80000000 != 0 {
				walk(to&0x7fffffff, depth+1, typ)
				continue
			}
			dRVA, size := le.Uint32(data[to:]), le.Uint32(data[to+4:])
			start := dRVA - rva
			if dRVA < rva || uint64(start)+uint64(size) > uint64(len(data)) {
				fmt.Printf("FEHLER: Ressource Typ %d außerhalb des Abschnitts (RVA %#x)\n", typ, dRVA)
				os.Exit(1)
			}
			found[typ]++
			if typ == 24 && !strings.Contains(string(data[start:start+size]), "asInvoker") {
				fmt.Println("FEHLER: Manifest ohne asInvoker")
				os.Exit(1)
			}
		}
	}
	walk(0, 0, 0)
	if found[3] == 0 || found[14] == 0 || found[24] == 0 {
		fmt.Printf("FEHLER: Ressourcen unvollständig: %v\n", found)
		os.Exit(1)
	}
	fmt.Printf("OK: %d Symbole, %d Symbolgruppe, %d Manifest, Subsystem %d, ImageBase %#x\n",
		found[3], found[14], found[24], oh.Subsystem, oh.ImageBase)
}
