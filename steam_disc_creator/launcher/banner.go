// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"image"
	_ "image/jpeg" // Dekoder für banner.jpg
	_ "image/png"  // Dekoder für banner.png
	"math"
	"os"
)

// Banner ist ein fertig skaliertes Bild für die Oberfläche (32 Bit BGRA, Zeilen von unten nach oben).
type Banner struct {
	W, H int
	BGRA []byte
}

const (
	bannerMinH = 180
	bannerMaxH = 380
)

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// loadBanner liest ein Bild (JPEG/PNG) und skaliert es auf die Fensterbreite w.
func loadBanner(path string, w int) *Banner {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return renderBanner(img, w)
}

// renderBanner skaliert proportional auf die Breite w. Die Höhe wird auf 180–380 Pixel begrenzt:
// zu hohe Bilder werden mittig beschnitten, zu flache bekommen einen dunkelblauen Rand.
func renderBanner(img image.Image, w int) *Banner {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 || w <= 0 {
		return nil
	}
	dh := w * sh / sw
	boxH := clampInt(dh, bannerMinH, bannerMaxH)
	out := make([]byte, w*boxH*4)
	for i := 0; i < len(out); i += 4 { // Hintergrund: Navy (RGB 0,0,128)
		out[i], out[i+1], out[i+2], out[i+3] = 128, 0, 0, 255
	}
	off := (boxH - dh) / 2
	for y := 0; y < boxH; y++ {
		dy := y - off
		if dy < 0 || dy >= dh {
			continue
		}
		fy := (float64(dy)+0.5)*float64(sh)/float64(dh) - 0.5
		row := (boxH - 1 - y) * w * 4
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			r, g, bl := sampleBilinear(img, b, fx, fy)
			o := row + x*4
			out[o], out[o+1], out[o+2], out[o+3] = bl, g, r, 255
		}
	}
	return &Banner{W: w, H: boxH, BGRA: out}
}

func sampleBilinear(img image.Image, b image.Rectangle, fx, fy float64) (r, g, bl uint8) {
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	px := func(x, y int) (float64, float64, float64) {
		x = clampInt(x, 0, b.Dx()-1)
		y = clampInt(y, 0, b.Dy()-1)
		cr, cg, cb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
		return float64(cr >> 8), float64(cg >> 8), float64(cb >> 8)
	}
	r00, g00, b00 := px(x0, y0)
	r10, g10, b10 := px(x0+1, y0)
	r01, g01, b01 := px(x0, y0+1)
	r11, g11, b11 := px(x0+1, y0+1)
	mix := func(a, b, c, d float64) uint8 {
		top := a + (b-a)*tx
		bot := c + (d-c)*tx
		return uint8(top + (bot-top)*ty + 0.5)
	}
	return mix(r00, r10, r01, r11), mix(g00, g10, g01, g11), mix(b00, b10, b01, b11)
}
