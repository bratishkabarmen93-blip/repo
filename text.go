package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
)

// Seg is a coloured run of text.
type Seg struct {
	S   string
	Col color.RGBA
}

// pm returns c with alpha a as a premultiplied colour.
func pm(c color.RGBA, a float64) color.RGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

func (a *App) textWidth(s string, sc int) int {
	w := 0
	for _, r := range s {
		if r == ' ' {
			w += art.FontSpace
			continue
		}
		if g, ok := a.as.Glyphs[r]; ok {
			w += g.adv
		}
	}
	if w > 0 {
		w--
	}
	return w * sc
}

func (a *App) segsWidth(segs []Seg, sc int) int {
	s := ""
	for _, g := range segs {
		s += g.S
	}
	return a.textWidth(s, sc)
}

// drawText draws s with its top-left at (x, y) using integer scale sc.
func (a *App) drawText(dst *ebiten.Image, s string, x, y, sc int, col color.RGBA, alpha float64) int {
	return a.drawTextRaw(dst, s, x, y, sc, col, alpha, true)
}

func (a *App) drawTextRaw(dst *ebiten.Image, s string, x, y, sc int, col color.RGBA, alpha float64, shadow bool) int {
	if shadow {
		a.drawGlyphs(dst, s, x+sc, y+sc, sc, pm(art.Ink, alpha*0.85))
	}
	return a.drawGlyphs(dst, s, x, y, sc, pm(col, alpha))
}

func (a *App) drawGlyphs(dst *ebiten.Image, s string, x, y, sc int, c color.RGBA) int {
	op := &ebiten.DrawImageOptions{}
	for _, r := range s {
		if r == ' ' {
			x += art.FontSpace * sc
			continue
		}
		g, ok := a.as.Glyphs[r]
		if !ok {
			continue
		}
		op.GeoM.Reset()
		op.GeoM.Scale(float64(sc), float64(sc))
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(c)
		dst.DrawImage(g.img, op)
		x += g.adv * sc
	}
	return x
}

func (a *App) drawSegs(dst *ebiten.Image, segs []Seg, x, y, sc int, alpha float64) int {
	for _, g := range segs {
		x = a.drawText(dst, g.S, x, y, sc, g.Col, alpha)
	}
	return x
}

func (a *App) drawIcon(dst *ebiten.Image, name string, x, y, sc int, alpha float64) {
	img := a.as.Icons[name]
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(sc), float64(sc))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(img, op)
}

func fillRect(dst *ebiten.Image, x, y, w, h int, c color.RGBA) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

// panel draws a framed, semi-transparent box.
func (a *App) panel(dst *ebiten.Image, r Rect, alpha float64) {
	u := a.u
	fillRect(dst, r.X, r.Y, r.W, r.H, pm(art.Ink, 0.86*alpha))
	fillRect(dst, r.X, r.Y, r.W, u, pm(art.Slate, alpha))
	fillRect(dst, r.X, r.Y+r.H-u, r.W, u, pm(art.Navy, alpha))
	fillRect(dst, r.X, r.Y, u, r.H, pm(art.Navy, alpha))
	fillRect(dst, r.X+r.W-u, r.Y, u, r.H, pm(art.Navy, alpha))
}
