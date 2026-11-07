// Command scene renders a level with turrets and enemies into a PNG (art review).
package main

import (
	"image"
	"image/png"
	"math"
	"os"

	"outpost/internal/art"
	"outpost/internal/game"
)

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
	spec := game.LevelByID(os.Args[1])
	lvl, err := game.ParseLevel(spec)
	if err != nil {
		panic(err)
	}
	const P = art.CellPx
	ter, ti := art.Terrain(lvl)
	oy := ti.OY
	shadow := art.NewCanvas(ter.W, ter.H)
	towers := []string{"blaster", "cryo", "flamer", "railgun", "skyguard"}
	var objs []func(*art.Canvas)
	for i, s := range lvl.Slots {
		c := s.Center()
		x, y := c.X*P, c.Y*P
		id := towers[i%len(towers)]
		lv := i % 3
		near := lvl.NearestRoad(c)
		ang := near.Sub(c).Angle()
		base, head := art.TurretBase(id, lv), art.TurretHead(id, lv, ang)
		objs = append(objs, func(dst *art.Canvas) {
			put(dst, shadow, oy, base, x, y, art.PadH)
			put(dst, shadow, oy, head, x, y, art.PadH)
		})
	}
	enemies := []string{"beetle", "crawler", "beetle", "runner", "walker", "swarmling", "beetle"}
	k := 0
	for _, rt := range lvl.Routes {
		for d := 2.0; d < rt.Path.Len-2; d += 2.6 {
			p, dir := rt.Path.At(d)
			x, y := p.X*P, p.Y*P
			id := enemies[k%len(enemies)]
			k++
			sp := art.EnemySprite(id, dir.Angle(), k%4, 8)
			objs = append(objs, func(dst *art.Canvas) { put(dst, shadow, oy, sp, x, y, ti.Height(x, y)) })
		}
	}
	bc := lvl.BaseCenter()
	base, radar := art.Base(lvl.EastGate()), art.Radar(0.7)
	objs = append(objs, func(dst *art.Canvas) {
		put(dst, nil, oy, base, bc.X*P, bc.Y*P, 0)
		put(dst, nil, oy, radar, bc.X*P+art.RadarPos.X, bc.Y*P+art.RadarPos.Y, art.RadarPos.Z)
	})
	tmp := art.NewCanvas(ter.W, ter.H)
	for _, f := range objs {
		f(tmp)
	}
	scene := art.NewCanvas(ter.W, ter.H)
	scene.Stamp(ter, 0, 0)
	for y := 0; y < shadow.H; y++ {
		for x := 0; x < shadow.W; x++ {
			if shadow.Get(x, y).A > 0 && scene.Get(x, y).A > 0 {
				scene.Blend(x, y, art.Ink, 0.38)
			}
		}
	}
	for _, f := range objs {
		f(scene)
	}
	bg := art.NewCanvas(scene.W+40, scene.H+40)
	bg.Rect(0, 0, bg.W, bg.H, art.Hex(0x100c18))
	bg.Stamp(scene, 20, 20)
	s := 2
	out := image.NewRGBA(image.Rect(0, 0, bg.W*s, bg.H*s))
	for y := 0; y < bg.H*s; y++ {
		for x := 0; x < bg.W*s; x++ {
			out.Set(x, y, bg.Img.At(x/s, y/s))
		}
	}
	f, _ := os.Create(os.Args[2])
	png.Encode(f, out)
	f.Close()
}
