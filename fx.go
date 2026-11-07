package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
)

// Particle is a small square or glow in world space (z is height).
type Particle struct {
	X, Y, Z    float64
	VX, VY, VZ float64
	Floor      float64 // ground height where it lands
	T, Life    float64
	Size       float64
	Size1      float64
	Col        color.RGBA
	Drag, Grav float64
	Glow       bool
	Smoke      bool
	Plus       bool // drawn as a small cross
	Flake      bool // drawn as a snowflake
}

// RailFX is a fading railgun beam.
type RailFX struct {
	X0, Y0, Z0 float64
	X1, Y1, Z1 float64
	T, Life    float64
	Pierce     bool
}

// Ring is an expanding circle on the ground (build, upgrade, leak effects).
type Ring struct {
	X, Y, Z, R0, R1 float64
	T, Life         float64
	Width           float64
	Col             color.RGBA
}

// Decal is a fading mark on the ground (goo or scorch).
type Decal struct {
	X, Y, Z float64
	Img     *ebiten.Image
	S       float64
	Flip    bool
	T, Life float64
}

// Corpse is the short death pop of an alien or the sinking wreck of a robot.
type Corpse struct {
	X, Y, Z float64
	Look    *enemyLook
	Spr     *sprite // drawn instead of the look's frame (wyrm body parts)
	Heading int
	Frame   int
	T, Life float64
	Hold    float64 // seconds it waits whole before bursting
	Flash   color.RGBA
	Wreck   bool
}

// Beam is a short repair ray between two enemies (looked up by ID each frame).
type Beam struct {
	From, To int
	T, Life  float64
}

type delayed struct {
	t  float64
	fn func()
}

// Floater is a short rising text anchored to a world point.
type Floater struct {
	X, Y, Z float64
	T, Life float64
	Text    string
	Col     color.RGBA
	Big     bool
	Leak    int // base damage in percent; leak popups merge while young
}

// FX owns all purely visual, short-lived effects.
type FX struct {
	Parts    []Particle
	Rings    []Ring
	Decals   []Decal
	Corpses  []Corpse
	Floaters []Floater
	Beams    []Beam
	Rails    []RailFX
	later    []delayed
	as       *Assets
	rng      *rand.Rand
}

func NewFX(as *Assets) *FX { return &FX{as: as, rng: rand.New(rand.NewSource(7))} }

func (f *FX) Reset() {
	f.Parts, f.Rings, f.Decals, f.Corpses, f.Floaters = f.Parts[:0], f.Rings[:0], f.Decals[:0], f.Corpses[:0], f.Floaters[:0]
	f.Beams, f.later, f.Rails = f.Beams[:0], f.later[:0], f.Rails[:0]
}

func (f *FX) after(t float64, fn func()) { f.later = append(f.later, delayed{t, fn}) }

func (f *FX) decal(d Decal) {
	if len(f.Decals) > 160 {
		f.Decals = f.Decals[1:]
	}
	f.Decals = append(f.Decals, d)
}

func (f *FX) r(lo, hi float64) float64 { return lo + f.rng.Float64()*(hi-lo) }

func (f *FX) add(p Particle) {
	if len(f.Parts) >= 900 {
		return
	}
	if p.Size1 == 0 {
		p.Size1 = p.Size
	}
	f.Parts = append(f.Parts, p)
}

// Burst emits n particles from a point. dir/spread pick the horizontal direction,
// up is the typical upward speed and grav pulls them back to the floor.
func (f *FX) Burst(x, y, z, floor float64, n int, cols []color.RGBA, speed, up, grav, life, size float64, glow bool, dir, spread float64) {
	for i := 0; i < n; i++ {
		a := dir + f.r(-spread, spread)
		s := speed * f.r(0.35, 1)
		f.add(Particle{
			X: x, Y: y, Z: z, VX: math.Cos(a) * s, VY: math.Sin(a) * s, VZ: up * f.r(0.2, 1.1), Floor: floor,
			Life: life * f.r(0.6, 1), Size: size * f.r(0.75, 1.25), Size1: size * 0.45,
			Col: cols[f.rng.Intn(len(cols))], Drag: 5, Grav: grav, Glow: glow,
		})
	}
}

func (f *FX) HitSparks(x, y, z, floor, angle float64, level int, armored bool) {
	cols := []color.RGBA{art.Cyan, art.White, art.Blue}
	if armored {
		// Plasma splashes off the plates with hot metal sparks.
		f.Burst(x, y, z, floor, 3+level, cols, 70, 35, 220, 0.25, 1.8, true, angle+math.Pi, 1.2)
		f.Burst(x, y, z, floor, 5+level*2, []color.RGBA{art.Yellow, art.White, art.Amber}, 150, 70, 380, 0.32, 1.3, true, angle+math.Pi, 1.5)
		return
	}
	f.Burst(x, y, z, floor, 5+level*2, cols, 85, 45, 220, 0.3, 2, true, angle+math.Pi, 1.1)
}

func (f *FX) Fizzle(x, y, z, floor float64) {
	f.Burst(x, y, z, floor, 4, []color.RGBA{art.Cyan, art.Blue}, 40, 20, 120, 0.22, 1.2, true, 0, math.Pi)
}

// Kill plays the death of an enemy: aliens burst into goo, robots explode.
func (f *FX) Kill(lk *enemyLook, x, y, z, floor float64, heading, frame int) {
	if lk.robot {
		f.RobotExplosion(lk, x, y, z, floor, heading, frame)
		return
	}
	s := lk.size
	// Flying aliens pop where they fly; their goo rains down on the ground.
	f.Corpses = append(f.Corpses, Corpse{X: x, Y: y, Z: floor + lk.hover, Look: lk, Heading: heading, Frame: frame, Life: 0.2 + 0.1*(s-1)})
	n := int(16 * s * math.Sqrt(s))
	sp := 80 * math.Sqrt(s)
	f.Burst(x, y, z, floor, n, lk.goo, sp, 75*math.Sqrt(s), 300, 0.7, 2.6*math.Sqrt(s), false, 0, math.Pi)
	f.Burst(x, y, z, floor, max(3, int(6*s)), []color.RGBA{lk.goo[0], art.Yellow}, sp*0.6, 40, 60, 0.35, 2, true, 0, math.Pi)
	f.decal(Decal{X: x + f.r(-2, 2), Y: y + f.r(-2, 2), Z: floor, Img: lk.splats[f.rng.Intn(len(lk.splats))], S: math.Max(0.55, s*0.9), Flip: f.rng.Intn(2) == 0, Life: 30})
	if lk.def.Boss {
		// The queen bursts in several waves of goo.
		f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 10, R1: 70, Life: 0.6, Width: 3, Col: art.Magenta})
		for i := 1; i <= 4; i++ {
			dx, dy := f.r(-14, 14), f.r(-10, 10)
			f.after(0.12*float64(i), func() {
				f.Burst(x+dx, y+dy, z+f.r(0, 6), floor, 22, lk.goo, 110, 110, 300, 0.8, 3, false, 0, math.Pi)
				f.Burst(x+dx, y+dy, z, floor, 6, []color.RGBA{art.Lime, art.Yellow}, 70, 50, 60, 0.4, 2.4, true, 0, math.Pi)
				f.decal(Decal{X: x + dx, Y: y + dy, Z: floor, Img: lk.splats[f.rng.Intn(len(lk.splats))], S: 1.2, Flip: f.rng.Intn(2) == 0, Life: 30})
			})
		}
	}
}

// RobotExplosion blows a robot apart: fire, sparks, flying scrap, smoke and a wreck.
func (f *FX) RobotExplosion(lk *enemyLook, x, y, z, floor float64, heading, frame int) {
	s := lk.size
	fire := []color.RGBA{art.Orange, art.Amber, art.Yellow, art.White, art.Red}
	f.Burst(x, y, z, floor, int(24*s), fire, 95*s, 80*s, 120, 0.55, 3.2*s, true, 0, math.Pi)
	f.Burst(x, y, z, floor, int(14*s), []color.RGBA{art.Yellow, art.White}, 190, 110, 420, 0.5, 1.3, true, 0, math.Pi)
	f.Burst(x, y, z, floor, int(12*s), []color.RGBA{art.Slate, art.Steel, art.Rust, art.Navy}, 110, 150, 420, 1.1, 2.6, false, 0, math.Pi)
	for i := 0; i < int(5*s); i++ {
		f.add(Particle{
			X: x + f.r(-5, 5), Y: y + f.r(-4, 4), Z: z + f.r(-2, 4), VX: f.r(-10, 10), VY: f.r(-8, 8), VZ: f.r(14, 30),
			Life: f.r(0.9, 1.5), Size: f.r(3, 4.5), Size1: f.r(7, 10), Col: []color.RGBA{art.Steel, art.Gray, art.Slate}[f.rng.Intn(3)],
			Drag: 1.2, Smoke: true, Floor: -100,
		})
	}
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 6, R1: 34 * s, Life: 0.45, Width: 2.5, Col: art.Orange})
	if lk.hover == 0 {
		f.Corpses = append(f.Corpses, Corpse{X: x, Y: y, Z: floor, Look: lk, Heading: heading, Frame: frame, Life: 1.4, Wreck: true})
		f.decal(Decal{X: x, Y: y, Z: floor, Img: f.as.Scorch[f.rng.Intn(len(f.as.Scorch))], S: 0.9 + 0.1*s, Flip: f.rng.Intn(2) == 0, Life: 30})
	}
}

// RepairPulse is the ring a repair drone sends over the ground around it.
func (f *FX) RepairPulse(x, y, floor, radius float64, active bool) {
	al, w := 0.55, 1.6
	if active {
		al, w = 1, 2.4
	}
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 6, R1: radius, Life: 0.7, Width: w, Col: pm(art.Lime, al)})
}

// Repaired floats a few green crosses up from a patched enemy.
func (f *FX) Repaired(from, to int, x, y, z float64) {
	f.Beams = append(f.Beams, Beam{From: from, To: to, Life: 0.35})
	for i := 0; i < 3; i++ {
		f.add(Particle{
			X: x + f.r(-5, 5), Y: y + f.r(-3, 3), Z: z + f.r(0, 4), VX: f.r(-4, 4), VY: f.r(-3, 3), VZ: f.r(16, 26),
			Life: f.r(0.55, 0.8), Size: 1, Size1: 1, Col: art.Lime, Drag: 1.5, Glow: true, Floor: -100, Plus: true,
		})
	}
}

// Brood is the puff of the queen laying eggs behind her.
func (f *FX) Brood(x, y, floor float64) {
	f.Burst(x, y, floor+4, floor, 14, []color.RGBA{art.Magenta, art.Purple, art.Pink, art.Lime}, 55, 60, 240, 0.55, 2.2, false, 0, math.Pi)
	f.Burst(x, y, floor+3, floor, 5, []color.RGBA{art.Lime, art.Yellow}, 30, 30, 60, 0.4, 1.8, true, 0, math.Pi)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 4, R1: 22, Life: 0.4, Width: 1.6, Col: art.Magenta})
}

// Split is a splitter bursting into shards: crystal splinters and a flash.
func (f *FX) Split(x, y, z, floor float64) {
	f.Burst(x, y, z, floor, 12, []color.RGBA{art.Lilac, art.White, art.Violet}, 95, 55, 260, 0.4, 1.6, true, 0, math.Pi)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 4, R1: 20, Life: 0.35, Width: 1.6, Col: art.Lilac})
}

var (
	waterCols = []color.RGBA{art.White, art.Hex(0xbfefff), art.Cyan, art.Hex(0x5fb7e6)}
	iceCols   = []color.RGBA{art.White, art.Hex(0xd8f6ff), art.Hex(0x9fd4ea), art.Silver}
)

// Churn is the water stirring in an ice hole while a wyrm slides through.
func (f *FX) Churn(x, y, floor float64) {
	a := f.r(0, 2*math.Pi)
	r := f.r(0, 5)
	x, y = x+math.Cos(a)*r, y+math.Sin(a)*r*0.7
	f.Burst(x, y, floor+1, floor, 3, waterCols, 35, 70, 300, 0.5, 1.5, false, 0, math.Pi)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 3, R1: f.r(10, 15), Life: 0.55, Width: 1.2, Col: pm(art.Hex(0xbfefff), 0.7)})
}

// Splash is a wyrm crashing through an ice hole: water, ice chunks and rings.
func (f *FX) Splash(x, y, floor float64) {
	f.Burst(x, y, floor+3, floor, 30, waterCols, 75, 130, 330, 0.85, 2.2, false, 0, math.Pi)
	f.Burst(x, y, floor+2, floor, 12, iceCols, 95, 90, 420, 1.1, 2.8, false, 0, math.Pi)
	f.Burst(x, y, floor+4, floor, 8, []color.RGBA{art.Cyan, art.White}, 50, 60, 80, 0.45, 1.8, true, 0, math.Pi)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 6, R1: 38, Life: 0.65, Width: 2.4, Col: art.Hex(0xd8f6ff)})
	f.after(0.18, func() {
		f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 4, R1: 24, Life: 0.5, Width: 1.6, Col: art.Cyan})
	})
}

// Spit throws a glob of violet goo from (x, y, z) that lands at (tx, ty) where
// a splitter crawls out of it.
func (f *FX) Spit(x, y, z, tx, ty, floor float64) {
	const T, g = 0.28, 500.0
	for i := 0; i < 4; i++ {
		f.add(Particle{
			X: x, Y: y, Z: z, VX: (tx - x) / T * f.r(0.95, 1.05), VY: (ty - y) / T * f.r(0.95, 1.05),
			VZ: (floor-z)/T + g*T/2, Life: T, Size: 3.2 - 0.5*float64(i), Size1: 2.4,
			Col: []color.RGBA{art.Lilac, art.Violet, art.White, art.Violet}[i], Grav: g, Glow: i > 1, Floor: -100,
		})
	}
	f.after(T, func() {
		f.Burst(tx, ty, floor+4, floor, 14, []color.RGBA{art.Lilac, art.Violet, art.Purple, art.White}, 80, 70, 280, 0.5, 2, false, 0, math.Pi)
		f.Burst(tx, ty, floor+3, floor, 5, []color.RGBA{art.Lilac, art.White}, 40, 40, 60, 0.4, 1.8, true, 0, math.Pi)
		f.Rings = append(f.Rings, Ring{X: tx, Y: ty, Z: floor, R0: 4, R1: 22, Life: 0.4, Width: 1.6, Col: art.Lilac})
	})
}

// WyrmPop bursts one part of a dead wyrm after hold seconds; the head goes
// last with the biggest blast.
func (f *FX) WyrmPop(lk *enemyLook, x, y, z, floor, hold float64, head bool) {
	f.after(hold, func() {
		f.Burst(x, y, z, floor, 18, lk.goo, 85, 85, 300, 0.7, 2.6, false, 0, math.Pi)
		f.Burst(x, y, z, floor, 8, iceCols, 110, 100, 420, 0.9, 2.4, false, 0, math.Pi)
		f.Burst(x, y, z, floor, 5, []color.RGBA{art.Cyan, art.White}, 50, 40, 60, 0.35, 2, true, 0, math.Pi)
		f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 4, R1: 20, Life: 0.35, Width: 1.6, Col: art.Cyan})
		f.decal(Decal{X: x + f.r(-2, 2), Y: y + f.r(-2, 2), Z: floor, Img: lk.splats[f.rng.Intn(len(lk.splats))], S: 0.8, Flip: f.rng.Intn(2) == 0, Life: 30})
		if !head {
			return
		}
		f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 10, R1: 80, Life: 0.7, Width: 3, Col: art.Hex(0xbfefff)})
		for i := 1; i <= 4; i++ {
			dx, dy := f.r(-14, 14), f.r(-10, 10)
			f.after(0.12*float64(i), func() {
				f.Burst(x+dx, y+dy, z+f.r(0, 6), floor, 22, lk.goo, 110, 110, 300, 0.8, 3, false, 0, math.Pi)
				f.Burst(x+dx, y+dy, z, floor, 8, iceCols, 120, 120, 420, 1, 2.6, false, 0, math.Pi)
				f.Burst(x+dx, y+dy, z, floor, 6, []color.RGBA{art.Cyan, art.White}, 70, 50, 60, 0.4, 2.4, true, 0, math.Pi)
				f.decal(Decal{X: x + dx, Y: y + dy, Z: floor, Img: lk.splats[f.rng.Intn(len(lk.splats))], S: 1.2, Flip: f.rng.Intn(2) == 0, Life: 30})
			})
		}
	})
}

func (f *FX) Leak(x, y, z, floor, size float64) {
	f.Burst(x, y, z, floor, int(14*size), []color.RGBA{art.Red, art.Orange, art.Amber}, 90*math.Sqrt(size), 60, 200, 0.45, 1.8, true, math.Pi, 1.3)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 4, R1: 30 * math.Sqrt(size), Life: 0.45, Width: 2, Col: art.Red})
}

func (f *FX) Build(x, y, z float64) {
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: z, R0: 8, R1: 26, Life: 0.4, Width: 2, Col: art.Cyan})
	for i := 0; i < 12; i++ {
		a := float64(i)/12*2*math.Pi + f.r(-0.2, 0.2)
		f.add(Particle{
			X: x + math.Cos(a)*10, Y: y + math.Sin(a)*10, Z: z + 1, VX: math.Cos(a) * 34, VY: math.Sin(a) * 34, VZ: f.r(8, 24),
			Floor: z, Life: f.r(0.35, 0.55), Size: f.r(2, 3), Size1: 0.5, Col: art.Gray, Drag: 5, Grav: 80,
		})
	}
}

func (f *FX) Upgrade(x, y, z float64, level int) {
	col := art.Cyan
	if level >= 2 {
		col = art.Amber
	}
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: z, R0: 6, R1: 30, Life: 0.5, Width: 2.5, Col: col})
	for i := 0; i < 16; i++ {
		a := f.r(0, 2*math.Pi)
		d := f.r(3, 12)
		f.add(Particle{
			X: x + math.Cos(a)*d, Y: y + math.Sin(a)*d, Z: z + f.r(0, 8), VX: f.r(-5, 5), VY: f.r(-5, 5), VZ: f.r(28, 55),
			Floor: z, Life: f.r(0.5, 0.9), Size: f.r(1.2, 2), Size1: 0.4, Col: col, Drag: 1.5, Glow: true,
		})
	}
}

func (f *FX) Spawn(x, y, z, floor float64) {
	f.Burst(x, y, z, floor, 8, []color.RGBA{art.Magenta, art.Purple, art.Pink}, 40, 55, 220, 0.5, 1.8, false, 0, math.Pi)
}

func (f *FX) Explosion(x, y, z, floor float64) {
	f.Burst(x, y, z, floor, 70, []color.RGBA{art.Red, art.Orange, art.Amber, art.Yellow, art.White}, 160, 130, 160, 1.0, 4, true, 0, math.Pi)
	f.Burst(x, y, z, floor, 36, []color.RGBA{art.Slate, art.Steel, art.Navy}, 100, 160, 320, 1.6, 4.5, false, 0, math.Pi)
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor, R0: 10, R1: 90, Life: 0.7, Width: 4, Col: art.Orange})
}

func (f *FX) Smoke(x, y, z float64) {
	col := []color.RGBA{art.Steel, art.Gray, art.Slate}[f.rng.Intn(3)]
	f.add(Particle{
		X: x + f.r(-22, 22), Y: y + f.r(-22, 22), Z: z + f.r(0, 6), VX: f.r(-3, 9), VY: f.r(-4, 4), VZ: f.r(18, 32),
		Life: f.r(1.6, 2.4), Size: f.r(3, 5), Size1: f.r(8, 11), Col: col, Drag: 0.3, Smoke: true, Floor: -100,
	})
	if f.rng.Intn(3) == 0 {
		f.add(Particle{
			X: x + f.r(-18, 18), Y: y + f.r(-18, 18), Z: z, VX: f.r(-8, 8), VY: f.r(-8, 8), VZ: f.r(25, 50),
			Life: f.r(0.4, 0.8), Size: f.r(1.2, 2), Size1: 0.3, Col: []color.RGBA{art.Orange, art.Amber}[f.rng.Intn(2)], Drag: 1, Glow: true, Floor: -100,
		})
	}
}

func (f *FX) Sparkle(x, y, z float64) {
	f.add(Particle{
		X: x + f.r(-44, 44), Y: y + f.r(-44, 44), Z: z + f.r(0, 20), VZ: f.r(10, 26),
		Life: f.r(0.5, 0.9), Size: f.r(1.5, 2.5), Size1: 0.3, Floor: -100,
		Col: []color.RGBA{art.Cyan, art.Amber, art.White}[f.rng.Intn(3)], Glow: true,
	})
}

func (f *FX) Float(x, y, z float64, text string, col color.RGBA, life float64) {
	if len(f.Floaters) > 40 {
		f.Floaters = f.Floaters[1:]
	}
	f.Floaters = append(f.Floaters, Floater{X: x, Y: y, Z: z, Life: life, Text: text, Col: col})
}

// LeakFloat pops a big red "-N%" over the base; a burst of leaks sums into one popup.
func (f *FX) LeakFloat(x, y, z float64, pct int) {
	for i := len(f.Floaters) - 1; i >= 0; i-- {
		if t := &f.Floaters[i]; t.Leak > 0 && t.T < 0.45 {
			t.Leak += pct
			t.Text = fmt.Sprintf("-%d%%", t.Leak)
			return
		}
	}
	f.Float(x, y, z, fmt.Sprintf("-%d%%", pct), art.Red, 1.3)
	t := &f.Floaters[len(f.Floaters)-1]
	t.Big, t.Leak = true, pct
}

// Update advances effects by the game-time step dt.
func (f *FX) Update(dt float64) {
	ps := f.Parts[:0]
	for _, p := range f.Parts {
		p.T += dt
		if p.T >= p.Life {
			continue
		}
		d := math.Exp(-p.Drag * dt)
		p.VX *= d
		p.VY *= d
		p.VZ = p.VZ*d - p.Grav*dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Z += p.VZ * dt
		if p.Z < p.Floor {
			p.Z = p.Floor
			p.VZ = -p.VZ * 0.3
			p.VX *= 0.5
			p.VY *= 0.5
		}
		ps = append(ps, p)
	}
	f.Parts = ps
	rs := f.Rings[:0]
	for _, r := range f.Rings {
		r.T += dt
		if r.T < r.Life {
			rs = append(rs, r)
		}
	}
	f.Rings = rs
	ds := f.Decals[:0]
	for _, d := range f.Decals {
		d.T += dt
		if d.T < d.Life {
			ds = append(ds, d)
		}
	}
	f.Decals = ds
	cs := f.Corpses[:0]
	for _, c := range f.Corpses {
		c.T += dt
		if c.T < c.Hold+c.Life {
			cs = append(cs, c)
		}
	}
	f.Corpses = cs
	bs := f.Beams[:0]
	for _, b := range f.Beams {
		b.T += dt
		if b.T < b.Life {
			bs = append(bs, b)
		}
	}
	f.Beams = bs
	rl := f.Rails[:0]
	for _, r := range f.Rails {
		r.T += dt
		if r.T < r.Life {
			rl = append(rl, r)
		}
	}
	f.Rails = rl
	if len(f.later) > 0 {
		var run []func()
		ls := f.later[:0]
		for _, d := range f.later {
			d.t -= dt
			if d.t <= 0 {
				run = append(run, d.fn)
			} else {
				ls = append(ls, d)
			}
		}
		f.later = ls
		for _, fn := range run {
			fn()
		}
	}
	fs := f.Floaters[:0]
	for _, t := range f.Floaters {
		t.T += dt
		if t.T < t.Life {
			fs = append(fs, t)
		}
	}
	f.Floaters = fs
}

var (
	fireCols  = []color.RGBA{art.Yellow, art.Amber, art.Orange, art.Orange, art.Red}
	frostCols = []color.RGBA{art.White, art.Cyan, art.Hex(0x9fe8ff)}
	iceRing   = art.Hex(0x9fe8ff)
)

// Flame sprays a short burst of fire from a nozzle along angle, reaching reach pixels.
func (f *FX) Flame(x, y, z, floor, angle, cone, reach float64) {
	for i := 0; i < 6; i++ {
		a := angle + f.r(-cone, cone)*0.8
		life := f.r(0.28, 0.42)
		sp := reach / life * f.r(0.9, 1.25)
		f.add(Particle{
			X: x, Y: y, Z: z, VX: math.Cos(a) * sp, VY: math.Sin(a) * sp, VZ: f.r(-6, 10), Floor: floor + 1,
			Life: life, Size: f.r(1.8, 2.6), Size1: f.r(4.5, 6.5), Col: fireCols[f.rng.Intn(len(fireCols))],
			Drag: 1.6, Grav: -18, Glow: true,
		})
	}
	if f.rng.Intn(3) == 0 {
		a := angle + f.r(-cone, cone)
		f.add(Particle{
			X: x + math.Cos(a)*reach*0.7, Y: y + math.Sin(a)*reach*0.7, Z: z + 2, VX: math.Cos(a) * 20, VY: math.Sin(a) * 20, VZ: f.r(14, 24),
			Life: f.r(0.6, 0.9), Size: 2.5, Size1: 6, Col: []color.RGBA{art.Slate, art.Steel}[f.rng.Intn(2)], Drag: 1, Smoke: true, Floor: -100,
		})
	}
}

// Burning shows flames licking up from a burning enemy.
func (f *FX) Burning(x, y, z, spread float64) {
	f.add(Particle{
		X: x + f.r(-spread, spread), Y: y + f.r(-spread, spread)*0.6, Z: z + f.r(-1, 2), VX: f.r(-4, 4), VY: f.r(-3, 3), VZ: f.r(18, 32),
		Life: f.r(0.3, 0.5), Size: f.r(1.6, 2.4), Size1: 0.6, Col: fireCols[f.rng.Intn(len(fireCols))], Drag: 2, Glow: true, Floor: -100,
	})
}

// NapalmFire flickers flames over a burning pool.
func (f *FX) NapalmFire(x, y, floor, r float64) {
	a, d := f.r(0, 2*math.Pi), r*math.Sqrt(f.r(0, 1))
	f.add(Particle{
		X: x + math.Cos(a)*d, Y: y + math.Sin(a)*d, Z: floor + 1, VX: f.r(-3, 3), VY: f.r(-3, 3), VZ: f.r(14, 28),
		Life: f.r(0.35, 0.6), Size: f.r(1.8, 2.8), Size1: 0.6, Col: fireCols[f.rng.Intn(len(fireCols))], Drag: 2, Glow: true, Floor: -100,
	})
}

func (f *FX) NapalmSplash(x, y, floor float64) {
	f.Burst(x, y, floor+2, floor, 10, fireCols, 60, 40, 200, 0.4, 2, true, 0, math.Pi)
}

// MissileTrail leaves smoke and sparks behind a missile.
func (f *FX) MissileTrail(x, y, z float64) {
	f.add(Particle{
		X: x, Y: y, Z: z, VX: f.r(-3, 3), VY: f.r(-3, 3), VZ: f.r(2, 7),
		Life: f.r(0.45, 0.7), Size: 1.6, Size1: f.r(3.5, 5), Col: []color.RGBA{art.Gray, art.Silver, art.Steel}[f.rng.Intn(3)],
		Drag: 1.5, Smoke: true, Floor: -100,
	})
	if f.rng.Intn(2) == 0 {
		f.add(Particle{X: x, Y: y, Z: z, Life: 0.12, Size: 1.6, Size1: 0.5, Col: art.Amber, Glow: true, Floor: -100})
	}
}

// MissileBlast is the air burst of an anti-air missile.
func (f *FX) MissileBlast(x, y, z, floor, r float64) {
	f.Burst(x, y, z, floor, 18, []color.RGBA{art.Orange, art.Amber, art.Yellow, art.White, art.Red}, 90, 40, 60, 0.4, 2.6, true, 0, math.Pi)
	f.Burst(x, y, z, floor, 8, []color.RGBA{art.Slate, art.Steel, art.Gray}, 60, 60, 260, 0.8, 1.6, false, 0, math.Pi)
	for i := 0; i < 4; i++ {
		f.add(Particle{
			X: x + f.r(-4, 4), Y: y + f.r(-3, 3), Z: z + f.r(-2, 2), VX: f.r(-8, 8), VY: f.r(-6, 6), VZ: f.r(6, 16),
			Life: f.r(0.7, 1.1), Size: 3, Size1: f.r(6, 8), Col: []color.RGBA{art.Steel, art.Gray}[f.rng.Intn(2)], Drag: 1.2, Smoke: true, Floor: -100,
		})
	}
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: z, R0: 3, R1: math.Max(12, r), Life: 0.3, Width: 2, Col: art.Amber})
}

// FrostPulse is the wave a cryo turret sends over the ground.
func (f *FX) FrostPulse(x, y, z, floor, r float64, freeze bool) {
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor + 1, R0: 8, R1: r, Life: 0.55, Width: 2.6, Col: iceRing})
	n := 8
	if freeze {
		f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: floor + 1, R0: 6, R1: r * 0.92, Life: 0.75, Width: 5, Col: art.White})
		n = 22
	}
	for i := 0; i < n; i++ {
		a, d := f.r(0, 2*math.Pi), r*math.Sqrt(f.r(0.05, 1))
		f.add(Particle{
			X: x + math.Cos(a)*d, Y: y + math.Sin(a)*d, Z: floor + f.r(2, 10), VX: math.Cos(a) * 8, VY: math.Sin(a) * 8, VZ: f.r(-6, 4),
			Life: f.r(0.6, 1.0), Size: 1, Size1: 0.6, Col: frostCols[f.rng.Intn(len(frostCols))], Drag: 1, Glow: true, Floor: floor, Flake: true,
		})
	}
	f.Burst(x, y, z, floor, 6, frostCols, 50, 30, 40, 0.4, 1.6, true, 0, math.Pi)
}

// FreezeBurst covers a freezing enemy in ice shards.
func (f *FX) FreezeBurst(x, y, z, floor float64) {
	f.Burst(x, y, z, floor, 10, frostCols, 55, 50, 220, 0.45, 1.8, true, 0, math.Pi)
}

// RailSparks fly off an enemy hit by a railgun beam.
func (f *FX) RailSparks(x, y, z, floor, angle float64) {
	f.Burst(x, y, z, floor, 10, []color.RGBA{art.Lilac, art.White, art.Violet}, 110, 50, 240, 0.35, 1.8, true, angle, 0.9)
}

// Rail adds a beam and a trail of ions along it.
func (f *FX) Rail(x0, y0, z0, x1, y1, z1 float64, pierce bool) {
	f.Rails = append(f.Rails, RailFX{X0: x0, Y0: y0, Z0: z0, X1: x1, Y1: y1, Z1: z1, Life: 0.35, Pierce: pierce})
	n := int(math.Hypot(x1-x0, y1-y0) / 9)
	for i := 0; i < n; i++ {
		k := f.r(0, 1)
		f.add(Particle{
			X: x0 + (x1-x0)*k, Y: y0 + (y1-y0)*k, Z: z0 + (z1-z0)*k, VX: f.r(-6, 6), VY: f.r(-6, 6), VZ: f.r(4, 14),
			Life: f.r(0.3, 0.6), Size: 1.4, Size1: 0.4, Col: []color.RGBA{art.Lilac, art.Violet, art.White}[f.rng.Intn(3)], Drag: 2, Glow: true, Floor: -100,
		})
	}
}

// Launch is the puff of a missile leaving its tube.
func (f *FX) Launch(x, y, z, angle float64) {
	for i := 0; i < 4; i++ {
		f.add(Particle{
			X: x, Y: y, Z: z, VX: -math.Cos(angle)*f.r(10, 25) + f.r(-8, 8), VY: -math.Sin(angle)*f.r(10, 25) + f.r(-8, 8), VZ: f.r(2, 10),
			Life: f.r(0.5, 0.8), Size: 2.2, Size1: f.r(5, 7), Col: []color.RGBA{art.Gray, art.Silver}[f.rng.Intn(2)], Drag: 2, Smoke: true, Floor: -100,
		})
	}
}

// Sold breaks a turret down into credits.
func (f *FX) Sold(x, y, z float64) {
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: z, R0: 6, R1: 24, Life: 0.4, Width: 2, Col: art.Amber})
	f.Burst(x, y, z+6, z, 14, []color.RGBA{art.Gray, art.Steel, art.Slate}, 60, 70, 260, 0.6, 2.2, false, 0, math.Pi)
	f.Burst(x, y, z+6, z, 8, []color.RGBA{art.Yellow, art.Amber}, 40, 50, 60, 0.5, 1.6, true, 0, math.Pi)
}

// Cancelled fizzles a hologram away.
func (f *FX) Cancelled(x, y, z float64) {
	f.Rings = append(f.Rings, Ring{X: x, Y: y, Z: z, R0: 14, R1: 4, Life: 0.3, Width: 2, Col: art.Cyan})
	f.Burst(x, y, z+8, z, 12, []color.RGBA{art.Cyan, art.Blue, art.White}, 45, 40, 30, 0.45, 1.6, true, 0, math.Pi)
}

// BuildSparks crackle along the build line of a turret under construction.
func (f *FX) BuildSparks(x, y, z float64, col color.RGBA) {
	a := f.r(0, 2*math.Pi)
	f.add(Particle{
		X: x + math.Cos(a)*f.r(4, 10), Y: y + math.Sin(a)*f.r(3, 8), Z: z, VX: f.r(-14, 14), VY: f.r(-10, 10), VZ: f.r(4, 18),
		Life: f.r(0.25, 0.45), Size: 1.3, Size1: 0.3, Col: col, Drag: 3, Grav: 60, Glow: true, Floor: art.PadH,
	})
}

// Chimney puffs a little smoke from a colony chimney.
func (f *FX) Chimney(x, y, z float64) {
	col := []color.RGBA{art.Steel, art.Gray, art.Silver}[f.rng.Intn(3)]
	f.add(Particle{
		X: x + f.r(-1, 1), Y: y + f.r(-1, 1), Z: z, VX: f.r(2, 6), VY: f.r(-2, 1), VZ: f.r(9, 15),
		Life: f.r(2.2, 3.2), Size: f.r(1.6, 2.4), Size1: f.r(5, 7), Col: col, Drag: 0.4, Smoke: true, Floor: -100,
	})
}

// Weld throws a few bright sparks from a welding torch.
func (f *FX) Weld(x, y, z float64) {
	for i := 0; i < 6; i++ {
		a := f.r(0, 2*math.Pi)
		s := f.r(10, 30)
		f.add(Particle{
			X: x, Y: y, Z: z, VX: math.Cos(a) * s, VY: math.Sin(a) * s, VZ: f.r(10, 35), Floor: art.SlabH,
			Life: f.r(0.25, 0.5), Size: 1.2, Size1: 0.3, Col: []color.RGBA{art.White, art.Cyan, art.Yellow}[f.rng.Intn(3)], Drag: 2, Grav: 90, Glow: true,
		})
	}
	f.add(Particle{X: x, Y: y, Z: z + 1, Life: 0.09, Size: 4, Size1: 2, Col: art.White, Glow: true, Floor: -100})
}

// Dust blows over the ground under the engines of a shuttle (k: strength).
func (f *FX) Dust(x, y, z, k float64) {
	a := f.r(0, 2*math.Pi)
	d := f.r(6, 14)
	s := f.r(25, 45) * k
	f.add(Particle{
		X: x + math.Cos(a)*d, Y: y + math.Sin(a)*d, Z: z + 0.5, VX: math.Cos(a) * s, VY: math.Sin(a) * s, VZ: f.r(3, 10),
		Life: f.r(0.6, 1.1), Size: f.r(2, 3), Size1: f.r(5, 8), Col: []color.RGBA{art.Ochre, art.Rust, art.Peach}[f.rng.Intn(3)],
		Drag: 2.2, Smoke: true, Floor: -100,
	})
}

// RiseSparks crackle along the build line of a colony building.
func (f *FX) RiseSparks(x, y, z, w float64) {
	f.add(Particle{
		X: x + f.r(-w, w), Y: y + f.r(-3, 3), Z: z, VX: f.r(-12, 12), VY: f.r(-8, 8), VZ: f.r(4, 18),
		Life: f.r(0.25, 0.45), Size: 1.3, Size1: 0.3, Col: []color.RGBA{art.Cyan, art.White}[f.rng.Intn(2)],
		Drag: 3, Grav: 60, Glow: true,
	})
}
