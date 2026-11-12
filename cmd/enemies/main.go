// Command enemies renders a review sheet of all enemy models.
package main

import (
	"image"
	"image/png"
	"math"
	"os"

	"outpost/internal/art"
)

func main() {
	ids := []string{"crawler", "runner", "swarmling", "walker", "drone", "queen", "beetle", "bloater", "stalker", "kraken"}
	frames := map[string]int{"crawler": 6, "runner": 6, "swarmling": 4, "walker": 8, "drone": 4, "queen": 8, "beetle": 6, "bloater": 6, "stalker": 6, "kraken": 8}
	cell := 60
	W := cell*8 + 20
	H := len(ids)*cell*2 + 20 + 80
	c := art.NewCanvas(W, H)
	c.Rect(0, 0, W, H, art.Hex(0x7a4137))
	for r, id := range ids {
		n := frames[id]
		for i := 0; i < 8; i++ {
			sp := art.EnemySprite(id, float64(i)/8*2*math.Pi, 0, n)
			x, y := 10+i*cell+cell/2, 10+r*2*cell+cell*2/3
			draw(c, sp, x, y, id == "drone")
		}
		for f := 0; f < n && f < 8; f++ {
			sp := art.EnemySprite(id, art.PortraitHeading, f, n)
			x, y := 10+f*cell+cell/2, 10+(r*2+1)*cell+cell*2/3
			draw(c, sp, x, y, id == "drone")
		}
	}
	// Portraits (icon size and card size).
	x := 10
	for _, id := range ids {
		p := art.EnemyPortrait(id, 18, 15)
		c.Stamp(p.C, x, H-70)
		q := art.EnemyPortrait(id, 44, 34)
		c.Stamp(q.C, x+22, H-50)
		x += 75
	}
	out := image.NewRGBA(image.Rect(0, 0, W*3, H*3))
	for y := 0; y < H*3; y++ {
		for x := 0; x < W*3; x++ {
			out.Set(x, y, c.Img.At(x/3, y/3))
		}
	}
	f, _ := os.Create("/data/td/preview_enemies.png")
	png.Encode(f, out)
	f.Close()
}

func draw(c *art.Canvas, sp *art.Sprite, x, y int, fly bool) {
	lift := 0.0
	if fly {
		lift = 14
	}
	if sp.Sh != nil {
		dx := int(math.Round(lift * 0.861))
		dy := int(math.Round(lift * 0.444 * art.GY))
		for j := 0; j < sp.Sh.H; j++ {
			for i := 0; i < sp.Sh.W; i++ {
				if sp.Sh.Get(i, j).A > 0 {
					c.Blend(x-sp.SX+i+dx, y-sp.SY+j+dy, art.Ink, 0.42)
				}
			}
		}
	}
	c.Stamp(sp.C, x-sp.OX, y-sp.OY-int(lift*art.HZ))
}
