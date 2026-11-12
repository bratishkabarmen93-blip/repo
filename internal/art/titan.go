package art

import "math"

// ---- Titan (chapter 3) ----------------------------------------------------------
//
// Saturn's hazy moon: dark dunes of hydrocarbon sand, a road of packed tholin
// dust, cobbles of water ice and pale crystals of frozen methane. Its
// creatures are a methane sack (the Bloater), a pale hunter that hides in the
// fog (the Stalker) and the Kraken of the methane sea.

var titanBiome = &biome{
	ground:  Ramp{Ink, Hex(0x2b2519), Hex(0x3f3622), Hex(0x55492c), Hex(0x6b5c36), Hex(0x827042)},
	road:    Ramp{Hex(0x2b2519), Hex(0x5c4728), Hex(0x7e6133), Hex(0x9f7d40), Hex(0xbd9a55), Hex(0xd9b870)},
	rim:     [2]Color{Hex(0x9f7d40), Hex(0x6b5c36)},
	cliff:   Ramp{Ink, Hex(0x221d16), Hex(0x362e22), Hex(0x4c412d), Hex(0x635539)},
	rock:    mat(Ramp{Ink, Hex(0x3a3942), Hex(0x6a676c), Hex(0x9f9a94), Hex(0xcdc5b6)}, 0, 0.2),
	crystal: mat(Ramp{Navy, DeepTeal, Hex(0x3f8580), Hex(0x7cc4b4), Hex(0xd0f2e6)}, 0.02, 0.8),
	shrub:   mat(Ramp{Ink, Hex(0x2c3022), Hex(0x464a2c), Hex(0x646638)}, -0.05, 0.2),
	bulb:    mat(Ramp{Plum, Rust, Ochre, Amber}, 0.05, 0.6),
}

var (
	mBloat    = mat(Ramp{Ink, Plum, Rust, Ochre, Amber, Yellow}, -0.02, 0.5)
	mBloatSp  = mat(Ramp{Ink, Plum, Brown, Rust, Ochre}, -0.06, 0.3)
	mBloatHd  = mat(Ramp{Ink, Plum, Brown, Rust, Ochre}, -0.04, 0.35)
	mBloatLeg = mat(Ramp{Ink, Plum, Brown, Rust}, -0.08, 0.15)
	mGasG     = glow(Ramp{Ochre, Amber, Yellow, White}, -0.15)

	mStalk   = mat(Ramp{Navy, DeepTeal, Hex(0x4f8682), Hex(0x8cc0b4), Hex(0xc8e8dc), White}, 0.06, 0.45)
	mStalkDk = mat(Ramp{Ink, DeepTeal, Hex(0x3e6a6c), Hex(0x6a9a94), Hex(0x9cc8bc)}, 0, 0.3)
	mStalkG  = glow(Ramp{DeepTeal, Cyan, White}, -0.1)

	mKraken    = mat(Ramp{Ink, Plum, Purple, Magenta, Pink, Peach}, -0.05, 0.45)
	mKrakenArm = mat(Ramp{Ink, Plum, Purple, Magenta, Pink}, -0.08, 0.4)
	mKrakenDk  = mat(Ramp{Ink, Ink, Plum, Purple, Magenta}, -0.06, 0.35)
	mKrakenSk  = mat(Ramp{Plum, Magenta, Pink, Peach, Sand}, 0, 0.3)
	mKrakenG   = glow(Ramp{Rust, Orange, Amber, Yellow, White}, -0.1)

	mBeaconG    = glow(Ramp{Ochre, Amber, Yellow, White}, 0.05)
	mBeaconTrim = mat(Ramp{Ink, Brown, Ochre, Amber, Yellow}, 0, 0.4)
)

// BloaterScene is a slow sack of methane waddling on four stubby legs.
func BloaterScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.5}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	sw := 1 + 0.045*math.Sin(phase*2) // the sack swells as it breathes
	bob := 0.3 * math.Abs(math.Sin(phase))
	for li, hx := range []float64{2.0, -2.2} {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			fw, up := gait(phase, g, 1.3, 1.0)
			hip := V3{hx, side * 2.8, 3.6 + bob}
			foot := V3{hx*1.2 + fw, side * 4.6, 0.5 + up}
			knee := V3{(hx+foot.X)/2 + 0.2*fw, side * 4.5, 3.2 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.8), mBloatLeg))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.65), mBloatLeg))
		}
	}
	sack := Ellipsoid(x.p(V3{-0.4, 0, 7.4 + bob}), x.v(V3{5.4 * sw, 4.8 * sw, 4.4 * sw}), R, mBloat)
	sack.WithPaint(func(p, n V3) *Mat {
		if math.Sin(p.X*1.3+p.Z*0.7)*math.Sin(p.Y*1.5-p.Z*0.4) > 0.45 {
			return mBloatSp
		}
		return nil
	})
	s.Add(sack)
	s.Add(Ellipsoid(x.p(V3{4.7, 0, 5.2 + bob}), x.v(V3{1.9, 1.9, 1.5}), R, mBloatHd))
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{6.1, side * 0.85, 5.7 + bob}), x.r(0.5), mAmberG))
	}
	// Gas vents on the back glow with methane.
	for i := 0; i < 3; i++ {
		a := float64(i)/3*2*math.Pi + 0.5
		s.Add(Sphere(x.p(V3{-0.6 + math.Cos(a)*2.3, math.Sin(a) * 2.3, 7.4 + bob + 4.0*sw}), x.r(0.8), mGasG))
	}
	return s
}

// StalkerScene is a lean hunter on four long legs, its flat head held low,
// its eyes and the spines along its back glowing faintly.
func StalkerScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.3}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	bob := 0.4 * math.Abs(math.Sin(phase))
	type leg struct{ hx, fx, fy float64 }
	legs := []leg{{2.0, 5.2, 4.2}, {-2.2, -4.6, 4.4}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			fw, up := gait(phase, g, 2.6, 1.8)
			hip := V3{lg.hx, side * 1.3, 5.0 + bob}
			foot := V3{lg.fx + fw, side * lg.fy, 0.5 + up}
			knee := V3{(lg.hx+foot.X)/2 + 0.4*fw, side * (lg.fy*0.7 + 0.6), 8.2 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.85), mStalkDk))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.7), mStalk))
		}
	}
	s.Add(Ellipsoid(x.p(V3{0, 0, 5.4 + bob}), x.v(V3{4.2, 2.3, 1.9}), R.Mul(Pitch(-0.08)), mStalk))
	tail := []V3{{-3.2, 0, 5.4}, {-5.8, 0.6 * math.Sin(phase), 5.9}, {-8.0, 1.2 * math.Sin(phase-0.8), 6.8}}
	for i := 0; i+1 < len(tail); i++ {
		a, b := tail[i], tail[i+1]
		a.Z += bob
		b.Z += bob
		s.Add(Rod(x.p(a), x.p(b), x.r(0.9-0.25*float64(i)), mStalkDk))
	}
	s.Add(Rod(x.p(V3{3.0, 0, 5.9 + bob}), x.p(V3{4.6, 0, 6.4 + bob}), x.r(1.0), mStalk))
	s.Add(Ellipsoid(x.p(V3{5.8, 0, 6.3 + bob}), x.v(V3{2.3, 1.45, 1.15}), R.Mul(Pitch(0.12)), mStalk))
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{7.3, side * 0.75, 6.95 + bob}), x.r(0.45), mStalkG))
	}
	for i := 0; i < 3; i++ {
		px := 1.8 - 1.7*float64(i)
		s.Add(Rod(x.p(V3{px, 0, 6.8 + bob}), x.p(V3{px - 0.9, 0, 8.8 + bob - 0.3*float64(i)}), x.r(0.5), mStalkDk))
		s.Add(Sphere(x.p(V3{px - 0.9, 0, 8.8 + bob - 0.3*float64(i)}), x.r(0.55), mStalkG))
	}
	return s
}

// KrakenScene is the boss of Titan: a huge cephalopod hauling itself along on
// eight tentacles, its mantle raised behind its eyes.
func KrakenScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.7}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	heave := 0.5 * math.Sin(phase)
	for i := 0; i < 8; i++ {
		a := (float64(i) + 0.5) / 8 * 2 * math.Pi
		ph := phase + float64(i%2)*math.Pi // the arms pull in turns
		prev := V3{math.Cos(a) * 2.6, math.Sin(a) * 2.6, 3.4}
		const segs = 6
		for k := 1; k <= segs; k++ {
			t := float64(k) / segs
			reach := 2.6 + 9.5*t*(0.9+0.1*math.Cos(ph))
			b := a + 0.35*t*math.Sin(ph-t*2.2)
			z := 3.4*(1-t) + 0.7 + 1.6*t*t*math.Max(0, math.Sin(ph+1.5-t*2))
			p := V3{math.Cos(b) * reach, math.Sin(b) * reach, z}
			s.Add(Rod(x.p(prev), x.p(p), x.r(1.7*(1-t)+0.5), mKrakenArm))
			prev = p
		}
	}
	mantle := Ellipsoid(x.p(V3{-2.6, 0, 8.6 + heave}), x.v(V3{7.0, 5.4, 5.8}), R.Mul(Pitch(-0.4)), mKraken)
	mantle.WithPaint(func(p, n V3) *Mat {
		if math.Sin(p.X*0.9+p.Y*0.5)*math.Sin(p.Y*1.1-p.Z*0.6) > 0.6 {
			return mKrakenSk
		}
		return nil
	})
	s.Add(mantle)
	s.Add(Ellipsoid(x.p(V3{2.6, 0, 5.2 + heave*0.6}), x.v(V3{3.8, 4.8, 3.3}), R, mKrakenDk))
	for _, side := range []float64{-1, 1} {
		s.Add(Ellipsoid(x.p(V3{4.9, side * 2.8, 6.4 + heave*0.6}), x.v(V3{1.3, 1.1, 1.3}), R, mKrakenSk))
		s.Add(Sphere(x.p(V3{5.8, side * 3.0, 6.5 + heave*0.6}), x.r(0.8), mKrakenG))
	}
	// Glowing spots along the crest of the mantle.
	for i := 0; i < 4; i++ {
		px := -0.6 - 2.2*float64(i)
		s.Add(Sphere(x.p(V3{px, 0, 13.2 + heave - 0.5*float64(i)}), x.r(0.7), mKrakenG))
	}
	return s
}

// beaconLampZ is the height of a Beacon's lamp above the pivot.
func beaconLampZ(lv int) float64 { return 5.0 + 0.8*float64(min(lv, 2)) }

// beaconHead is the lamp of the Beacon: a mast with a big glass lantern, a
// reflector that sweeps around with the head and a small cap on top (kept
// narrow so that the lantern can be seen from above).
func beaconHead(s *Scene, x xf, lv int) {
	R := x.R
	trim := levelTrim("beacon", lv)
	h := beaconLampZ(lv)
	s.Add(Lathe(x.p(V3{0, 0, -0.4}), 4.8, 4.4, 1.6, 16, R, mDark))
	s.Add(Cylinder(x.p(V3{0, 0, 0.6}), x.p(V3{0, 0, h - 2.2}), 1.5, mMetal))
	s.Add(Lathe(x.p(V3{0, 0, h - 2.9}), 2.9, 2.4, 0.9, 16, R, trim))
	s.Add(Sphere(x.p(V3{0, 0, h}), 2.6, mBeaconG))
	s.Add(Block(x.p(V3{-2.6, 0, h - 0.3}), V3{0.45, 2.2, 1.8}, R, 0.6, 0.3, mSilver))
	s.Add(Lathe(x.p(V3{0, 0, h + 2.3}), 1.5, 0.5, 0.9, 12, R, mDark))
	s.Add(Cylinder(x.p(V3{0, 0, h + 3.1}), x.p(V3{0, 0, h + 4.4}), 0.35, mMetal))
	if lv >= 1 {
		// A cage of two arches around the lantern.
		for _, sg := range []float64{-1, 1} {
			s.Add(Rod(x.p(V3{0.4, sg * 2.7, h - 2.2}), x.p(V3{0.4, sg * 2.7, h + 1.0}), 0.4, trim))
			s.Add(Rod(x.p(V3{0.4, sg * 2.7, h + 1.0}), x.p(V3{0, sg * 1.1, h + 2.5}), 0.4, trim))
		}
	}
	if lv >= 2 {
		s.Add(Sphere(x.p(V3{0, 0, h + 4.7}), 0.7, mRedG))
	}
	if lv == 3 {
		// Overdrive: a ring of light around the lantern.
		ring := Lathe(x.p(V3{0, 0, h - 0.5}), 3.8, 3.8, 0.7, 20, R, turretStyles["beacon"].glow)
		ring.Sub(mDark, Cylinder(x.p(V3{0, 0, h - 1.3}), x.p(V3{0, 0, h + 0.5}), 2.9, nil))
		s.Add(ring)
	}
}

// titanIcon draws the HUD icons of Titan's threats.
func titanIcon(c *Canvas, name string) {
	switch name {
	case "fog":
		c.Disc(3.6, 4.6, 3.0, 2.2, Tan, 1)
		c.Disc(7.0, 3.8, 3.3, 2.5, Sand, 1)
		c.Disc(5.2, 6.4, 4.6, 2.2, Peach, 1)
		c.Rect(1, 8, 9, 1, Copper)
		c.Rect(2, 10, 7, 1, Clay)
	case "kraken":
		c.Ball(5.5, 4.0, 3.8, 3.3, 0, FleshRamp, 0.1, 0)
		for i := 0; i < 4; i++ {
			x0 := 2.4 + float64(i)*2.1
			c.Rod(x0, 6.2, x0+(float64(i)-1.5)*0.9, 10.2, 0.75, FleshRamp, 0)
		}
		c.Set(4, 4, Amber)
		c.Set(7, 4, Amber)
	}
}
