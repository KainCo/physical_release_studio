//go:build ignore

// mkrsrc erzeugt das Disc-Icon (icon.ico, icon_preview.png) und die
// Windows-Ressourcendatei rsrc_windows_amd64.syso (Icon in der EXE).
// Aufruf: go run tools/mkrsrc/main.go   (im Launcher-Ordner)
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"
)

// render zeichnet die Disc in beliebiger Groesse (4x Supersampling).
func render(size int) *image.NRGBA {
	const ss = 4
	n := size * ss
	big := image.NewNRGBA(image.Rect(0, 0, n, n))
	c := float64(n) / 2
	R := c * 0.94
	hole := R * 0.17
	ring := R * 0.30
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := float64(x)+0.5-c, float64(y)+0.5-c
			d := math.Hypot(dx, dy)
			if d > R || d < hole {
				continue
			}
			ang := math.Atan2(dy, dx)
			// Regenbogen-aehnlicher Schimmer auf der Datenflaeche
			base := 0.74 + 0.16*math.Sin(ang*2+1.0)
			r, g, b := base, base+0.03, base+0.08
			if d > ring {
				t := (d - ring) / (R - ring)
				h := ang*3 + t*2
				r += 0.10 * math.Sin(h)
				g += 0.10 * math.Sin(h+2.1)
				b += 0.10 * math.Sin(h+4.2)
			} else {
				r, g, b = 0.86, 0.88, 0.92 // Spiegelring innen
			}
			if d > R*0.965 { // Rand
				r, g, b = 0.45, 0.50, 0.58
			}
			if d < hole*1.55 && d >= hole { // Nabenring
				r, g, b = 0.93, 0.94, 0.97
			}
			cl := func(v float64) uint8 { return uint8(math.Max(0, math.Min(1, v)) * 255) }
			big.SetNRGBA(x, y, color.NRGBA{cl(r), cl(g), cl(b), 255})
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var sr, sg, sb, sa int
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					p := big.NRGBAAt(x*ss+i, y*ss+j)
					sr += int(p.R) * int(p.A)
					sg += int(p.G) * int(p.A)
					sb += int(p.B) * int(p.A)
					sa += int(p.A)
				}
			}
			if sa == 0 {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(sr / sa), uint8(sg / sa), uint8(sb / sa), uint8(sa / (ss * ss))})
		}
	}
	return out
}

func pngBytes(im image.Image) []byte {
	var b bytes.Buffer
	png.Encode(&b, im)
	return b.Bytes()
}

// dib erzeugt BMP-Daten (32 bit + AND-Maske) fuer kleine Icons.
func dib(im *image.NRGBA) []byte {
	s := im.Bounds().Dx()
	var b bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&b, le, uint32(40))
	binary.Write(&b, le, int32(s))
	binary.Write(&b, le, int32(s*2))
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, uint16(32))
	binary.Write(&b, le, [6]uint32{0, uint32(s * s * 4), 0, 0, 0, 0})
	for y := s - 1; y >= 0; y-- {
		for x := 0; x < s; x++ {
			p := im.NRGBAAt(x, y)
			b.Write([]byte{p.B, p.G, p.R, p.A})
		}
	}
	rb := ((s + 31) / 32) * 4
	b.Write(make([]byte, rb*s))
	return b.Bytes()
}

type icon struct {
	size int
	data []byte
}

func main() {
	var icons []icon
	for _, s := range []int{16, 24, 32, 48, 64, 128, 256} {
		im := render(s)
		if s <= 48 {
			icons = append(icons, icon{s, dib(im)})
		} else {
			icons = append(icons, icon{s, pngBytes(im)})
		}
	}
	os.WriteFile("icon_preview.png", pngBytes(render(256)), 0o644)

	le := binary.LittleEndian
	// .ico
	var ico bytes.Buffer
	binary.Write(&ico, le, [3]uint16{0, 1, uint16(len(icons))})
	off := 6 + 16*len(icons)
	for _, ic := range icons {
		wh := byte(ic.size)
		if ic.size >= 256 {
			wh = 0
		}
		ico.Write([]byte{wh, wh, 0, 0})
		binary.Write(&ico, le, [2]uint16{1, 32})
		binary.Write(&ico, le, uint32(len(ic.data)))
		binary.Write(&ico, le, uint32(off))
		off += len(ic.data)
	}
	for _, ic := range icons {
		ico.Write(ic.data)
	}
	os.WriteFile("icon.ico", ico.Bytes(), 0o644)

	// GRPICONDIR
	var grp bytes.Buffer
	binary.Write(&grp, le, [3]uint16{0, 1, uint16(len(icons))})
	for i, ic := range icons {
		wh := byte(ic.size)
		if ic.size >= 256 {
			wh = 0
		}
		grp.Write([]byte{wh, wh, 0, 0})
		binary.Write(&grp, le, [2]uint16{1, 32})
		binary.Write(&grp, le, uint32(len(ic.data)))
		binary.Write(&grp, le, uint16(i+1))
	}

	// Ressourcen: type -> id -> data
	type res struct {
		typ, id uint32
		data    []byte
	}
	var rs []res
	for i, ic := range icons {
		rs = append(rs, res{3, uint32(i + 1), ic.data})
	}
	rs = append(rs, res{14, 1, grp.Bytes()})
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].typ != rs[j].typ {
			return rs[i].typ < rs[j].typ
		}
		return rs[i].id < rs[j].id
	})
	var types []uint32
	byType := map[uint32][]int{}
	for i, r := range rs {
		if _, ok := byType[r.typ]; !ok {
			types = append(types, r.typ)
		}
		byType[r.typ] = append(byType[r.typ], i)
	}

	// Offsets berechnen
	rootSize := 16 + 8*len(types)
	pos := rootSize
	typeDirOff := map[uint32]int{}
	for _, t := range types {
		typeDirOff[t] = pos
		pos += 16 + 8*len(byType[t])
	}
	langDirOff := make([]int, len(rs))
	for i := range rs {
		langDirOff[i] = pos
		pos += 16 + 8
	}
	dataEntOff := make([]int, len(rs))
	for i := range rs {
		dataEntOff[i] = pos
		pos += 16
	}
	rawOff := make([]int, len(rs))
	for i, r := range rs {
		rawOff[i] = pos
		pos += (len(r.data) + 3) &^ 3
	}
	sec := make([]byte, pos)
	put32 := func(o int, v uint32) { le.PutUint32(sec[o:], v) }
	put16 := func(o int, v uint16) { le.PutUint16(sec[o:], v) }
	dir := func(o, nid int) { put16(o+14, uint16(nid)) }
	dir(0, len(types))
	for k, t := range types {
		put32(16+8*k, t)
		put32(16+8*k+4, 0x80000000|uint32(typeDirOff[t]))
		d := typeDirOff[t]
		dir(d, len(byType[t]))
		for m, idx := range byType[t] {
			put32(d+16+8*m, rs[idx].id)
			put32(d+16+8*m+4, 0x80000000|uint32(langDirOff[idx]))
		}
	}
	var relocs []int
	for i, r := range rs {
		d := langDirOff[i]
		dir(d, 1)
		put32(d+16, 0x0409)
		put32(d+20, uint32(dataEntOff[i]))
		put32(dataEntOff[i], uint32(rawOff[i])) // + RVA per Relocation
		put32(dataEntOff[i]+4, uint32(len(r.data)))
		copy(sec[rawOff[i]:], r.data)
		relocs = append(relocs, dataEntOff[i])
	}

	// COFF-Objekt
	var o bytes.Buffer
	relOff := 20 + 40 + len(sec)
	symOff := relOff + 10*len(relocs)
	binary.Write(&o, le, uint16(0x8664))
	binary.Write(&o, le, uint16(1))
	binary.Write(&o, le, uint32(0))
	binary.Write(&o, le, uint32(symOff))
	binary.Write(&o, le, uint32(1))
	binary.Write(&o, le, uint16(0))
	binary.Write(&o, le, uint16(0))
	o.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(&o, le, [2]uint32{0, uint32(len(sec))})
	binary.Write(&o, le, uint32(len(sec)))
	binary.Write(&o, le, uint32(60))
	binary.Write(&o, le, uint32(relOff))
	binary.Write(&o, le, uint32(0))
	binary.Write(&o, le, uint16(len(relocs)))
	binary.Write(&o, le, uint16(0))
	binary.Write(&o, le, uint32(0x40000040))
	o.Write(sec)
	for _, r := range relocs {
		binary.Write(&o, le, uint32(r))
		binary.Write(&o, le, uint32(0))
		binary.Write(&o, le, uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
	}
	o.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(&o, le, uint32(0))
	binary.Write(&o, le, uint16(1))
	binary.Write(&o, le, uint16(0))
	o.Write([]byte{3, 0})
	binary.Write(&o, le, uint32(4)) // String-Tabelle
	os.WriteFile("rsrc_windows_amd64.syso", o.Bytes(), 0o644)
	_ = draw.Src
}
