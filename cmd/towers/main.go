// Command towers renders a review sheet of all turret models and times their baking.
package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"

	"outpost/internal/art"
)

func main() {
	ids := art.TurretIDs
	angles := []float64{art.PortraitAngle, math.Pi / 2, 2.6, -1.9}
	cell := 52
	W := cell*len(angles)*art.TurretLevels + 20
	H := len(ids)*cell + 20 + 70
	c := art.NewCanvas(W, H)
	c.Rect(0, 0, W, H, art.Hex(0x7a4137))
	for r, id := range ids {
		for lv := 0; lv < art.TurretLevels; lv++ {
			for ai, a := range angles {
				s := &art.Scene{}
				s.Add(art.PadScene(0, 0).Parts...)
				b := art.TurretBaseScene(id, lv)
				h := art.TurretHeadScene(id, lv, a)
				lift(b, art.PadH)
				lift(h, art.PadH)
				s.Add(b.Parts...)
				s.Add(h.Parts...)
				sp := s.Bake(art.Opts{Inner: 3, Shadows: true}, true, 0, 0)
				x := 10 + (lv*len(angles)+ai)*cell + cell/2
				y := 10 + r*cell + cell*2/3
				drawShadow(c, sp, x, y)
				c.Stamp(sp.C, x-sp.OX, y-sp.OY)
			}
		}
	}
	x := 10
	for _, id := range ids {
		for lv := 0; lv < art.TurretLevels; lv++ {
			p := art.TurretPortrait(id, lv, 26, 26, true)
			c.Stamp(p.C, x, H-62)
			x += 30
		}
		x += 12
	}
	const k = 4
	out := image.NewRGBA(image.Rect(0, 0, W*k, H*k))
	for y := 0; y < H*k; y++ {
		for x := 0; x < W*k; x++ {
			out.Set(x, y, c.Img.At(x/k, y/k))
		}
	}
	f, _ := os.Create("/data/td/preview_towers.png")
	png.Encode(f, out)
	f.Close()

	// Bake timing (single thread) for the in-game sprite set.
	for _, id := range ids {
		t0 := time.Now()
		n := 64
		if art.TurretSpin(id) < 6 {
			n = 16
		}
		for lv := 0; lv < art.TurretLevels; lv++ {
			art.TurretBase(id, lv)
			for i := 0; i < n; i++ {
				art.TurretHead(id, lv, float64(i)/float64(n)*art.TurretSpin(id))
			}
		}
		fmt.Printf("%-9s %d heads/level: %v\n", id, n, time.Since(t0).Round(time.Millisecond))
	}
}

func lift(s *art.Scene, z float64) { art.Lift(s, z) }

func drawShadow(c *art.Canvas, sp *art.Sprite, x, y int) {
	if sp.Sh == nil {
		return
	}
	for j := 0; j < sp.Sh.H; j++ {
		for i := 0; i < sp.Sh.W; i++ {
			if sp.Sh.Get(i, j).A > 0 {
				c.Blend(x-sp.SX+i, y-sp.SY+j, art.Ink, 0.42)
			}
		}
	}
}
