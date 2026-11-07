package art

import "math"

// Europa's buildings use cold metal with warm windows and warning stripes.
var (
	mPolar      = mat(Ramp{Navy, Slate, Steel, Gray, Silver, White}, 0.04, 0.4)
	mPolarDark  = mat(Ramp{Ink, Navy, Slate, Steel, Gray}, -0.02, 0.3)
	mFrostGlass = mat(Ramp{Ink, DeepBlue, Blue, Cyan, White}, -0.04, 0.85)
	mHeat       = glow(Ramp{Rust, Orange, Amber, Yellow}, -0.12)
)

func polarHabitatScene(stage int) *Scene {
	s := habitatScene(stage)
	// Insulating ribs and a warm light distinguish Europa's homes.
	s.Add(Block(V3{-8, -5, 14.5}, V3{12, 2, 1}, Ident, 1, 0, mPolar))
	s.Add(Cylinder(V3{-21, -5, 2}, V3{-21, -5, 13}, 1.1, mPolarDark))
	s.Add(Block(V3{-21, -5, 13.5}, V3{2, 2, 1}, Ident, 0, 0, mHeat))
	if stage >= 2 {
		s.Add(Block(V3{13, 10, 10}, V3{8, 1.3, 0.8}, Ident, 1, 0, mPolar))
	}
	return s
}

func iceDrillScene(stage int) *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 2}, V3{23, 19, 2}, Ident, 2, 0, mPolarDark))
	s.Add(Lathe(V3{0, -3, 4}, 10, 8, 5, 20, Ident, mTrim))
	s.Add(Cylinder(V3{0, -3, 9}, V3{0, -3, 33}, 3.8, mPolar))
	for z := 11.0; z < 32; z += 5 {
		s.Add(Lathe(V3{0, -3, z}, 4.6, 4.6, 0.8, 16, Ident, mTrim))
	}
	for _, x := range []float64{-11, 11} {
		s.Add(Rod(V3{x, -11, 4}, V3{x * 0.35, -3, 39}, 0.9, mPolarDark))
		s.Add(Rod(V3{x, 7, 4}, V3{x * 0.35, -3, 39}, 0.9, mPolarDark))
	}
	s.Add(Block(V3{0, -3, 39}, V3{6, 5, 2}, Ident, 1, 0, mTrim))
	s.Add(Sphere(V3{0, -3, 42}, 1.3, mWindow))
	s.Add(Block(V3{-16, 10, 7}, V3{5, 5, 5}, Ident, 1, 1, mPolar))
	if stage >= 2 {
		s.Add(Lathe(V3{17, -8, 4}, 6, 6, 12, 20, Ident, mPolar))
		s.Add(dome(V3{17, -8, 16}, 6, 3, mFrostGlass))
		s.Add(Cylinder(V3{9, -8, 9}, V3{16, -8, 9}, 1.4, mTrim))
		s.Add(Block(V3{18, 10, 7}, V3{5, 5, 3}, Ident, 1, 0, mPolar))
	}
	return s
}

func researchLabScene(stage int) *Scene {
	s := &Scene{}
	s.Add(Block(V3{-5, 0, 7}, V3{18, 13, 7}, Ident, 2, 3, mPolar))
	s.Add(Block(V3{-5, 0, 15}, V3{16, 11, 1}, Ident, 1, 0, mPolarDark))
	for _, x := range []float64{-15, -8, -1, 6} {
		s.Add(Block(V3{x, 13.2, 9}, V3{2, 0.5, 2}, Ident, 0, 0, mFrostGlass))
	}
	s.Add(Block(V3{-10, 13.5, 3}, V3{2, 0.5, 3}, Ident, 0, 0, mWindow))
	if stage >= 2 {
		s.Add(Lathe(V3{17, -8, 0}, 9, 9, 8, 24, Ident, mPolarDark))
		s.Add(dome(V3{17, -8, 8}, 9, 9, mFrostGlass))
		s.Add(Cylinder(V3{17, -8, 17}, V3{17, -8, 28}, 0.8, mPolarDark))
		s.Add(Sphere(V3{17, -8, 29}, 1.2, mRedG))
		s.Add(Cylinder(V3{8, -8, 4}, V3{15, -8, 4}, 2.4, mPolar))
	}
	return s
}

func thermalPlantScene(stage int) *Scene {
	s := &Scene{}
	generator := func(x float64) {
		s.Add(Block(V3{x, 0, 6}, V3{9, 11, 6}, Ident, 1.5, 2, mPolarDark))
		s.Add(Lathe(V3{x, -9, 11}, 4, 3, 14, 16, Ident, mPolar))
		s.Add(Lathe(V3{x, -9, 25}, 4, 4, 2, 16, Ident, mTrim))
		s.Add(Block(V3{x, 11.3, 8}, V3{5, 0.4, 2}, Ident, 0, 0, mHeat))
		s.Add(Block(V3{x, 0, 12.5}, V3{6, 6, 0.8}, Ident, 1, 0, mPolar))
	}
	generator(-14)
	if stage >= 2 {
		generator(14)
		s.Add(Cylinder(V3{-5, 2, 5}, V3{5, 2, 5}, 2, mTrim))
	}
	return s
}

func wyrmMonumentScene() *Scene {
	s := &Scene{}
	s.Add(Block(V3{0, 0, 3}, V3{31, 26, 3}, Ident, 6, 1, mStoneLo))
	s.Add(Block(V3{0, 0, 8}, V3{25, 21, 2.4}, Ident, 4, 1, mStone))
	// The head rises out of a curling, plated body, like the 2-7 boss.
	w := &Scene{}
	w.Add(WyrmPartScene(WyrmTail, -0.6, V3{-16, -4, 0}).Parts...)
	w.Add(WyrmPartScene(WyrmSegS, 0.0, V3{-8, -7, 0}).Parts...)
	w.Add(WyrmPartScene(WyrmSegM, 0.5, V3{0, -8, 0}).Parts...)
	w.Add(WyrmPartScene(WyrmSegL, 1.0, V3{7, -2, 0}).Parts...)
	w.Add(WyrmHeadScene(math.Pi/2, 3, 8).Parts...)
	bronze(w)
	Shift(w, V3{0, 2, 10.4})
	s.Add(w.Parts...)
	for _, x := range []float64{-27, 27} {
		s.Add(Cylinder(V3{x, 23, 6}, V3{x, 23, 12}, 0.7, mPanel))
		s.Add(Sphere(V3{x, 23, 12.6}, 1.4, mCyanG))
	}
	return s
}
