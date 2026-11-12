package art

import "math"

// ---- The Titan colony ----------------------------------------------------------
//
// Sand-coloured hulls against the dark dunes and amber lamps in the haze: the
// colony lives off the methane sea and keeps the fog away with a giant Beacon.

var (
	mDuneHull = mat(Ramp{Ink, Brown, Hex(0x7a6650), Hex(0xa8927a), Hex(0xd2c2a8), Hex(0xf0e6d2)}, 0.04, 0.4)
	mDuneDark = mat(Ramp{Ink, Hex(0x2b2519), Hex(0x4a3f30), Hex(0x6b5c47), Hex(0x8a7a62)}, -0.02, 0.3)
	mFuelTank = mat(Ramp{Ink, Slate, Steel, Gray, Silver, White}, 0.06, 0.6)
	mBlimp    = mat(Ramp{Ink, Brown, Ochre, Hex(0xc8a060), Hex(0xe8cf98), Hex(0xfff2d6)}, 0.05, 0.45)
	mFlareG   = glow(Ramp{Rust, Orange, Amber, Yellow, White}, -0.05)
)

// Dune habitat: the habitat domes behind fences against the drifting sand,
// with sealing ribs and an amber lamp.
func duneHabitatScene(stage int) *Scene {
	s := habitatScene(stage)
	for _, f := range [][3]float64{{-22.5, -13, 1.2}, {-20.5, -18.5, 0.8}, {-15.5, -21.5, 0.35}} {
		s.Add(Block(V3{f[0], f[1], 3}, V3{2.6, 0.5, 3}, Yaw(f[2]), 0.3, 0, mDuneDark))
	}
	s.Add(Block(V3{-8, -5, 14.5}, V3{12, 2, 1}, Ident, 1, 0, mDuneHull))
	s.Add(Cylinder(V3{-21, -5, 2}, V3{-21, -5, 13}, 1.1, mDuneDark))
	s.Add(Sphere(V3{-21, -5, 13.8}, 1.4, mBeaconG))
	if stage >= 2 {
		s.Add(Block(V3{13, 10, 10}, V3{8, 1.3, 0.8}, Ident, 1, 0, mDuneHull))
	}
	return s
}

// Methane refinery: a distillation column, a pump house and a flare stack,
// then storage spheres full of fuel.
func methaneRefineryScene(stage int) *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 2}, V3{23, 19, 2}, Ident, 2, 0, mDuneDark))
	cx, cy := -9.0, -5.0
	s.Add(Lathe(V3{cx, cy, 4}, 6.5, 5.5, 4, 20, Ident, mTrim))
	col := Cylinder(V3{cx, cy, 8}, V3{cx, cy, 38}, 3.8, mDuneHull)
	col.WithPaint(windowBand(mWindow, 0, 0, -12, -10, 6, 0.18))
	s.Add(col)
	for z := 13.0; z < 37; z += 6 {
		s.Add(Lathe(V3{cx, cy, z}, 4.8, 4.8, 0.7, 16, Ident, mTrim))
	}
	s.Add(dome(V3{cx, cy, 38}, 3.8, 2.6, mDuneHull))
	s.Add(Rod(V3{cx + 3.9, cy + 1.2, 8}, V3{cx + 3.9, cy + 1.2, 38}, 0.4, mMetal))
	// The pump house and the pipe from the column.
	s.Add(Block(V3{8, 7, 5}, V3{7, 5, 3}, Ident, 1, 1, mDuneHull))
	s.Add(Block(V3{8, 12.2, 4.2}, V3{2.4, 0.4, 2.4}, Ident, 0, 0, mWindow))
	s.Add(Rod(V3{cx + 3, cy + 2.5, 26}, V3{3, 6, 8}, 1, mMetal))
	// The flare stack burns off what the column cannot use.
	s.Add(Cylinder(V3{14, -13, 4}, V3{14, -13, 42}, 1.3, mTrim))
	for _, d := range [][2]float64{{-5, 3}, {5, 3}, {0, -5}} {
		s.Add(Rod(V3{14 + d[0], -13 + d[1], 4}, V3{14, -13, 30}, 0.25, mMetal))
	}
	s.Add(Lathe(V3{14, -13, 42}, 1.9, 1.4, 1.6, 12, Ident, mMetal))
	s.Add(Sphere(V3{14, -13, 45}, 1.8, mFlareG))
	if stage >= 2 {
		for _, t := range [][3]float64{{-12, 11, 6}, {3, -11, 5}} {
			x, y, r := t[0], t[1], t[2]
			z := r + 4.5
			for _, d := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				s.Add(Rod(V3{x + d[0]*r*0.62, y + d[1]*r*0.62, 4}, V3{x + d[0]*r*0.45, y + d[1]*r*0.45, z}, 0.5, mMetal))
			}
			tank := Sphere(V3{x, y, z}, r, mFuelTank)
			tank.WithPaint(func(p, n V3) *Mat {
				if math.Abs(p.Z) < 0.55 {
					return mTrim
				}
				return nil
			})
			s.Add(tank)
			s.Add(Cylinder(V3{x, y, z + r - 0.3}, V3{x, y, z + r + 1.2}, 0.8, mTrim))
		}
		s.Add(Cylinder(V3{-12, 11, 5}, V3{1, 8, 5}, 0.8, mMetal))
		s.Add(Cylinder(V3{3, -11, 5}, V3{3, 2, 5}, 0.8, mMetal))
	}
	return s
}

// Fog lighthouse: a lattice tower with a Beacon lantern, then a lens ring,
// warning lights and a keeper's hut.
func fogLighthouseScene(stage int) *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 2.5}, V3{16, 16, 2.5}, Ident, 2, 1, mDuneDark))
	const z0, top = 5.0, 40.0
	legs := [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}
	leg := func(i int, z float64) V3 {
		t := (z - z0) / (top - z0)
		k := 11 + (3.6-11)*t
		return V3{legs[i][0] * k, legs[i][1] * k, z}
	}
	for i := range legs {
		s.Add(Rod(leg(i, z0), leg(i, top), 1, mTrim))
		j := (i + 1) % 4
		for _, zz := range [][2]float64{{z0, 16}, {16, 27}, {27, 38}} {
			s.Add(Rod(leg(i, zz[0]), leg(j, zz[1]), 0.4, mMetal))
			s.Add(Rod(leg(i, zz[1]), leg(j, zz[0]), 0.4, mMetal))
		}
	}
	s.Add(Block(V3{0, 0, top + 1}, V3{6, 6, 1}, Ident, 0.5, 0, mDuneHull))
	// The lantern: a Beacon's light, many times over.
	s.Add(Cylinder(V3{0, 0, top + 2}, V3{0, 0, top + 8}, 3.4, mBeaconG))
	for k := 0; k < 6; k++ {
		a := float64(k) / 6 * 2 * math.Pi
		x, y := 3.5*math.Cos(a), 3.5*math.Sin(a)
		s.Add(Rod(V3{x, y, top + 2}, V3{x, y, top + 8}, 0.35, mBeaconTrim))
	}
	s.Add(Lathe(V3{0, 0, top + 8}, 4.6, 0.6, 4, 16, Ident, mBeaconTrim))
	s.Add(Sphere(V3{0, 0, top + 12.4}, 0.8, mBeaconTrim))
	if stage >= 2 {
		s.Add(Lathe(V3{0, 0, top + 4.4}, 4.4, 4.4, 1.2, 20, Ident, mBeaconG))
		for _, d := range legs {
			s.Add(Sphere(V3{d[0] * 5.4, d[1] * 5.4, top + 2.6}, 0.8, mRedG))
		}
		s.Add(Block(V3{0, 14, 7}, V3{5, 2, 2}, Ident, 0.5, 1, mDuneHull))
		s.Add(Block(V3{0, 16.1, 6.6}, V3{1.4, 0.3, 1.5}, Ident, 0, 0, mWindow))
	}
	return s
}

// Airship dock: a mooring mast with cargo waiting, then a cargo airship tied
// to the mast.
func airshipDockScene(stage int) *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 2}, V3{22, 18, 2}, Ident, 2, 0, mDuneDark))
	mx, my := -15.0, -7.0
	s.Add(Lathe(V3{mx, my, 4}, 5, 4, 3, 16, Ident, mTrim))
	s.Add(Lathe(V3{mx, my, 7}, 2.6, 1.3, 29.5, 12, Ident, mDuneHull))
	s.Add(Lathe(V3{mx, my, 36.5}, 2.6, 0.8, 3.5, 12, Ident, mBeaconTrim))
	for _, d := range [][2]float64{{-1, -1}, {1, -1}, {0.2, 1.3}} {
		s.Add(Rod(V3{mx + d[0]*6, my + d[1]*6, 4}, V3{mx, my, 28}, 0.25, mMetal))
	}
	// The dock office with its fuel line to the mast, and cargo waiting.
	s.Add(Block(V3{-5, 11, 6.5}, V3{5, 3.5, 2.5}, Ident, 0.6, 1, mDuneHull))
	s.Add(Block(V3{-5, 14.6, 6.8}, V3{3, 0.2, 1}, Ident, 0, 0, mWindow))
	s.Add(Rod(V3{-8, 11, 9}, V3{-8, 11, 15}, 0.25, mMetal))
	s.Add(Cylinder(V3{-10, 8, 4.6}, V3{-14, -2.5, 4.6}, 0.6, mTrim))
	s.Add(Block(V3{12, 11, 6.5}, V3{4, 3, 2.5}, Ident, 0.4, 0, mDuneHull))
	s.Add(Block(V3{4.5, 12, 5.5}, V3{2.5, 2.5, 1.5}, Ident, 0.4, 0, mPanel))
	s.Add(Block(V3{12, 11, 10}, V3{2.5, 2, 1}, Ident, 0.3, 0, mPanel))
	if stage >= 2 {
		c := V3{3, my, 39}
		hull := Ellipsoid(c, V3{17, 6.5, 6.5}, Ident, mBlimp)
		hull.WithPaint(func(p, n V3) *Mat {
			if math.Abs(p.X+3) < 0.7 || math.Abs(p.X-7) < 0.7 {
				return mTrim
			}
			return nil
		})
		s.Add(hull)
		// The gondola under it, and the tail fins.
		s.Add(Block(V3{3, my, 31.6}, V3{6, 2.4, 1.8}, Ident, 0.8, 0, mDuneHull))
		s.Add(Block(V3{3, my + 2.45, 31.8}, V3{4.5, 0.2, 0.8}, Ident, 0, 0, mWindow))
		s.Add(Block(V3{17.5, my, 44}, V3{2.8, 0.35, 3.6}, Ident, 0.3, 0, mDuneHull))
		s.Add(Block(V3{17.5, my, 39}, V3{2.8, 9.5, 0.35}, Ident, 0.3, 0, mDuneHull))
	}
	return s
}

// Kraken monument: the Kraken cast in bronze on a stone pedestal.
func krakenMonumentScene() *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 3}, V3{31, 26, 3}, Ident, 6, 1, mStoneLo))
	s.Add(Block(V3{0, 0, 8}, V3{25, 21, 2.4}, Ident, 4, 1, mStone))
	k := KrakenScene(math.Pi/2, 1, 8)
	bronze(k)
	Shift(k, V3{0, 1, 10.4})
	s.Add(k.Parts...)
	for _, x := range []float64{-27, 27} {
		s.Add(Cylinder(V3{x, 23, 6}, V3{x, 23, 12}, 0.7, mPanel))
		s.Add(Sphere(V3{x, 23, 12.6}, 1.4, mBeaconG))
	}
	return s
}

// titanEffects adds the animated spots of the Titan buildings (model space,
// not turned yet); it reports whether the building is one of them.
func titanEffects(fx *BuildingFX, kind string, stage int) bool {
	switch kind {
	case "refinery":
		fx.Smoke = append(fx.Smoke, V3{14, -13, 47.5})
		fx.Amber = append(fx.Amber, V3{14, -13, 45})
		fx.Red = append(fx.Red, V3{-9, -5, 41.2})
		if stage >= 2 {
			fx.Green = append(fx.Green, V3{-12, 11, 17.8})
		}
	case "lighthouse":
		fx.Amber = append(fx.Amber, V3{0, 0, 45})
		if stage >= 2 {
			fx.Red = append(fx.Red, V3{0, 0, 53.4})
		}
	case "airship":
		fx.Red = append(fx.Red, V3{-15, -7, 40.4})
		if stage >= 2 {
			fx.Green = append(fx.Green, V3{18, -7, 48})
		}
	default:
		return false
	}
	return true
}
