package art

import "math"

// Ice Wyrm materials: steel-blue armour plates over dark teal flesh, ice
// crystals along the spine, glowing cyan eyes and maw.
var (
	mWyrmPlate = mat(Ramp{Ink, Hex(0x13304a), Hex(0x1b5872), Hex(0x2a8fa6), Hex(0x5fd0de), Hex(0xbdf6f7)}, -0.04, 0.6)
	mWyrmSkin  = mat(Ramp{Ink, Ink, Navy, Hex(0x2b3466)}, 0, 0.2)
	mWyrmBelly = mat(Ramp{Navy, Slate, Steel, Gray, Silver}, -0.04, 0.3)
	mWyrmIce   = mat(Ramp{Blue, Cyan, Hex(0xa9f3ff), White, White}, 0.1, 0.9)
	mWyrmEye   = glow(Ramp{Ochre, Amber, Yellow, White}, 0.05)
	mWyrmMaw   = glow(Ramp{Navy, DeepBlue, Blue, Cyan}, -0.05)
	mWyrmTooth = mat(Ramp{Steel, Gray, Silver, White, White}, 0.12, 0.6)
)

// Wyrm body parts, from the neck to the tip of the tail.
const (
	WyrmSegL = iota // the thick segments behind the head
	WyrmSegM
	WyrmSegS
	WyrmTail
	WyrmKinds
)

// WyrmBody lists the parts behind the head: their kind and their distance
// behind the head along the road (cells).
var WyrmBody = []struct {
	Kind int
	Back float64
}{
	{WyrmSegL, 0.44}, {WyrmSegL, 0.74}, {WyrmSegM, 1.03}, {WyrmSegM, 1.31},
	{WyrmSegM, 1.58}, {WyrmSegS, 1.84}, {WyrmSegS, 2.08}, {WyrmTail, 2.34},
}

var wyrmRadius = [WyrmKinds]float64{8.6, 7.4, 6.2, 4.6}

// WyrmPartScene builds one body part turned to heading, placed at off.
func WyrmPartScene(kind int, heading float64, off V3) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), off, 1}
	R := x.R
	r := wyrmRadius[kind]
	if kind == WyrmTail {
		s.Add(Ellipsoid(x.p(V3{-1.5, 0, 4.4}), x.v(V3{7.5, 4.6, 4.0}), R.Mul(Pitch(0.12)), mWyrmSkin))
		s.Add(Ellipsoid(x.p(V3{-1.2, 0, 5.6}), x.v(V3{6.6, 3.9, 3.0}), R.Mul(Pitch(0.12)), mWyrmPlate))
		// A crystal fin sweeping back from the tip.
		s.Add(Shard(x.p(V3{-4.5, 0, 6.2}), 1.3, 1.5, 6.5, R.Mul(Pitch(1.15)), mWyrmIce))
		s.Add(Shard(x.p(V3{0.5, 0, 7.8}), 1.0, 1.0, 3.2, R.Mul(Pitch(0.55)), mWyrmIce))
		return s
	}
	zc := r * 0.86
	s.Add(Ellipsoid(x.p(V3{0, 0, zc}), x.v(V3{r * 1.08, r * 0.9, r * 0.84}), R, mWyrmSkin))
	plate := Ellipsoid(x.p(V3{-0.9, 0, zc + r*0.2}), x.v(V3{r * 1.0, r * 0.88, r * 0.7}), R, mWyrmPlate)
	plate.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.05 {
			return mWyrmBelly // the pale underside below the armour
		}
		return nil
	})
	s.Add(plate)
	// The spine crystal, leaning back, and two smaller ones on big segments.
	top := zc + r*0.2 + r*0.6
	s.Add(Shard(x.p(V3{-0.8, 0, top}), r*0.2, r*0.25, r*0.72, R.Mul(Pitch(0.5)), mWyrmIce))
	if kind == WyrmSegL {
		for _, side := range []float64{-1, 1} {
			s.Add(Shard(x.p(V3{-0.4, side * r * 0.55, top - r*0.3}), r*0.12, r*0.15, r*0.42, R.Mul(Roll(-side*0.7)).Mul(Pitch(0.45)), mWyrmIce))
		}
	}
	return s
}

// WyrmHeadScene is the head of the Ice Wyrm at animation frame f of n (the
// jaws open and close).
func WyrmHeadScene(heading float64, f, n int) *Scene { return wyrmHead(heading, f, n, V3{}) }

func wyrmHead(heading float64, f, n int, off V3) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), off, 1.12}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	jaw := 0.1 + 0.26*(0.5-0.5*math.Cos(phase))
	// Neck.
	s.Add(Ellipsoid(x.p(V3{-4.5, 0, 8.2}), x.v(V3{6.8, 8.4, 7.6}), R, mWyrmSkin))
	neck := Ellipsoid(x.p(V3{-5.2, 0, 9.8}), x.v(V3{6.2, 8.0, 6.4}), R, mWyrmPlate)
	neck.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.05 {
			return mWyrmBelly
		}
		return nil
	})
	s.Add(neck)
	// Skull and a long snout.
	skull := Ellipsoid(x.p(V3{2.5, 0, 11.2}), x.v(V3{9.6, 7.4, 5.6}), R.Mul(Pitch(-0.1)), mWyrmPlate)
	skull.WithPaint(func(p, n V3) *Mat {
		if n.Z < -0.25 {
			return mWyrmBelly
		}
		return nil
	})
	s.Add(skull)
	s.Add(Ellipsoid(x.p(V3{11.5, 0, 9.4}), x.v(V3{6.6, 5.0, 3.4}), R.Mul(Pitch(-0.22)), mWyrmPlate))
	// The glowing maw behind the teeth.
	s.Add(Ellipsoid(x.p(V3{8.5, 0, 7.6 - jaw*3}), x.v(V3{6.0, 3.8, 1.6 + jaw*3.4}), R, mWyrmMaw))
	// Upper fangs.
	for _, t := range [][2]float64{{15.0, 1.9}, {12.6, 3.2}, {10.0, 4.0}} {
		for _, side := range []float64{-1, 1} {
			s.Add(Shard(x.p(V3{t[0], side * t[1], 7.8}), x.r(0.6), x.r(0.3), x.r(2.3), R.Mul(Pitch(math.Pi)), mWyrmTooth))
		}
	}
	// Lower jaw, hinged under the skull.
	pivot := V3{1.5, 0, 6.6}
	J := R.Mul(Pitch(-jaw))
	jp := func(v V3) V3 { return x.T.Add(J.Apply(v.Mul(x.K))).Add(R.Apply(pivot.Mul(x.K))) }
	s.Add(Ellipsoid(jp(V3{7.4, 0, -1.2}), x.v(V3{8.4, 5.2, 2.3}), J, mWyrmBelly))
	for _, t := range [][2]float64{{13.4, 1.6}, {10.8, 2.9}} {
		for _, side := range []float64{-1, 1} {
			s.Add(Shard(jp(V3{t[0], side * t[1], 0.4}), x.r(0.55), x.r(0.3), x.r(2.0), J, mWyrmTooth))
		}
	}
	// Glowing eyes under a brow of ice.
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{8.8, side * 4.5, 12.0}), x.r(1.8), mWyrmEye))
		s.Add(Ellipsoid(x.p(V3{7.6, side * 3.9, 13.9}), x.v(V3{3.6, 1.7, 1.1}), R.Mul(Yaw(-side*0.35)), mWyrmPlate))
		// Long horns of ice, swept back.
		s.Add(Shard(x.p(V3{0.8, side * 4.0, 14.6}), x.r(1.6), x.r(2.0), x.r(9.0), R.Mul(Roll(-side*0.5)).Mul(Pitch(1.2)), mWyrmIce))
	}
	// A crest of crystals down the neck.
	s.Add(Shard(x.p(V3{-2.5, 0, 16.0}), x.r(1.2), x.r(1.4), x.r(4.2), R.Mul(Pitch(0.65)), mWyrmIce))
	s.Add(Shard(x.p(V3{-6.5, 0, 15.6}), x.r(1.4), x.r(1.5), x.r(5.0), R.Mul(Pitch(0.55)), mWyrmIce))
	return s
}

// WyrmPortraitScene is the whole wyrm curled up behind its head, which
// faces the viewer.
func WyrmPortraitScene() *Scene {
	s := &Scene{}
	const P = CellPx
	h0 := PortraitHeading
	// The body curls back from the head in a tightening arc.
	at := func(back float64) (V3, float64) {
		var x, y float64
		a := h0
		const steps = 32
		for i := 0; i < steps; i++ {
			t := back * (float64(i) + 0.5) / steps
			a = h0 + 1.25*t
			x -= math.Cos(a) * back * P / steps
			y -= math.Sin(a) * back * P / steps
		}
		return V3{x, y, 0}, a
	}
	for i := len(WyrmBody) - 1; i >= 0; i-- {
		b := WyrmBody[i]
		p, a := at(b.Back)
		s.Add(WyrmPartScene(b.Kind, a, p).Parts...)
	}
	s.Add(wyrmHead(h0, 3, 8, V3{}).Parts...)
	return s
}

// WyrmPart bakes one body part with its ground shadow.
func WyrmPart(kind int, heading float64) *Sprite {
	return WyrmPartScene(kind, heading, V3{}).Bake(Opts{Inner: 3, Shadows: true}, true, 0, 0)
}
