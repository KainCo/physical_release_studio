// mkrsrc erzeugt das Disc-Symbol (autorun.ico) und eine Windows-Ressourcendatei (.syso) mit
// Symbol und Manifest, die Go beim Linken automatisch in die EXE einbettet.
//
//	go run ./mkrsrc -ico ../src/autorun.ico -syso ../launcher/rsrc_windows_amd64.syso
//
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

const manifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application><supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/></application>
  </compatibility>
</assembly>
`

func main() {
	icoPath := flag.String("ico", "autorun.ico", "Ausgabe: ICO-Datei")
	sysoPath := flag.String("syso", "", "Ausgabe: .syso-Ressourcendatei (leer = keine)")
	flag.Parse()
	sizes := []int{16, 32, 48, 256}
	var pngs [][]byte
	for _, s := range sizes {
		var b bytes.Buffer
		if err := png.Encode(&b, drawDisc(s)); err != nil {
			fail(err)
		}
		pngs = append(pngs, b.Bytes())
	}
	if err := os.WriteFile(*icoPath, buildICO(sizes, pngs), 0o644); err != nil {
		fail(err)
	}
	if *sysoPath != "" {
		if err := os.WriteFile(*sysoPath, buildSyso(sizes, pngs), 0o644); err != nil {
			fail(err)
		}
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "Fehler:", err); os.Exit(1) }

// ---------- Symbol ----------

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// drawDisc zeichnet eine silberne Disc mit Regenbogen-Schimmer und Loch (vierfach überabgetastet).
func drawDisc(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	c := float64(size) / 2
	rOut, rHub, rHole := c*0.92, c*0.34, c*0.16
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)/ss - c
					py := float64(y) + (float64(sy)+0.5)/ss - c
					d := math.Hypot(px, py)
					if d > rOut || d < rHole {
						continue
					}
					th := math.Atan2(py, px)
					// silbrige Grundfarbe mit zwei hellen Reflexen
					shine := 0.5 + 0.5*math.Cos(2*(th-0.7))
					base := 168 + 80*math.Pow(shine, 3)
					cr, cg, cb := base, base+4, base+12
					// feiner Regenbogen-Schimmer im Datenbereich
					if d > rHub {
						hue := math.Mod(th/(2*math.Pi)*2+d/c*0.8+2, 1)
						hr, hg, hb := hsv(hue)
						t := 0.22
						cr, cg, cb = cr*(1-t)+hr*t, cg*(1-t)+hg*t, cb*(1-t)+hb*t
					} else { // Nabe: hellgrau
						cr, cg, cb = 205+30*shine, 208+30*shine, 214+30*shine
					}
					// dunkler Außenrand und Nabenring
					if d > rOut-math.Max(1, c*0.05) || (d > rHub-math.Max(0.6, c*0.02) && d < rHub+math.Max(0.6, c*0.02)) {
						cr, cg, cb = cr*0.62, cg*0.62, cb*0.66
					}
					r += cr
					g += cg
					b += cb
					a++
				}
			}
			n := float64(ss * ss)
			if a > 0 {
				img.SetNRGBA(x, y, color.NRGBA{uint8(clamp01(r/a/255) * 255), uint8(clamp01(g/a/255) * 255), uint8(clamp01(b/a/255) * 255), uint8(a / n * 255)})
			}
		}
	}
	return img
}

func hsv(h float64) (r, g, b float64) {
	i := int(h * 6)
	f := h*6 - float64(i)
	q, t := 1-f, f
	switch i % 6 {
	case 0:
		return 255, t * 255, 0
	case 1:
		return q * 255, 255, 0
	case 2:
		return 0, 255, t * 255
	case 3:
		return 0, q * 255, 255
	case 4:
		return t * 255, 0, 255
	}
	return 255, 0, q * 255
}

// ---------- ICO ----------

func dim(s int) byte {
	if s >= 256 {
		return 0
	}
	return byte(s)
}

func buildICO(sizes []int, pngs [][]byte) []byte {
	var b bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&b, le, [3]uint16{0, 1, uint16(len(sizes))})
	off := uint32(6 + 16*len(sizes))
	for i, s := range sizes {
		b.Write([]byte{dim(s), dim(s), 0, 0})
		binary.Write(&b, le, uint16(1))
		binary.Write(&b, le, uint16(32))
		binary.Write(&b, le, uint32(len(pngs[i])))
		binary.Write(&b, le, off)
		off += uint32(len(pngs[i]))
	}
	for _, p := range pngs {
		b.Write(p)
	}
	return b.Bytes()
}

// ---------- Ressourcen (.syso, COFF für amd64) ----------

type res struct {
	typ, id uint16
	data    []byte
}

const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtManifest  = 24
)

func buildSyso(sizes []int, pngs [][]byte) []byte {
	le := binary.LittleEndian
	var rs []res
	for i := range pngs {
		rs = append(rs, res{rtIcon, uint16(i + 1), pngs[i]})
	}
	var grp bytes.Buffer
	binary.Write(&grp, le, [3]uint16{0, 1, uint16(len(sizes))})
	for i, s := range sizes {
		grp.Write([]byte{dim(s), dim(s), 0, 0})
		binary.Write(&grp, le, uint16(1))
		binary.Write(&grp, le, uint16(32))
		binary.Write(&grp, le, uint32(len(pngs[i])))
		binary.Write(&grp, le, uint16(i+1))
	}
	rs = append(rs, res{rtGroupIcon, 1, grp.Bytes()})
	rs = append(rs, res{rtManifest, 1, []byte(manifest)})
	sort.Slice(rs, func(a, b int) bool {
		if rs[a].typ != rs[b].typ {
			return rs[a].typ < rs[b].typ
		}
		return rs[a].id < rs[b].id
	})
	var types []uint16
	byType := map[uint16][]int{}
	for i, r := range rs {
		if _, ok := byType[r.typ]; !ok {
			types = append(types, r.typ)
		}
		byType[r.typ] = append(byType[r.typ], i)
	}

	// Aufbau: Wurzel | Typ-Verzeichnisse | Sprach-Verzeichnisse | Datenbeschreibungen | Daten
	rootSize := 16 + 8*len(types)
	typeOff := map[uint16]int{}
	cur := rootSize
	for _, t := range types {
		typeOff[t] = cur
		cur += 16 + 8*len(byType[t])
	}
	langOff := make([]int, len(rs))
	for i := range rs {
		langOff[i] = cur
		cur += 16 + 8
	}
	entryOff := make([]int, len(rs))
	for i := range rs {
		entryOff[i] = cur
		cur += 16
	}
	dataOff := make([]int, len(rs))
	for i := range rs {
		cur = (cur + 7) &^ 7
		dataOff[i] = cur
		cur += len(rs[i].data)
	}
	total := (cur + 7) &^ 7
	sec := make([]byte, total)
	put16 := func(o int, v uint16) { le.PutUint16(sec[o:], v) }
	put32 := func(o int, v uint32) { le.PutUint32(sec[o:], v) }
	dir := func(o, nId int) { put16(o+14, uint16(nId)) } // NumberOfIdEntries

	dir(0, len(types))
	for i, t := range types {
		e := 16 + 8*i
		put32(e, uint32(t))
		put32(e+4, 0x80000000|uint32(typeOff[t]))
	}
	for _, t := range types {
		o := typeOff[t]
		dir(o, len(byType[t]))
		for j, idx := range byType[t] {
			e := o + 16 + 8*j
			put32(e, uint32(rs[idx].id))
			put32(e+4, 0x80000000|uint32(langOff[idx]))
		}
	}
	var relocs []uint32
	for i := range rs {
		o := langOff[i]
		dir(o, 1)
		put32(o+16, 0x0409) // Sprache: Englisch (USA)
		put32(o+20, uint32(entryOff[i]))
		e := entryOff[i]
		put32(e, uint32(dataOff[i])) // RVA: Offset im Abschnitt; der Linker addiert die Abschnittsadresse
		put32(e+4, uint32(len(rs[i].data)))
		copy(sec[dataOff[i]:], rs[i].data)
		relocs = append(relocs, uint32(e))
	}

	var out bytes.Buffer
	symOff := 20 + 40 + total + 10*len(relocs)
	binary.Write(&out, le, uint16(0x8664)) // IMAGE_FILE_MACHINE_AMD64
	binary.Write(&out, le, uint16(1))
	binary.Write(&out, le, uint32(0))
	binary.Write(&out, le, uint32(symOff))
	binary.Write(&out, le, uint32(1))
	binary.Write(&out, le, uint16(0))
	binary.Write(&out, le, uint16(0))
	// Abschnittskopf
	out.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(&out, le, uint32(0))
	binary.Write(&out, le, uint32(0))
	binary.Write(&out, le, uint32(total))
	binary.Write(&out, le, uint32(20+40))
	binary.Write(&out, le, uint32(20+40+total))
	binary.Write(&out, le, uint32(0))
	binary.Write(&out, le, uint16(len(relocs)))
	binary.Write(&out, le, uint16(0))
	binary.Write(&out, le, uint32(0x40000040)) // initialisierte Daten, lesbar
	out.Write(sec)
	for _, r := range relocs {
		binary.Write(&out, le, r)
		binary.Write(&out, le, uint32(0))      // Symbol 0
		binary.Write(&out, le, uint16(0x0003)) // IMAGE_REL_AMD64_ADDR32NB
	}
	out.Write([]byte("$R000000"))
	binary.Write(&out, le, uint32(0))
	binary.Write(&out, le, uint16(1)) // Abschnitt 1
	binary.Write(&out, le, uint16(0))
	out.Write([]byte{3, 0}) // statisch, keine Zusatzeinträge
	binary.Write(&out, le, uint32(4))
	return out.Bytes()
}
