// Command wyrm renders a review sheet of the Ice Wyrm: the head, every body
// part, a whole wyrm along a curve and the portraits.
package main

import (
	"image"
	"image/png"
	"math"
	"os"

	"outpost/internal/art"
)

func main() {
	cell := 64
	W, H := cell*8+20, cell*7+40
	c := art.NewCanvas(W, H)
	c.Rect(0, 0, W, H, art.Hex(0x8e9ab4))
	c.Rect(0, cell*5, W, H-cell*5, art.Hex(0x616d8c))
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		stamp(c, art.EnemySprite("wyrm", a, 0, 8), 10+i*cell+cell/2, cell*2/3+10)
		stamp(c, art.EnemySprite("wyrm", 0.5, i, 8), 10+i*cell+cell/2, cell+cell*2/3+10)
	}
	for k := 0; k < art.WyrmKinds; k++ {
		for i := 0; i < 8; i += 2 {
			a := float64(i) / 8 * 2 * math.Pi
			x := 10 + (k*4+i/2)*cell/2 + cell/4
			stamp(c, art.WyrmPart(k, a), x, 2*cell+cell*2/3+10)
		}
	}
	// A whole wyrm along a curve, the way the game lays it on the road.
	for v := 0; v < 4; v++ {
		heading := []float64{0, 2.4, -math.Pi / 2, math.Pi / 2}[v]
		ox, oy := 10+cell*1+v*cell*2, 4*cell+cell/2
		rad := []float64{1.5, 0.9, 99, 99}[v] * art.CellPx
		pos := func(back float64) (float64, float64, float64) {
			d := back * art.CellPx
			a := heading - d/rad
			px := float64(ox) - math.Sin(heading)*rad + math.Sin(a)*rad
			py := float64(oy) + math.Cos(heading)*rad - math.Cos(a)*rad
			if rad > 1000 {
				px, py = float64(ox)-math.Cos(heading)*d, float64(oy)-math.Sin(heading)*d
			}
			return px, py, a
		}
		type item struct {
			sp *art.Sprite
			x  float64
			y  float64
		}
		var items []item
		for i := len(art.WyrmBody) - 1; i >= 0; i-- {
			b := art.WyrmBody[i]
			x, y, a := pos(b.Back)
			items = append(items, item{art.WyrmPart(b.Kind, a), x, y})
		}
		x, y, a := pos(0)
		items = append(items, item{art.EnemySprite("wyrm", a, 2, 8), x, y})
		for _, it := range items {
			shadow(c, it.sp, int(it.x), int(it.y*art.GY)/1)
		}
		for _, it := range items {
			c.Stamp(it.sp.C, int(it.x)-it.sp.OX, int(it.y*art.GY)-it.sp.OY)
		}
	}
	p := art.EnemyPortrait("wyrm", 44, 34)
	c.Stamp(p.C, 10, 6*cell+10)
	q := art.EnemyPortrait("wyrm", 18, 14)
	c.Stamp(q.C, 70, 6*cell+20)
	out := image.NewRGBA(image.Rect(0, 0, W*3, H*3))
	for y := 0; y < H*3; y++ {
		for x := 0; x < W*3; x++ {
			out.Set(x, y, c.Img.At(x/3, y/3))
		}
	}
	f, _ := os.Create(os.Args[1])
	png.Encode(f, out)
	f.Close()
}

func shadow(c *art.Canvas, sp *art.Sprite, x, y int) {
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

func stamp(c *art.Canvas, sp *art.Sprite, x, y int) {
	shadow(c, sp, x, y)
	c.Stamp(sp.C, x-sp.OX, y-sp.OY)
}
