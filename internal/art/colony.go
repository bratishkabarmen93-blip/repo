package art

import (
	"math"
	"math/rand"
	"strings"

	"outpost/internal/game"
)

// ---- Colonies: the buildings, the settlers and the ground of a location ----

var (
	mHull    = mat(MetalRamp, 0.06, 0.3)
	mHullMid = mat(MetalRamp, -0.05, 0.28)
	mPanel   = mat(DarkMetal, 0.04, 0.2)
	mTrim    = mat(Ramp{Plum, Rust, Orange, Amber}, 0.04, 0.3)
	mWindow  = glow(Ramp{Ochre, Amber, Yellow, White}, -0.15)
	mSolar   = mat(Ramp{Ink, Navy, DeepBlue, Blue, Cyan, White}, -0.06, 0.9)
	mGreenG  = mat(Ramp{Ink, DeepTeal, Forest, Green, Lime, White}, -0.02, 0.9)
	mPlantG  = glow(Ramp{Forest, Green, Lime, Yellow}, -0.25)
	mGreenL  = glow(Ramp{Forest, Green, Lime, White}, 0)
	mWater   = mat(Ramp{Ink, Navy, DeepBlue, Blue, Cyan}, 0.02, 0.6)
	mBronze  = mat(Ramp{Ink, Plum, Brown, Rust, Copper, Tan, Sand}, 0, 0.6)
	mStone   = mat(Ramp{Ink, Navy, Slate, Steel, Gray, Silver}, 0.02, 0.1)
	mStoneLo = mat(Ramp{Ink, Navy, Slate, Steel, Gray}, -0.04, 0.1)
	mSlab    = mat(Ramp{Ink, Navy, Slate, Steel}, 0.04, 0.1)
	mSuitB   = mat(Ramp{Ink, Navy, DeepBlue, Blue, Cyan, White}, 0.06, 0.3)
	mSuitO   = mat(Ramp{Plum, Rust, Orange, Amber, Yellow}, 0.02, 0.3)
)

// SlabH is the height of the concrete foundation of a colony lot.
const SlabH = 1.5

// turnPart rotates a part and its carving parts around the vertical axis
// through the origin (clip planes and paint stay in local coordinates).
func turnPart(p *Part, R M3) {
	p.pos, p.bc = R.Apply(p.pos), R.Apply(p.bc)
	p.rot = R.Mul(p.rot)
	p.inv = p.rot.T()
	for _, q := range p.sub {
		turnPart(q, R)
	}
}

// Turn rotates a whole scene around the vertical axis through the origin.
func Turn(s *Scene, a float64) {
	R := Yaw(a)
	for _, p := range s.Parts {
		turnPart(p, R)
	}
}

// Shift moves a whole scene.
func Shift(s *Scene, d V3) {
	for _, p := range s.Parts {
		p.move(d)
	}
}

// clipWorld keeps the world half-space n·x <= d of a part.
func clipWorld(p *Part, n V3, d float64) *Part { return p.Clip(p.inv.Apply(n), d-n.Dot(p.pos)) }

// worldPaint gives a part a pattern in world coordinates.
func worldPaint(p *Part, f func(w, n V3) *Mat) *Part {
	R, T := p.rot, p.pos
	return p.WithPaint(func(l, n V3) *Mat { return f(R.Apply(l).Add(T), R.Apply(n)) })
}

// dome is a half ellipsoid standing on the ground at c.
func dome(c V3, r, h float64, m *Mat) *Part {
	return Ellipsoid(c, V3{r, r, h}, Ident, m).Clip(V3{0, 0, -1}, 0)
}

// windowBand paints lit windows in a ring between heights z0 and z1 of a
// round part centred on (cx, cy): n windows around, each w radians wide.
func windowBand(m *Mat, cx, cy, z0, z1 float64, n int, w float64) func(p, nn V3) *Mat {
	return func(p, nn V3) *Mat {
		if p.Z < z0 || p.Z > z1 || nn.Z > 0.75 {
			return nil
		}
		a := math.Atan2(p.Y-cy, p.X-cx) + math.Pi
		k := a / (2 * math.Pi) * float64(n)
		if k-math.Floor(k) < w {
			return m
		}
		return nil
	}
}

// Facing is the turn of a building's model so that its front (+y in the
// model) looks at the street.
func Facing(loc *game.Location, b *game.Building) float64 {
	c, d := b.Center(), loc.Door(b)
	return math.Atan2(d.Y-c.Y, d.X-c.X) - math.Pi/2
}

// kind is the part of a building ID after the location: "habitat".
func kind(b *game.Building) string {
	_, k, _ := strings.Cut(b.ID, ".")
	return k
}

// BuildingScene is a colony building at a stage (1 = the first one) in its
// model space: the lot centre on top of the foundation, front towards +y.
func BuildingScene(b *game.Building, stage int) *Scene {
	switch kind(b) {
	case "habitat":
		return habitatScene(stage)
	case "solar":
		return solarScene(stage)
	case "greenhouse":
		return greenhouseScene(stage)
	case "workshop":
		return workshopScene(stage)
	case "comms":
		return commsScene(stage)
	case "pad":
		return landingPadScene(stage)
	case "monument":
		return monumentScene()
	}
	return &Scene{}
}

// Building bakes a colony building at a stage, turned to face its street.
// The anchor is the lot centre on the ground.
func Building(loc *game.Location, b *game.Building, stage int) *Sprite {
	s := BuildingScene(b, stage)
	Turn(s, Facing(loc, b))
	Shift(s, V3{0, 0, SlabH})
	return s.Bake(spriteOpts, true, 0, 0)
}

// BuildingFX are the spots of a building that the renderer animates
// (relative to the lot centre on the ground, already turned).
type BuildingFX struct {
	Red, Green, Amber []V3 // blinking lights
	Smoke             []V3 // chimneys
	Sparks            []V3 // welding
	Glow              []V3 // soft green glow (greenhouse)
	Land              *V3  // where the shuttle touches down
}

// Effects lists the animated spots of a building at a stage.
func Effects(loc *game.Location, b *game.Building, stage int) BuildingFX {
	var fx BuildingFX
	switch kind(b) {
	case "habitat":
		if stage >= 3 {
			fx.Red = append(fx.Red, V3{12, -12, 50.5})
		}
	case "solar":
		fx.Green = append(fx.Green, V3{-19, 10, 7.5})
		if stage >= 2 {
			fx.Green = append(fx.Green, V3{19, -2, 12.5}, V3{19, 6, 12.5})
		}
	case "greenhouse":
		fx.Glow = append(fx.Glow, V3{-2, -7, 6})
		if stage >= 2 {
			fx.Glow = append(fx.Glow, V3{-2, 9, 6})
		}
	case "workshop":
		fx.Smoke = append(fx.Smoke, V3{-16, -12, 30})
		fx.Amber = append(fx.Amber, V3{-9, 12.2, 15})
		if stage >= 2 {
			fx.Sparks = append(fx.Sparks, V3{17, 12, 3})
			fx.Red = append(fx.Red, V3{16, -13, 39})
		}
	case "comms":
		fx.Red = append(fx.Red, V3{11, -9, 50})
	case "pad":
		for _, d := range [][2]float64{{-30, -30}, {30, -30}, {-30, 30}, {30, 30}} {
			fx.Amber = append(fx.Amber, V3{d[0], d[1], 4})
		}
		if stage >= 2 {
			fx.Red = append(fx.Red, V3{34, -36, 37})
			fx.Land = &V3{0, 0, 3}
		}
	}
	R := Yaw(Facing(loc, b))
	turn := func(ps []V3) {
		for i := range ps {
			ps[i] = R.Apply(ps[i]).Add(V3{0, 0, SlabH})
		}
	}
	turn(fx.Red)
	turn(fx.Green)
	turn(fx.Amber)
	turn(fx.Smoke)
	turn(fx.Sparks)
	turn(fx.Glow)
	if fx.Land != nil {
		v := R.Apply(*fx.Land).Add(V3{0, 0, SlabH})
		fx.Land = &v
	}
	return fx
}

// Habitat: living domes, then a tunnel to a second dome, then a tower.
func habitatScene(stage int) *Scene {
	s := &Scene{}
	addDome := func(c V3, r, h float64) {
		s.Add(Lathe(c, r+1, r+0.6, 2.4, 28, Ident, mPanel))
		d := dome(c.Add(V3{0, 0, 2.4}), r, h, mHull)
		seams := windowBand(mWindow, c.X, c.Y, 2.4+h*0.25, 2.4+h*0.42, 9, 0.28)
		d.WithPaint(func(p, n V3) *Mat {
			w := p.Add(c).Add(V3{0, 0, 2.4})
			if m := seams(w, n); m != nil {
				return m
			}
			if p.Z > h*0.55 && math.Mod(math.Atan2(p.Y, p.X)+math.Pi+0.2, math.Pi/3) < 0.09 {
				return mHullMid
			}
			return nil
		})
		s.Add(d)
	}
	addDome(V3{-8, -5, 0}, 15, 13)
	// The airlock towards the street.
	s.Add(Cylinder(V3{-8, 7, 4.4}, V3{-8, 18, 4.4}, 4, mHullMid))
	door := Block(V3{-8, 18.2, 3.8}, V3{3, 0.5, 3.8}, Ident, 0, 0, mTrim)
	s.Add(door)
	s.Add(Block(V3{-8, 18.5, 3.5}, V3{1.6, 0.4, 2.9}, Ident, 0, 0, mWindow))
	if stage >= 2 {
		s.Add(Cylinder(V3{-2, 0, 5}, V3{12, 9, 5}, 3.4, mHullMid))
		addDome(V3{13, 10, 0}, 10.5, 10)
	}
	if stage >= 3 {
		s.Add(Cylinder(V3{-3, -11, 5}, V3{10, -12, 5}, 3, mHullMid))
		tw := Lathe(V3{12, -12, 0}, 9, 7.6, 40, 24, Ident, mHull)
		tw.WithPaint(func(p, n V3) *Mat {
			if n.Z > 0.75 {
				return nil
			}
			if math.Mod(p.Z, 6) > 3 && math.Mod(p.Z, 6) < 4.6 && p.Z > 5 && p.Z < 38 {
				a := math.Atan2(p.Y, p.X) + math.Pi
				if k := a / (2 * math.Pi) * 12; k-math.Floor(k) < 0.5 {
					return mWindow
				}
				return mPanel
			}
			return nil
		})
		s.Add(tw)
		s.Add(Lathe(V3{12, -12, 40}, 8.4, 6.2, 3, 24, Ident, mTrim))
		s.Add(dome(V3{12, -12, 43}, 6.2, 4, mGlass))
		s.Add(Cylinder(V3{12, -12, 45}, V3{12, -12, 50}, 0.5, mPanel))
		s.Add(Sphere(V3{12, -12, 50.5}, 0.9, mRedG))
	}
	return s
}

// Solar farm: tilted panel rows facing the camera, then more rows and batteries.
func solarScene(stage int) *Scene {
	s := &Scene{}
	row := func(y, x0, x1 float64) {
		R := Roll(-0.42)
		cx := (x0 + x1) / 2
		p := Block(V3{cx, y, 6}, V3{(x1 - x0) / 2, 4.6, 0.55}, R, 0, 0, mSolar)
		p.WithPaint(func(l, n V3) *Mat {
			if math.Abs(n.Z) < 0.5 {
				return mPanel
			}
			if math.Mod(l.X+40, 4.2) < 0.55 || math.Abs(l.Y) < 0.35 {
				return mPanel
			}
			return nil
		})
		s.Add(p)
		for x := x0 + 3; x <= x1-2.9; x += (x1 - x0 - 6) / 2 {
			s.Add(Cylinder(V3{x, y + 1, 0}, V3{x, y + 1, 5.5}, 0.6, mPanel))
		}
	}
	row(-6, -24, 12)
	row(6, -24, 12)
	// The inverter box.
	inv := Block(V3{-19, 13, 3}, V3{4, 2.5, 3}, Ident, 0.6, 0.6, mHull)
	s.Add(inv)
	s.Add(Block(V3{-19, 15.3, 4.4}, V3{2.4, 0.4, 0.8}, Ident, 0, 0, mPanel))
	if stage >= 2 {
		row(-18, -24, 12)
		row(18, -24, 12)
		bank := Block(V3{19, 2, 5.5}, V3{5.5, 11, 5.5}, Ident, 1, 1, mHull)
		bank.WithPaint(func(l, n V3) *Mat {
			if n.Z < 0.9 && math.Mod(l.Y+40, 5.5) < 0.6 {
				return mPanel
			}
			return nil
		})
		s.Add(bank)
		s.Add(Block(V3{19, 2, 11.5}, V3{3.5, 9, 0.6}, Ident, 0, 0.4, mTrim))
	}
	return s
}

// tunnel is an arched glass greenhouse along x at row y.
func tunnel(s *Scene, y, x0, x1, r float64) {
	g := Cylinder(V3{x0, y, 0}, V3{x1, y, 0}, r, mGreenG)
	clipWorld(g, V3{0, 0, -1}, 0)
	worldPaint(g, func(w, n V3) *Mat {
		if w.Z < 2.2 && math.Abs(n.Z) < 0.9 {
			return mPlantG
		}
		if math.Mod(w.X-x0+40, 6) < 0.8 {
			return mHull
		}
		return nil
	})
	s.Add(g)
	s.Add(Block(V3{x1 + 0.6, y, 3.3}, V3{1, 3, 3.3}, Ident, 0, 0.5, mHullMid))
	s.Add(Block(V3{x0 - 0.6, y, 3.3}, V3{1, 3, 3.3}, Ident, 0, 0.5, mHullMid))
}

// Greenhouse: one glass tunnel, then a second one and a water tank.
func greenhouseScene(stage int) *Scene {
	s := &Scene{}
	tunnel(s, -7, -21, 15, 8.5)
	if stage >= 2 {
		tunnel(s, 9, -21, 15, 7)
		tank := Lathe(V3{21, -12, 0}, 5.5, 5.5, 13, 24, Ident, mHull)
		tank.WithPaint(func(p, n V3) *Mat {
			if p.Z > 7 && p.Z < 9 && n.Z < 0.7 {
				return mWater
			}
			return nil
		})
		s.Add(tank)
		s.Add(dome(V3{21, -12, 13}, 5.5, 2.5, mHullMid))
		s.Add(Cylinder(V3{16, -9, 2}, V3{21, -9, 2}, 1, mPanel))
	}
	return s
}

// Workshop: a hangar with a chimney, then a second bay and a crane.
func workshopScene(stage int) *Scene {
	s := &Scene{}
	body := Block(V3{-8, -2, 8}, V3{16, 14, 8}, Ident, 1.2, 5.5, mHullMid)
	body.WithPaint(func(l, n V3) *Mat {
		if n.Y > 0.6 && math.Abs(l.X-(-1)) < 7.5 && l.Z < 4 {
			if math.Mod(l.X+l.Z+40, 3) < 1.5 {
				return mTrim
			}
			return mPanel
		}
		if n.Z < 0.5 && math.Mod(l.Z+20, 4.5) < 0.5 {
			return mPanel
		}
		return nil
	})
	s.Add(body)
	s.Add(Block(V3{-9, 12.2, 14.5}, V3{1.4, 0.4, 0.9}, Ident, 0, 0, mWindow))
	s.Add(Cylinder(V3{-16, -12, 10}, V3{-16, -12, 28}, 2.6, mPanel))
	s.Add(Lathe(V3{-16, -12, 28}, 3.2, 3.2, 1.5, 16, Ident, mTrim))
	if stage >= 2 {
		bay := Block(V3{17, 4, 6}, V3{8, 9, 6}, Ident, 1, 3, mHull)
		bay.WithPaint(func(l, n V3) *Mat {
			if n.Y > 0.6 && math.Abs(l.X) < 4.5 && l.Z < 1.5 {
				return mPanel
			}
			return nil
		})
		s.Add(bay)
		// The crane: a mast, a jib over the yard and a hook.
		s.Add(Block(V3{16, -13, 18}, V3{1.4, 1.4, 18}, Ident, 0, 0, mTrim))
		s.Add(Block(V3{16, -13, 36.5}, V3{2.2, 2.2, 1}, Ident, 0, 0, mPanel))
		jib := Rod(V3{24, -13, 36.5}, V3{-2, -13, 36.5}, 0.9, mTrim)
		s.Add(jib)
		s.Add(Block(V3{25, -13, 35.5}, V3{2.5, 2, 2}, Ident, 0, 0, mPanel))
		s.Add(Cylinder(V3{4, -13, 22}, V3{4, -13, 36}, 0.3, mPanel))
		s.Add(Block(V3{4, -13, 20.5}, V3{2, 2, 1.5}, Ident, 0, 0, mAmber))
		s.Add(Sphere(V3{16, -13, 39}, 0.8, mRedG))
	}
	return s
}

// Comms array: a lattice mast, then a big dish.
func commsScene(stage int) *Scene {
	s := &Scene{}
	mx, my := 11.0, -9.0
	const top = 46.0
	legs := [][2]float64{{-7, -7}, {7, -7}, {7, 7}, {-7, 7}}
	at := func(i int, z float64) V3 {
		k := 1 - z/top*0.85
		return V3{mx + legs[i][0]*k, my + legs[i][1]*k, z}
	}
	for i := range legs {
		s.Add(Rod(at(i, 0), at(i, top), 0.55, mTrim))
	}
	for z := 6.0; z < top-2; z += 7 {
		for i := range legs {
			j := (i + 1) % 4
			s.Add(Rod(at(i, z), at(j, z), 0.35, mHull))
			s.Add(Rod(at(i, z), at(j, z+7), 0.3, mHull))
		}
	}
	s.Add(Cylinder(V3{mx, my, top}, V3{mx, my, 50}, 0.5, mPanel))
	s.Add(Sphere(V3{mx, my, 50}, 0.9, mRedG))
	s.Add(Block(V3{mx, my, 0.8}, V3{8, 8, 0.8}, Ident, 1, 0, mPanel))
	hut := Block(V3{-11, 12, 5.5}, V3{10, 7, 5.5}, Ident, 1, 2, mHull)
	hut.WithPaint(func(l, n V3) *Mat {
		if n.Y > 0.6 && l.Z > 0.5 && l.Z < 2.5 && math.Mod(l.X+40, 4) < 2.2 {
			return mWindow
		}
		return nil
	})
	s.Add(hut)
	s.Add(Block(V3{-11, 12, 11.4}, V3{6, 4, 0.5}, Ident, 0.5, 0, mPanel))
	s.Add(Block(V3{-3, 19.2, 3.2}, V3{2, 0.4, 3.2}, Ident, 0, 0, mTrim))
	if stage >= 2 {
		cx, cy := -12.0, -9.0
		s.Add(Lathe(V3{cx, cy, 0}, 6.5, 5, 4, 20, Ident, mHullMid))
		s.Add(Cylinder(V3{cx, cy, 4}, V3{cx, cy, 16}, 2.4, mPanel))
		R := Yaw(2.25).Mul(Pitch(0.85))
		x := xf{R, V3{cx, cy, 21}}
		dish := Sphere(x.p(V3{0, 0, 0}), 15, mHull)
		dish.Sub(mLamp, Sphere(x.p(V3{5.6, 0, 0}), 15, nil))
		dish.Clip(R.Apply(V3{1, 0, 0}), -6)
		RT := R.T()
		dish.WithPaint(func(p, n V3) *Mat {
			if q := RT.Apply(p); math.Mod(math.Hypot(q.Y, q.Z), 4) < 0.5 {
				return mHullMid
			}
			return nil
		})
		s.Add(dish)
		s.Add(Rod(x.p(V3{-9, 0, 0}), x.p(V3{5, 0, 0}), 0.6, mPanel))
		for _, a := range []float64{0, 2.1, 4.2} {
			s.Add(Rod(x.p(V3{-6.5, 12.5 * math.Cos(a), 12.5 * math.Sin(a)}), x.p(V3{5, 0, 0}), 0.3, mPanel))
		}
		s.Add(Sphere(x.p(V3{5.4, 0, 0}), 1.1, mRedG))
	}
	return s
}

// Landing pad: a marked deck with lights, then a control tower.
func landingPadScene(stage int) *Scene {
	s := &Scene{}
	deck := Prism(V3{0, 0, 0}, 37, 36, 2.5, 8, Yaw(math.Pi/8), mPanel)
	deck.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		r := math.Hypot(p.X, p.Y)
		switch {
		case r > 23 && r < 26:
			return mAmber
		case r < 18 && (math.Abs(p.X) < 1.6 || math.Abs(p.Y) < 1.6) && r > 4:
			return mHull
		case r > 31 && int(math.Floor((math.Atan2(p.Y, p.X)+math.Pi)*16/math.Pi))%2 == 0:
			return mAmber
		}
		return nil
	})
	s.Add(deck)
	for _, d := range [][2]float64{{-30, -30}, {30, -30}, {-30, 30}, {30, 30}} {
		s.Add(Cylinder(V3{d[0], d[1], 0}, V3{d[0], d[1], 3.5}, 1.4, mPanel))
		s.Add(Sphere(V3{d[0], d[1], 3.8}, 1.1, mWindow))
	}
	if stage >= 2 {
		tx, ty := 34.0, -36.0
		s.Add(Lathe(V3{tx, ty, 0}, 5.5, 4, 26, 16, Ident, mHull))
		cab := Lathe(V3{tx, ty, 26}, 6, 7.5, 5, 16, Ident, mGlass)
		s.Add(cab)
		s.Add(Lathe(V3{tx, ty, 31}, 8, 6, 2, 16, Ident, mTrim))
		s.Add(Cylinder(V3{tx, ty, 33}, V3{tx, ty, 37}, 0.5, mPanel))
		s.Add(Sphere(V3{tx, ty, 37}, 0.9, mRedG))
		// A fuel tank and its pipe.
		s.Add(Lathe(V3{-36, -34, 0}, 5, 5, 8, 20, Ident, mHullMid))
		s.Add(dome(V3{-36, -34, 8}, 5, 2.5, mTrim))
	}
	return s
}

// bronze returns the statue version of a scene: every surface bronze, the
// glowing bits gold, the patterns gone.
func bronze(s *Scene) {
	cache := map[*Mat]*Mat{}
	conv := func(m *Mat) *Mat {
		if m == nil {
			return nil
		}
		if c, ok := cache[m]; ok {
			return c
		}
		c := mat(mBronze.Ramp, m.Bias+0.04, 0.6)
		if m.Glow {
			c = mat(GoldRamp, 0.15, 0.8)
		}
		cache[m] = c
		return c
	}
	var fix func(p *Part)
	fix = func(p *Part) {
		p.M, p.Cap, p.Cut, p.Paint = conv(p.M), conv(p.Cap), conv(p.Cut), nil
		for _, q := range p.sub {
			fix(q)
		}
	}
	for _, p := range s.Parts {
		fix(p)
	}
}

// Monument: the Hive Queen cast in bronze on a stone pedestal.
func monumentScene() *Scene {
	s := &Scene{}
	base := Block(V3{0, 0, 3}, V3{30, 26, 3}, Ident, 6, 1, mStoneLo)
	s.Add(base)
	top := Block(V3{0, 0, 8}, V3{24, 20, 2.4}, Ident, 4, 1, mStone)
	top.WithPaint(func(l, n V3) *Mat {
		if n.Y > 0.6 && math.Abs(l.X) < 9 && l.Z < 1 && l.Z > -1.6 {
			return mBronze
		}
		return nil
	})
	s.Add(top)
	q := QueenScene(math.Pi/2, 1, 8)
	bronze(q)
	Shift(q, V3{0, 1, 10.4})
	s.Parts = append(s.Parts, q.Parts...)
	// Lamps at the corners of the plinth.
	for _, d := range [][2]float64{{-27, 23}, {27, 23}} {
		s.Add(Cylinder(V3{d[0], d[1], 6}, V3{d[0], d[1], 12}, 0.7, mPanel))
		s.Add(Sphere(V3{d[0], d[1], 12.6}, 1.4, mWindow))
	}
	return s
}

// ColonistScene is a settler in a pressure suit walking along heading at
// animation frame f of n (variant picks the suit colour).
func ColonistScene(variant int, heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.8}
	suit := []*Mat{mSuitO, mSuitB}[variant%2]
	ph := float64(f) / float64(n) * 2 * math.Pi
	for _, side := range []float64{-1, 1} {
		sw := math.Sin(ph) * 1.4 * side
		up := math.Max(0, math.Cos(ph)*side) * 0.7
		s.Add(Rod(x.p(V3{0, side * 0.85, 3.2}), x.p(V3{sw, side * 0.95, 0.6 + up}), x.r(0.7), mPanel))
	}
	s.Add(Ellipsoid(x.p(V3{0, 0, 4.6}), x.v(V3{1.4, 1.7, 1.9}), x.R, suit))
	s.Add(Block(x.p(V3{-1.4, 0, 4.9}), x.v(V3{0.7, 1.25, 1.4}), x.R, 0, 0, mPanel))
	for _, side := range []float64{-1, 1} {
		sw := -math.Sin(ph) * 1.1 * side
		s.Add(Rod(x.p(V3{0, side * 1.7, 5.6}), x.p(V3{sw, side * 2, 3.6}), x.r(0.55), suit))
	}
	s.Add(Sphere(x.p(V3{0.1, 0, 7.3}), x.r(1.45), suit))
	s.Add(Sphere(x.p(V3{0.8, 0, 7.4}), x.r(1), mGlass))
	return s
}

// ShuttleScene is the supply shuttle with its nose along heading, standing on
// its legs at the origin.
func ShuttleScene(heading float64) *Scene {
	s := &Scene{}
	x := xf{Yaw(heading), V3{}}
	R := x.R
	body := Ellipsoid(x.p(V3{0, 0, 6}), V3{15, 5.2, 4}, R, mHull)
	body.WithPaint(func(l, n V3) *Mat {
		if math.Abs(l.X+3) < 1.6 {
			return mTrim
		}
		return nil
	})
	s.Add(body)
	s.Add(Ellipsoid(x.p(V3{8.5, 0, 7.6}), V3{4.6, 3, 2.4}, R, mGlass))
	for _, side := range []float64{-1, 1} {
		w := Block(x.p(V3{-4, side * 8, 5}), V3{5.5, 5, 0.7}, R.Mul(Yaw(side*0.45)), 2.5, 0, mHullMid)
		s.Add(w)
		s.Add(Block(x.p(V3{-6, side * 12.2, 5.4}), V3{2.6, 0.9, 0.9}, R, 0, 0, mTrim))
		s.Add(Cylinder(x.p(V3{-17, side * 3.4, 5.5}), x.p(V3{-11, side * 3.4, 5.5}), 2.1, mPanel))
		s.Add(Sphere(x.p(V3{-17.2, side * 3.4, 5.5}), 1.5, mCyanG))
		for _, fx := range []float64{-7, 7} {
			s.Add(Rod(x.p(V3{fx, side * 3, 4}), x.p(V3{fx * 1.1, side * 5, 0.5}), 0.45, mPanel))
		}
	}
	s.Add(Block(x.p(V3{-11, 0, 10}), V3{3.5, 0.6, 4}, R, 0, 1, mTrim))
	return s
}

// ---- The ground of a colony ----

// Paved streets: warm grey concrete between the red dunes.
var concreteRamp = Ramp{Hex(0x2b2230), Hex(0x4b3f4c), Hex(0x6d5f69), Hex(0x8c7c82), Hex(0xa8989a), Hex(0xcbbcb6)}

func colonyBiome(loc *game.Location) *biome {
	b := *desertBiome
	if loc.Biome == "ice" {
		b = *iceBiome
	}
	b.road = concreteRamp
	return &b
}

// Street widths (half, world pixels).
const (
	avenueHW = 9.5
	spurHW   = 6.5
	// streetDepth is how far a street sits below the ground (a low kerb).
	streetDepth = 0.9
)

// streetSeg writes the distance to the street segment a-b (cells) into the
// field, scaled so that groundColor's road width (PathHalfWidth) maps to hw.
func streetSeg(dist []float64, W, H int, a, b game.Vec, hw float64) {
	k := PathHalfWidth / hw
	reach := hw + 10
	ax, ay := a.X*CellPx, a.Y*CellPx
	bx, by := b.X*CellPx, b.Y*CellPx
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	for y := max(int(math.Min(ay, by)-reach), 0); y <= min(int(math.Max(ay, by)+reach), H-1); y++ {
		for x := max(int(math.Min(ax, bx)-reach), 0); x <= min(int(math.Max(ax, bx)+reach), W-1); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l2))
			}
			if d := math.Hypot(px-ax-dx*t, py-ay-dy*t) * k; d < dist[y*W+x] {
				dist[y*W+x] = d
			}
		}
	}
}

// slabScene is the concrete foundation of a lot (w×h cells) centred on (cx, cy).
func slabScene(cx, cy float64, w, h int) *Scene {
	s := &Scene{}
	hx, hy := float64(w)*CellPx/2-2.5, float64(h)*CellPx/2-2.5
	slab := Block(V3{cx, cy, SlabH / 2}, V3{hx, hy, SlabH / 2}, Ident, 3, 0.5, mSlab)
	slab.WithPaint(func(l, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		ax, ay := math.Abs(l.X), math.Abs(l.Y)
		// Amber corner marks.
		if (ax > hx-4.5 && ax < hx-3 && ay > hy-9 && ay < hy-3) || (ay > hy-4.5 && ay < hy-3 && ax > hx-9 && ax < hx-3) {
			return mAmber
		}
		return nil
	})
	s.Add(slab)
	return s
}

// lampScene is a street lamp standing at (x, y).
func lampScene(x, y float64) *Scene {
	s := &Scene{}
	s.Add(Cylinder(V3{x, y, 0}, V3{x, y, 1}, 1.4, mPanel))
	s.Add(Cylinder(V3{x, y, 1}, V3{x, y, 11}, 0.55, mPanel))
	s.Add(Block(V3{x, y, 11.6}, V3{1.6, 1.6, 0.6}, Ident, 0.4, 0, mPanel))
	s.Add(Block(V3{x, y, 10.7}, V3{1.1, 1.1, 0.4}, Ident, 0, 0, mWindow))
	return s
}

// LampPos lists the street lamps of a location (world pixels, on the ground).
func LampPos(loc *game.Location) []V3 {
	st := loc.Streets()[0]
	hq := loc.HQCenter()
	var out []V3
	var stops []float64
	for _, b := range loc.Buildings {
		stops = append(stops, loc.Junction(b).X)
	}
	for x := st[0].X + 1.2; x <= st[1].X-1; x += 1.9 {
		if math.Abs(x-hq.X) < 2.4 {
			continue
		}
		ok := true
		for _, j := range stops {
			if math.Abs(x-j) < 0.8 {
				ok = false
			}
		}
		if !ok {
			continue
		}
		for _, side := range []float64{-1, 1} {
			out = append(out, V3{x * CellPx, (loc.Avenue + side*0.47) * CellPx, 0})
		}
	}
	return out
}

// ColonyTerrain renders the static ground of a colony location: the plateau
// with paved streets, the lot foundations, street lamps, rocks and the
// shadow of the command center.
func ColonyTerrain(loc *game.Location) (*Canvas, *TerrainInfo) {
	W, H := loc.W*CellPx, loc.H*CellPx
	ti := &TerrainInfo{OY: topPad, W: W, H: H, hg: make([]float32, W*H)}
	CW := W
	CH := topPad + int(math.Ceil(float64(H)*GY+cliffMax*HZ)) + 2
	c := NewCanvas(CW, CH)
	bio := colonyBiome(loc)

	dist := make([]float64, W*H)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	for i, st := range loc.Streets() {
		hw := spurHW
		if i == 0 {
			hw = avenueHW
		}
		streetSeg(dist, W, H, st[0], st[1], hw)
	}
	type box struct{ x0, y0, x1, y1 float64 }
	var boxes []box
	hq := loc.HQ
	boxes = append(boxes, box{float64(hq.X) * CellPx, float64(hq.Y) * CellPx, float64(hq.X+3) * CellPx, float64(hq.Y+3) * CellPx})
	for _, b := range loc.Buildings {
		boxes = append(boxes, box{float64(b.Lot.X) * CellPx, float64(b.Lot.Y) * CellPx, float64(b.Lot.X+b.W) * CellPx, float64(b.Lot.Y+b.H) * CellPx})
	}
	inBox := func(x, y, m float64) bool {
		for _, b := range boxes {
			if x > b.x0-m && x < b.x1+m && y > b.y0-m && y < b.y1+m {
				return true
			}
		}
		return false
	}

	// Decorations away from the streets and lots.
	r := rand.New(rand.NewSource(77))
	type crater struct{ x, y, r float64 }
	var craters []crater
	var placed [][2]float64
	var scenes []struct {
		y float64
		s *Scene
	}
	add := func(y float64, s *Scene) {
		scenes = append(scenes, struct {
			y float64
			s *Scene
		}{y, s})
	}
	free := func(x, y, rad float64) bool {
		if x < 6 || y < 6 || x >= float64(W)-6 || y >= float64(H)-6 {
			return false
		}
		for oy := -1.0; oy <= 1; oy++ {
			for ox := -1.0; ox <= 1; ox++ {
				px, py := int(x+ox*rad), int(y+oy*rad)
				if px >= 0 && py >= 0 && px < W && py < H && dist[py*W+px] < PathHalfWidth+8 {
					return false
				}
			}
		}
		if inBox(x, y, 7+rad) {
			return false
		}
		for _, p := range placed {
			if math.Hypot(p[0]-x, p[1]-y) < 16+rad {
				return false
			}
		}
		return true
	}
	for i := 0; i < 260; i++ {
		x, y := r.Float64()*float64(W), r.Float64()*float64(H)
		k := r.Float64()
		rad := 4.0
		if k < 0.24 {
			rad = 7
		}
		if !free(x, y, rad) {
			continue
		}
		placed = append(placed, [2]float64{x, y})
		switch {
		case k < 0.18:
			craters = append(craters, crater{x, y, 4.5 + r.Float64()*3})
		case k < 0.24:
			add(y, boulderScene(r, x, y, bio))
		case k < 0.58:
			add(y, rockScene(r, x, y, bio))
		case k < 0.72:
			add(y, crystalScene(r, x, y, bio))
		default:
			add(y, shrubScene(r, x, y, bio))
		}
	}
	for _, b := range loc.Buildings {
		cc := b.Center()
		add(cc.Y*CellPx-float64(b.H)*CellPx/2, slabScene(cc.X*CellPx, cc.Y*CellPx, b.W, b.H))
	}
	for _, p := range LampPos(loc) {
		add(p.Y, lampScene(p.X, p.Y))
	}

	// Height field: streets a little below the ground, craters.
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			h := 0.0
			if d := dist[y*W+x]; d < PathHalfWidth+3 && !inBox(fx, fy, -2) {
				h = -streetDepth * (1 - smoothstep(PathHalfWidth-1.2, PathHalfWidth+2.2, d))
			}
			for _, cr := range craters {
				d := math.Hypot(fx-cr.x, fy-cr.y)
				if d > cr.r+5 {
					continue
				}
				if d < cr.r {
					k := d / cr.r
					h -= 1.8 * (1 - k*k)
				}
				h += 1.1 * math.Exp(-sq((d-cr.r)/1.7))
			}
			ti.hg[y*W+x] = float32(h)
		}
	}

	objs := make([]decor, len(scenes))
	Parallel(len(scenes), func(i int) {
		objs[i] = decor{scenes[i].y, scenes[i].s.Bake(Opts{Inner: 2.5, Shadows: true}, true, 0, 0)}
	})
	shade := make([]bool, CW*CH)
	for _, o := range objs {
		markShadow(shade, CW, CH, o.sp.Sh, -o.sp.SX, topPad-o.sp.SY)
	}
	base := BaseScene(true)
	bc := loc.HQCenter()
	Shift(base, V3{bc.X * CellPx, bc.Y * CellPx, 0})
	x0, y0, x1, y1 := base.shadowBounds()
	bv := View{W: int(x1-x0) + 4, H: int(y1-y0) + 4, OX: -math.Floor(x0) + 2, OY: -math.Floor(y0) + 2}
	markShadow(shade, CW, CH, base.ShadowMask(bv), -int(bv.OX), topPad-int(bv.OY))

	paintGround(c, ti, bio, dist, shade)
	stampDecor(c, objs)
	return c, ti
}

// Hologram turns a sprite into the cyan projection of a building still to
// be built: brightness from the model, every other row faint.
func Hologram(src *Canvas) *Canvas {
	out := NewCanvas(src.W, src.H)
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			p := src.Get(x, y)
			if p.A == 0 {
				continue
			}
			l := (0.3*float64(p.R) + 0.59*float64(p.G) + 0.11*float64(p.B)) / 255
			k := 0.45 + 0.55*l
			if y%2 == 1 {
				k *= 0.55
			}
			// Edges glow brighter.
			edge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if q := src.Get(x+d[0], y+d[1]); q.A == 0 {
					edge = true
				}
			}
			if edge {
				k = 1
			}
			out.Set(x, y, Color{R: uint8(44 * k), G: uint8(232 * k), B: uint8(245 * k), A: 255})
		}
	}
	return out
}
