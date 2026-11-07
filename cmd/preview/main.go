// Command preview renders the procedural art into PNG files for review.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"outpost/internal/art"
	"outpost/internal/game"
)

func scale(c *art.Canvas, s int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, c.W*s, c.H*s))
	for y := 0; y < c.H*s; y++ {
		for x := 0; x < c.W*s; x++ {
			out.Set(x, y, c.Img.At(x/s, y/s))
		}
	}
	return out
}

func save(name string, img image.Image) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// put stamps a sprite with its shadow at world point (x, y, z) into the scene canvas.
func put(dst, shadows *art.Canvas, oy int, sp *art.Sprite, x, y, z float64) {
	px, py := art.Project(x, y, z)
	ix, iy := int(math.Floor(px)), int(math.Floor(py))+oy
	if sp.Sh != nil && shadows != nil {
		gx, gy := art.Project(x, y, z)
		shadows.Stamp(sp.Sh, int(math.Floor(gx))-sp.SX, int(math.Floor(gy))+oy-sp.SY)
	}
	dst.Stamp(sp.C, ix-sp.OX, iy-sp.OY)
}

func main() {
	lvl, err := game.ParseLevel(game.Level1)
	if err != nil {
		panic(err)
	}
	const P = art.CellPx
	t0 := time.Now()
	ter, ti := art.Terrain(lvl)
	fmt.Println("terrain", time.Since(t0))
	t0 = time.Now()
	base := art.Base(false)
	var heads [3][8]*art.Sprite
	var bases [3]*art.Sprite
	art.Parallel(3*8, func(i int) { heads[i/8][i%8] = art.TurretHead("blaster", i/8, float64(i%8)/8*2*math.Pi) })
	for lv := 0; lv < 3; lv++ {
		bases[lv] = art.TurretBase("blaster", lv)
	}
	var crawl [8][4]*art.Sprite
	art.Parallel(32, func(i int) { crawl[i/4][i%4] = art.Crawler(float64(i/4)/8*2*math.Pi, i%4, 4) })
	radar := art.Radar(0.7)
	fmt.Println("sprites", time.Since(t0))

	scene := art.NewCanvas(ter.W, ter.H)
	scene.Stamp(ter, 0, 0)
	shadow := art.NewCanvas(ter.W, ter.H)
	oy := ti.OY
	// Towers on pads.
	var objs []func()
	for i, s := range lvl.Slots[:7] {
		c := s.Center()
		lv := i % 3
		x, y := c.X*P, c.Y*P
		objs = append(objs, func() {
			put(scene, shadow, oy, bases[lv], x, y, art.PadH)
			put(scene, shadow, oy, heads[lv][i%8], x, y, art.PadH)
		})
	}
	// Enemies along the road.
	for i := 0; i < 10; i++ {
		p, dir := lvl.Path.At(3 + float64(i)*3.1)
		x, y := p.X*P, p.Y*P
		hi := int(math.Round(dir.Angle()/(2*math.Pi)*8)+8) % 8
		f := i % 4
		objs = append(objs, func() { put(scene, shadow, oy, crawl[hi][f], x, y, ti.Height(x, y)) })
	}
	bc := lvl.BaseCenter()
	objs = append(objs, func() {
		put(scene, nil, oy, base, bc.X*P, bc.Y*P, 0)
		put(scene, nil, oy, radar, bc.X*P+art.RadarPos.X, bc.Y*P+art.RadarPos.Y, art.RadarPos.Z)
	})
	// First pass collects shadows, then they are blended, then objects are drawn.
	for _, f := range objs {
		f()
	}
	scene2 := art.NewCanvas(ter.W, ter.H)
	scene2.Stamp(ter, 0, 0)
	for y := 0; y < shadow.H; y++ {
		for x := 0; x < shadow.W; x++ {
			if shadow.Get(x, y).A > 0 && scene2.Get(x, y).A > 0 {
				scene2.Blend(x, y, art.Ink, 0.38)
			}
		}
	}
	scene = scene2
	for _, f := range objs {
		f()
	}
	bg := art.NewCanvas(scene.W+40, scene.H+40)
	bg.Rect(0, 0, bg.W, bg.H, art.Hex(0x100c18))
	bg.Stamp(scene, 20, 20)
	save("/data/td/preview_scene.png", scale(bg, 2))

	// Sprite sheet.
	sheet := art.NewCanvas(8*56+8, 3*56+8*40+120)
	sheet.Rect(0, 0, sheet.W, sheet.H, art.Hex(0x5b3333))
	for lv := 0; lv < 3; lv++ {
		for i := 0; i < 8; i++ {
			x, y := 28+i*56, 40+lv*56
			sheet.Stamp(bases[lv].C, x-bases[lv].OX, y-bases[lv].OY)
			sheet.Stamp(heads[lv][i].C, x-heads[lv][i].OX, y-heads[lv][i].OY)
		}
	}
	for h := 0; h < 8; h++ {
		for f := 0; f < 4; f++ {
			x, y := 28+f*56, 3*56+30+h*40
			s := crawl[h][f]
			sheet.Stamp(s.C, x-s.OX, y-s.OY)
		}
	}
	_ = color.RGBA{}
	save("/data/td/preview_sheet.png", scale(sheet, 3))
}
