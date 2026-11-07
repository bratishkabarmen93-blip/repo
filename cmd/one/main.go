// Command one renders one enemy type large on a given ground colour (art review).
package main

import (
	"image"
	"image/png"
	"math"
	"os"
	"strconv"

	"outpost/internal/art"
)

func main() {
	id := os.Args[1]
	bg, _ := strconv.ParseUint(os.Args[2], 16, 32)
	n, _ := strconv.Atoi(os.Args[3])
	cell := 44
	W, H := cell*8, cell*3+30
	c := art.NewCanvas(W, H)
	c.Rect(0, 0, W, H, art.Hex(uint32(bg)))
	for i := 0; i < 8; i++ {
		sp := art.EnemySprite(id, float64(i)/8*2*math.Pi, 0, n)
		stamp(c, sp, i*cell+cell/2, cell*2/3)
	}
	p := art.EnemyPortrait(id, 44, 34)
	c.Stamp(p.C, 4, cell+20)
	q := art.EnemyPortrait(id, 18, 14)
	c.Stamp(q.C, 60, cell+30)
	// Animation frames at one heading along the bottom row.
	for f := 0; f < n && f < 8; f++ {
		stamp(c, art.EnemySprite(id, 0.5, f, n), f*cell+cell/2, cell*2+cell/2+10)
	}
	out := image.NewRGBA(image.Rect(0, 0, W*4, H*4))
	for y := 0; y < H*4; y++ {
		for x := 0; x < W*4; x++ {
			out.Set(x, y, c.Img.At(x/4, y/4))
		}
	}
	f, _ := os.Create(os.Args[4])
	png.Encode(f, out)
	f.Close()
}

func stamp(c *art.Canvas, sp *art.Sprite, x, y int) {
	if sp.Sh != nil {
		for j := 0; j < sp.Sh.H; j++ {
			for i := 0; i < sp.Sh.W; i++ {
				if sp.Sh.Get(i, j).A > 0 {
					c.Blend(x-sp.SX+i, y-sp.SY+j, art.Ink, 0.42)
				}
			}
		}
	}
	c.Stamp(sp.C, x-sp.OX, y-sp.OY)
}
