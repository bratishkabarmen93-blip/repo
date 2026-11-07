package art

import "math"

// Enemy materials: aliens are organic and warm, robots wear rust-red armour
// with red optics so they never look like the player's blue-grey turrets.
var (
	mRunner   = mat(Ramp{Ink, Plum, Rust, Ochre, Amber, Yellow}, -0.04, 0.35)
	mRunnerSt = mat(Ramp{Ink, Plum, Brown, Rust}, -0.05, 0.2)
	mRunLeg   = mat(Ramp{Ink, Plum, Brown, Rust, Ochre}, -0.08, 0.1)
	mSwarm    = mat(Ramp{Ink, Plum, Purple, Magenta, Pink, Peach}, 0, 0.4)
	mSwarmHd  = mat(Ramp{Ink, Plum, Purple, Magenta}, -0.05, 0.3)
	mPlate    = mat(Ramp{Ink, Plum, Brown, Rust, Ochre, Tan}, -0.04, 0.35)
	mPlateHi  = mat(Ramp{Ink, Brown, Rust, Ochre, Tan, Sand}, -0.02, 0.4)
	mJoint    = mat(DarkMetal, -0.1, 0.25)
	mJointL   = mat(DarkMetal, 0.04, 0.35)
	mGuard    = mat(Ramp{Ink, Navy, Slate, Steel, Gray}, 0.02, 0.35)
	mDroneB   = mat(Ramp{Ink, Navy, Slate, Gray, Silver, White}, 0.02, 0.5)
	mRotor    = mat(Ramp{Ink, Navy, Slate}, 0, 0.1)
	mLimeG    = glow(Ramp{Forest, Green, Lime, Yellow}, -0.05)
	mAmberG   = glow(Ramp{Ochre, Amber, Yellow, White}, -0.1)
	mQueen    = mat(Ramp{Ink, Plum, Purple, Magenta, Pink, Peach}, -0.06, 0.35)
	mQueenHd  = mat(Ramp{Ink, Plum, Purple, Magenta, Pink}, -0.16, 0.4)
	mQueenSp  = mat(Ramp{Ink, Ink, Plum, Purple}, -0.05, 0.3)
	mEggG     = glow(Ramp{Forest, Green, Lime, Yellow}, -0.12)
)

// EnemyScene builds the model of an enemy type walking along heading at frame f of n.
func EnemyScene(id string, heading float64, f, n int) *Scene {
	switch id {
	case "runner":
		return RunnerScene(heading, f, n)
	case "swarmling":
		return SwarmlingScene(heading, f, n)
	case "walker":
		return WalkerScene(heading, f, n)
	case "drone":
		return DroneScene(heading, f, n)
	case "queen":
		return QueenScene(heading, f, n)
	case "beetle":
		return BeetleScene(heading, f, n)
	case "splitter":
		return SplitterScene(heading, f, n)
	case "shard":
		return ShardMiteScene(heading, f, n)
	case "stinger":
		return StingerScene(heading, f, n)
	case "wyrm":
		return WyrmHeadScene(heading, f, n)
	}
	return CrawlerScene(heading, f, n)
}

// EnemySprite bakes one animation frame with its ground shadow.
func EnemySprite(id string, heading float64, f, n int) *Sprite {
	o := Opts{Inner: 2.5, Shadows: true}
	if id == "queen" || id == "walker" || id == "wyrm" {
		o.Inner = 3
	}
	return EnemyScene(id, heading, f, n).Bake(o, true, 0, 0)
}

// PortraitHeading shows an enemy from the front-left, facing the viewer.
const PortraitHeading = 2.2

// EnemyPortrait renders an enemy so that it fits a w x h pixel box.
func EnemyPortrait(id string, w, h float64) *Sprite {
	s := EnemyScene(id, PortraitHeading, 0, 8)
	if id == "wyrm" {
		s = WyrmPortraitScene()
	}
	x0, y0, x1, y1 := s.bounds()
	// bounds() is conservative (bounding spheres), so measure the real sprite once.
	probe := s.BakeScaled(Opts{Inner: 2.5, Shadows: true}, 1)
	pw, ph := float64(probe.C.W), float64(probe.C.H)
	_ = x0 + y0 + x1 + y1
	k := math.Min(w/pw, h/ph)
	// Depth is measured in world units: a small icon samples the model more
	// coarsely, so it needs a larger jump to count as an inner edge.
	return s.BakeScaled(Opts{Inner: 2.5 / k, Shadows: true}, k)
}

// xk is a placement transform with a uniform model scale k.
type xk struct {
	R M3
	T V3
	K float64
}

func (x xk) p(v V3) V3           { return x.R.Apply(v.Mul(x.K)).Add(x.T) }
func (x xk) r(v float64) float64 { return v * x.K }
func (x xk) v(v V3) V3           { return v.Mul(x.K) }

// gait returns the swing (forward offset) and lift of a leg in a tripod gait.
func gait(phase float64, group int, swing, lift float64) (float64, float64) {
	sw, up := math.Sin(phase)*swing, math.Max(0, math.Cos(phase))*lift
	if group == 1 {
		sw, up = -sw, math.Max(0, -math.Cos(phase))*lift
	}
	return sw, up
}

// RunnerScene is a small, long-legged bug built for speed.
func RunnerScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.15}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	bob := 0.35 * math.Abs(math.Sin(phase))
	type leg struct{ hx, fx, fy float64 }
	legs := []leg{{1.6, 6.4, 5.0}, {0.3, 0.8, 6.4}, {-1.0, -5.2, 5.6}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 2.3, 1.7)
			hip := V3{lg.hx, side * 1.4, 4.0 + bob}
			foot := V3{lg.fx + sw, side * lg.fy, 0.5 + up}
			knee := V3{(lg.hx+foot.X)/2 + 0.3*sw, side * (lg.fy*0.6 + 0.9), 7.4 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.55), mRunLeg))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.45), mRunLeg))
		}
	}
	ab := Ellipsoid(x.p(V3{-3.9, 0, 4.4 + bob}), x.v(V3{4.4, 2.7, 2.4}), R.Mul(Pitch(-0.12)), mRunner)
	ab.WithPaint(func(p, n V3) *Mat {
		if d := math.Mod(p.X+20, 2.4); d < 0.8 && p.Z > 0.3 {
			return mRunnerSt
		}
		return nil
	})
	s.Add(ab)
	s.Add(Ellipsoid(x.p(V3{0.5, 0, 4.4 + bob}), x.v(V3{2.4, 2.1, 1.9}), R, mRunner))
	s.Add(Ellipsoid(x.p(V3{3.4, 0, 4.3 + bob}), x.v(V3{1.9, 1.8, 1.5}), R, mRunnerSt))
	for _, side := range []float64{-1, 1} {
		s.Add(Rod(x.p(V3{4.4, side * 0.8, 5.0 + bob}), x.p(V3{7.6, side * 2.6, 7.4 + bob}), x.r(0.32), mRunLeg))
		s.Add(Sphere(x.p(V3{4.5, side * 1.0, 4.9 + bob}), x.r(0.6), mYellowG))
	}
	return s
}

// SwarmlingScene is a tiny larva that comes in large numbers.
func SwarmlingScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.45}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	for li, hx := range []float64{1.0, -0.2, -1.4} {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 0.9, 0.8)
			hip := V3{hx, side * 1.2, 1.9}
			foot := V3{hx*1.4 + sw, side * 3.1, 0.35 + up}
			s.Add(Rod(x.p(hip), x.p(foot), x.r(0.42), mSwarmHd))
		}
	}
	wig := 0.25 * math.Sin(phase*2)
	body := Ellipsoid(x.p(V3{-0.8, 0, 2.3}), x.v(V3{3.1, 2.3, 1.9}), R.Mul(Yaw(wig)), mSwarm)
	body.WithPaint(func(p, n V3) *Mat {
		if d := math.Mod(p.X+20, 2.4); d < 0.7 && p.Z > 0.6 {
			return mSwarmHd
		}
		return nil
	})
	s.Add(body)
	s.Add(Sphere(x.p(V3{2.2, 0, 2.2}), x.r(1.5), mSwarmHd))
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{3.2, side * 0.7, 2.7}), x.r(0.45), mYellowG))
		s.Add(Rod(x.p(V3{3.3, side * 0.6, 1.6}), x.p(V3{4.4, side * 0.2, 1.2}), x.r(0.3), mBone))
	}
	return s
}

// ik2 places the joint of a two-bone limb from a to b with bone lengths l1, l2,
// bending towards the direction bend.
func ik2(a, b V3, l1, l2 float64, bend V3) V3 {
	d := b.Sub(a)
	dist := math.Min(d.Len(), l1+l2-1e-3)
	dn := d.Norm()
	t := (l1*l1 - l2*l2 + dist*dist) / (2 * dist)
	h := math.Sqrt(math.Max(0, l1*l1-t*t))
	perp := bend.Sub(dn.Mul(bend.Dot(dn))).Norm()
	return a.Add(dn.Mul(t)).Add(perp.Mul(h))
}

// WalkerScene is an armoured two-legged war robot with reverse-jointed legs.
func WalkerScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.3}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	bob := 0.6 * math.Abs(math.Cos(phase))
	hz := 10.6 - bob
	for gi, side := range []float64{-1, 1} {
		ph := phase + float64(gi)*math.Pi
		sw := math.Sin(ph) * 2.8
		up := math.Max(0, math.Cos(ph)) * 2.0
		hip := V3{-0.4, side * 3.8, hz}
		foot := V3{sw, side * 4.4, 1.0 + up}
		// Reverse joint: the knee points backwards.
		knee := ik2(hip, foot, 5.6, 6.2, V3{-1, 0, 0.2})
		s.Add(Rod(x.p(hip), x.p(knee), x.r(1.2), mJointL))
		s.Add(Rod(x.p(knee), x.p(foot), x.r(0.95), mJointL))
		s.Add(Sphere(x.p(knee), x.r(1.4), mPlate))
		s.Add(Block(x.p(foot.Add(V3{0.5, 0, -0.35})), x.v(V3{2.5, 1.5, 0.75}), R, x.r(0.6), x.r(0.4), mJoint))
	}
	tz := hz + 4.0
	torso := Block(x.p(V3{-0.3, 0, tz}), x.v(V3{4.6, 4.0, 3.3}), R, x.r(1.6), x.r(1.3), mPlate)
	torso.WithPaint(func(p, n V3) *Mat {
		if n.Z > 0.9 && math.Abs(p.Y) < 1.2 {
			return mJoint
		}
		return nil
	})
	s.Add(torso)
	for _, side := range []float64{-1, 1} {
		s.Add(Block(x.p(V3{-0.6, side * 4.7, tz + 0.9}), x.v(V3{2.6, 1.3, 2.1}), R, x.r(0.8), x.r(0.9), mPlateHi))
		s.Add(Cylinder(x.p(V3{0.6, side * 5.2, tz - 1.6}), x.p(V3{6.8, side * 5.2, tz - 1.6}), x.r(0.95), mJointL))
		m := Cylinder(x.p(V3{5.6, side * 5.2, tz - 1.6}), x.p(V3{7.6, side * 5.2, tz - 1.6}), x.r(1.25), mDarker)
		m.Sub(mBore, Cylinder(x.p(V3{7.0, side * 5.2, tz - 1.6}), x.p(V3{8.0, side * 5.2, tz - 1.6}), x.r(0.6), nil))
		s.Add(m)
	}
	s.Add(Block(x.p(V3{3.7, 0, tz - 0.5}), x.v(V3{1.3, 3.2, 2.6}), R, x.r(0.8), x.r(0.8), mPlateHi))
	s.Add(Block(x.p(V3{4.2, 0, tz + 2.0}), x.v(V3{1.4, 2.1, 1.1}), R, x.r(0.5), x.r(0.4), mJoint))
	s.Add(Block(x.p(V3{5.45, 0, tz + 2.05}), x.v(V3{0.35, 1.6, 0.4}), R, 0, 0, mRedG))
	s.Add(Rod(x.p(V3{-3.4, 2.2, tz + 2.6}), x.p(V3{-4.4, 2.5, tz + 6.2}), x.r(0.3), mJointL))
	s.Add(Sphere(x.p(V3{-4.4, 2.5, tz + 6.4}), x.r(0.6), mRedG))
	return s
}

// DroneScene is a hovering repair drone (origin under its body; the renderer
// lifts it to its flight height). Rotor blades turn with the frame.
func DroneScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.2}
	R := x.R
	spin := float64(f) / float64(n) * math.Pi
	const cz = 4.0
	for i := 0; i < 4; i++ {
		a := float64(i)/4*2*math.Pi + math.Pi/4
		hub := V3{math.Cos(a) * 6.2, math.Sin(a) * 6.2, cz + 0.5}
		s.Add(Rod(x.p(V3{math.Cos(a) * 2.6, math.Sin(a) * 2.6, cz}), x.p(hub), x.r(0.6), mJointL))
		ring := Lathe(x.p(hub.Add(V3{0, 0, -0.2})), x.r(3.3), x.r(3.3), x.r(0.8), 18, R, mGuard)
		ring.Sub(mRotor, Cylinder(x.p(hub.Add(V3{0, 0, -0.5})), x.p(hub.Add(V3{0, 0, 1.0})), x.r(2.6), nil))
		s.Add(ring)
		s.Add(Cylinder(x.p(hub.Add(V3{0, 0, -0.6})), x.p(hub.Add(V3{0, 0, 0.6})), x.r(0.8), mJoint))
		b := spin + float64(i)*0.8
		s.Add(Block(x.p(hub.Add(V3{0, 0, 0.35})), x.v(V3{2.5, 0.5, 0.12}), R.Mul(Yaw(b)), 0, 0, mRotor))
	}
	s.Add(Lathe(x.p(V3{0, 0, cz - 1.0}), x.r(4.0), x.r(3.6), x.r(1.0), 20, R, mJoint))
	body := Ellipsoid(x.p(V3{0, 0, cz}), x.v(V3{3.7, 3.5, 2.0}), R, mDroneB)
	body.WithPaint(func(p, n V3) *Mat {
		if n.Z > 0.5 && ((math.Abs(p.X) < 0.9 && math.Abs(p.Y) < 2.5) || (math.Abs(p.Y) < 0.9 && math.Abs(p.X) < 2.5)) {
			return mLimeG
		}
		return nil
	})
	s.Add(body)
	s.Add(Sphere(x.p(V3{0, 0, cz - 1.8}), x.r(1.2), mLimeG))
	s.Add(Sphere(x.p(V3{3.7, 0, cz + 0.1}), x.r(0.7), mAmberG))
	return s
}

// QueenScene is the hive queen boss: a huge egg-laying alien.
func QueenScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.2}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	breathe := 1 + 0.03*math.Sin(phase*2)
	type leg struct{ hx, fx, fy, kz float64 }
	legs := []leg{{5.4, 13.0, 9.0, 14.0}, {3.0, 4.0, 12.0, 15.0}, {0.4, -6.0, 11.0, 14.0}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 2.4, 2.2)
			hip := V3{lg.hx, side * 3.6, 9.0}
			foot := V3{lg.fx + sw, side * lg.fy, 0.8 + up}
			knee := V3{(lg.hx+foot.X)/2 + 0.3*sw, side * (lg.fy*0.66 + 2), lg.kz + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(1.35), mQueenHd))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(1.05), mQueenHd))
			s.Add(Shard(x.p(knee.Add(V3{0, 0, 0.6})), x.r(0.8), x.r(0.4), x.r(2.2), R, mQueenSp))
		}
	}
	ab := Ellipsoid(x.p(V3{-10.5, 0, 9.0}), x.v(V3{11.5 * breathe, 8.6 * breathe, 7.6 * breathe}), R.Mul(Pitch(-0.1)), mQueen)
	ab.WithPaint(func(p, n V3) *Mat {
		p = p.Mul(1 / x.K) // keep the pattern size of the unscaled design
		a := math.Atan2(p.Y, p.Z)
		u := math.Floor(p.X / 2.6)
		v := math.Floor(a * 2.2)
		if n.Z > -0.3 && hash(int(u)+40, int(v)+40, 9) < 0.15 {
			return mEggG
		}
		if d := math.Mod(p.X+40, 3.6); d < 0.7 {
			return mQueenSp
		}
		return nil
	})
	s.Add(ab)
	s.Add(Ellipsoid(x.p(V3{1.6, 0, 9.4}), x.v(V3{5.8, 6.2, 5.0}), R, mQueenHd))
	s.Add(Ellipsoid(x.p(V3{8.4, 0, 9.0}), x.v(V3{3.9, 4.4, 3.4}), R, mQueenHd))
	for i, sp := range [][3]float64{{0.6, 0, 1}, {-0.8, 2.6, 0.8}, {-0.8, -2.6, 0.8}, {-5, 0, 1.1}, {-12, 0, 1}} {
		base := V3{sp[0], sp[1], 9.0 + 4.2 + 0.6*float64(i%2)}
		if sp[0] < -3 {
			base.Z = 9.0 + 7.2
		}
		s.Add(Shard(x.p(base), x.r(1.0*sp[2]), x.r(0.8), x.r(3.2*sp[2]), R.Mul(Pitch(-0.35)), mQueenSp))
	}
	for _, side := range []float64{-1, 1} {
		a := x.p(V3{11.0, side * 2.0, 7.8})
		b := x.p(V3{13.8, side * 3.0, 7.0})
		c := x.p(V3{15.4, side * 1.0, 6.4})
		s.Add(Rod(a, b, x.r(0.9), mBone), Rod(b, c, x.r(0.7), mBone))
		s.Add(Sphere(x.p(V3{10.6, side * 1.7, 11.0}), x.r(0.8), mYellowG))
		s.Add(Sphere(x.p(V3{11.2, side * 2.8, 10.0}), x.r(0.6), mYellowG))
	}
	return s
}

// Frost beetle materials: a glossy shell of blue ice over a dark body.
var (
	mIceShell = mat(Ramp{Ink, Navy, DeepBlue, Blue, Cyan}, -0.12, 0.45)
	mIceSeam  = mat(Ramp{Ink, Navy, DeepBlue}, -0.05, 0.3)
	mIceSpike = mat(Ramp{Blue, Cyan, Silver, White, White}, 0.18, 0.9)
	mBeetle   = mat(Ramp{Ink, Navy, Slate, Steel}, -0.06, 0.25)
	mBeetleLg = mat(Ramp{Ink, Navy, Slate, Steel}, -0.02, 0.2)
)

// BeetleScene is the frost beetle: a squat bug under a domed shell of ice.
func BeetleScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.25}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	type leg struct{ hx, fx, fy float64 }
	legs := []leg{{2.6, 7.0, 7.4}, {0.2, 0.8, 8.8}, {-2.4, -5.6, 8.0}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 1.6, 1.4)
			hip := V3{lg.hx, side * 3.0, 3.2}
			foot := V3{lg.fx + sw, side * lg.fy, 0.6 + up}
			knee := V3{(lg.hx+foot.X)/2 + 0.3*sw, side * (lg.fy*0.72 + 0.6), 6.2 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.9), mBeetleLg))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.75), mBeetleLg))
		}
	}
	bob := 0.2 * math.Abs(math.Sin(phase))
	s.Add(Ellipsoid(x.p(V3{-0.8, 0, 3.4 + bob}), x.v(V3{6.6, 4.2, 2.5}), R, mBeetle))
	shell := Ellipsoid(x.p(V3{-2.4, 0, 5.2 + bob}), x.v(V3{6.4, 5.3, 3.5}), R.Mul(Pitch(-0.1)), mIceShell)
	shell.WithPaint(func(p, n V3) *Mat {
		p = p.Mul(1 / x.K)
		switch {
		case math.Abs(p.Y) < 0.45 && n.Z > 0.2:
			return mIceSeam // the seam between the two wing cases
		case p.Z < -1.6:
			return mIceSeam // the shell's rim
		}
		return nil
	})
	s.Add(shell)
	s.Add(Ellipsoid(x.p(V3{3.2, 0, 4.7 + bob}), x.v(V3{2.6, 3.6, 2.4}), R, mIceShell))
	s.Add(Ellipsoid(x.p(V3{5.8, 0, 3.6 + bob}), x.v(V3{2.0, 2.4, 1.8}), R, mBeetle))
	// Ice crystals growing out of the shell.
	for _, sp := range [][4]float64{{-0.6, 2.2, 0.75, 0.6}, {-0.6, -2.2, 0.75, -0.6}, {-3.6, 0, 1.25, 0}} {
		base := V3{sp[0], sp[1], 5.2 + bob + 2.6 - math.Abs(sp[1])*0.4}
		s.Add(Shard(x.p(base), x.r(1.1*sp[2]), x.r(1.2), x.r(3.6*sp[2]), R.Mul(Roll(sp[3])).Mul(Pitch(-0.5)), mIceSpike))
	}
	for _, side := range []float64{-1, 1} {
		a := x.p(V3{7.0, side * 1.4, 3.0 + bob})
		b := x.p(V3{9.1, side * 2.2, 2.6 + bob})
		c := x.p(V3{10.2, side * 0.8, 2.4 + bob})
		s.Add(Rod(a, b, x.r(0.65), mIceSpike), Rod(b, c, x.r(0.55), mIceSpike))
		s.Add(Sphere(x.p(V3{6.8, side * 1.4, 4.6 + bob}), x.r(0.7), mYellowG))
	}
	return s
}

// Splitter materials: a violet bug carrying amethyst crystals that break off
// as Shards.
var (
	mSplit    = mat(Ramp{Ink, Navy, Purple, Violet, Lilac}, -0.02, 0.35)
	mSplitSm  = mat(Ramp{Ink, Ink, Navy, Purple}, -0.05, 0.25)
	mSplitLg  = mat(Ramp{Ink, Navy, Purple, Violet}, -0.04, 0.2)
	mAmethyst = mat(Ramp{Purple, Violet, Lilac, White, White}, 0.08, 0.8)
	mSplitG   = glow(Ramp{Purple, Violet, Lilac, White}, -0.15)
	mShardB   = mat(Ramp{Ink, Purple, Violet, Lilac, Lilac}, 0.03, 0.35)
	mShardLg  = mat(Ramp{Ink, Navy, Purple, Violet}, 0.02, 0.2)
	mShardG   = glow(Ramp{Purple, Violet, Lilac, White}, -0.08)
)

// SplitterScene is a squat crystal bug with four amethyst shards on its back.
func SplitterScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.12}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	type leg struct{ hx, fx, fy float64 }
	legs := []leg{{2.4, 6.2, 6.4}, {0.2, 0.6, 7.6}, {-2.2, -5.2, 6.8}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 1.7, 1.5)
			hip := V3{lg.hx, side * 2.6, 3.4}
			foot := V3{lg.fx + sw, side * lg.fy, 0.6 + up}
			knee := V3{(lg.hx+foot.X)/2 + 0.3*sw, side * (lg.fy*0.68 + 0.8), 6.4 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.8), mSplitLg))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.65), mSplitLg))
		}
	}
	bob := 0.25 * math.Abs(math.Sin(phase))
	body := Ellipsoid(x.p(V3{-0.6, 0, 3.6 + bob}), x.v(V3{6.0, 4.6, 2.6}), R, mSplit)
	body.WithPaint(func(p, n V3) *Mat {
		p = p.Mul(1 / x.K)
		// Seams where the body will break apart.
		if (math.Abs(p.X+0.2) < 0.4 || math.Abs(p.Y) < 0.4) && n.Z > 0.1 {
			return mSplitSm
		}
		return nil
	})
	s.Add(body)
	s.Add(Ellipsoid(x.p(V3{4.8, 0, 3.4 + bob}), x.v(V3{2.4, 2.7, 2.0}), R, mSplitSm))
	// Four crystals leaning out from the glowing core: the future Shards.
	s.Add(Sphere(x.p(V3{-0.8, 0, 5.4 + bob}), x.r(1.2), mSplitG))
	for _, c := range [][4]float64{{1.2, 1.7, 0.5, -0.35}, {1.2, -1.7, -0.5, -0.35}, {-2.8, 1.7, 0.5, 0.35}, {-2.8, -1.7, -0.5, 0.35}} {
		base := V3{c[0], c[1], 4.2 + bob}
		s.Add(Shard(x.p(base), x.r(1.15), x.r(2.2), x.r(3.4), R.Mul(Roll(c[2])).Mul(Pitch(c[3])), mAmethyst))
	}
	for _, side := range []float64{-1, 1} {
		a := x.p(V3{6.2, side * 1.3, 2.8 + bob})
		b := x.p(V3{8.0, side * 2.0, 2.4 + bob})
		c := x.p(V3{8.9, side * 0.7, 2.2 + bob})
		s.Add(Rod(a, b, x.r(0.55), mAmethyst), Rod(b, c, x.r(0.45), mAmethyst))
		s.Add(Sphere(x.p(V3{6.4, side * 1.25, 4.1 + bob}), x.r(0.6), mYellowG))
	}
	return s
}

// ShardMiteScene is a splinter of a Splitter scuttling on four thin legs.
func ShardMiteScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.15}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	for li, hx := range []float64{1.0, -1.0} {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, up := gait(phase, g, 1.1, 1.0)
			hip := V3{hx, side * 1.3, 2.2}
			foot := V3{hx*2.2 + sw, side * 3.6, 0.4 + up}
			knee := V3{hx*1.5 + 0.3*sw, side * 2.6, 3.6 + up*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.45), mShardLg))
			s.Add(Rod(x.p(knee), x.p(foot), x.r(0.38), mShardLg))
		}
	}
	s.Add(Ellipsoid(x.p(V3{0.2, 0, 2.2}), x.v(V3{2.6, 2.0, 1.4}), R, mShardB))
	s.Add(Shard(x.p(V3{-0.6, 0, 2.2}), x.r(1.35), x.r(1.4), x.r(3.0), R.Mul(Pitch(-0.85)), mShardG))
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{2.4, side * 0.7, 2.5}), x.r(0.45), mYellowG))
	}
	return s
}

// Stinger materials: a hive wasp of the ice moon, amber with dark bands,
// pale wings and icy venom.
var (
	mStingY  = mat(Ramp{Ink, Brown, Ochre, Amber, Yellow}, 0.02, 0.45)
	mStingK  = mat(Ramp{Ink, Ink, Plum, Navy, Slate}, 0, 0.3)
	mStingW  = mat(Ramp{Steel, Gray, Silver, White, White}, 0.08, 0.7)
	mStingWv = mat(Ramp{Slate, Steel, Gray, Silver}, 0.05, 0.5)
	mStingG  = glow(Ramp{DeepBlue, Blue, Cyan, White}, -0.05)
)

// StingerScene is a flying hive wasp with buzzing wings (the model floats;
// the renderer adds the flight height).
func StingerScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xk{Yaw(heading), V3{}, 1.15}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	const cz = 4.2
	// Dangling legs.
	for i, lx := range []float64{1.9, 1.1, 0.3} {
		for _, side := range []float64{-1, 1} {
			hip := V3{lx, side * 0.7, cz - 0.6}
			knee := V3{lx - 0.3 + 0.1*float64(i), side * 1.5, cz - 1.7}
			foot := V3{lx - 0.9 + 0.2*float64(i), side * 1.7, cz - 2.9}
			s.Add(Rod(x.p(hip), x.p(knee), x.r(0.28), mStingK), Rod(x.p(knee), x.p(foot), x.r(0.24), mStingK))
		}
	}
	// Abdomen with dark bands, hanging a little, and the icy sting.
	ab := Ellipsoid(x.p(V3{-2.6, 0, cz - 0.2}), x.v(V3{2.9, 1.75, 1.6}), R.Mul(Pitch(-0.22)), mStingY)
	ab.WithPaint(func(p, n V3) *Mat {
		if d := math.Mod(p.X+40, 1.5); d < 0.6 {
			return mStingK
		}
		return nil
	})
	s.Add(ab)
	s.Add(Shard(x.p(V3{-5.0, 0, cz - 1.0}), x.r(0.55), x.r(0.4), x.r(2.0), R.Mul(alongZ(V3{-1, 0, -0.45})), mStingG))
	s.Add(Rod(x.p(V3{-0.4, 0, cz}), x.p(V3{0.6, 0, cz + 0.2}), x.r(0.5), mStingK))
	s.Add(Ellipsoid(x.p(V3{1.3, 0, cz + 0.3}), x.v(V3{1.6, 1.3, 1.3}), R, mStingK))
	hd := Ellipsoid(x.p(V3{3.3, 0, cz + 0.3}), x.v(V3{1.2, 1.3, 1.1}), R, mStingY)
	s.Add(hd)
	for _, side := range []float64{-1, 1} {
		s.Add(Sphere(x.p(V3{3.7, side * 0.85, cz + 0.7}), x.r(0.55), mStingG))
		s.Add(Rod(x.p(V3{3.9, side * 0.4, cz + 1.2}), x.p(V3{5.6, side * 1.3, cz + 2.4}), x.r(0.2), mStingK))
	}
	// Two pairs of wings beating fast.
	for wi, w := range []struct{ hx, span, chord, lift, amp, off float64 }{
		{1.6, 3.4, 1.25, 0.45, 0.75, 0}, {0.6, 2.6, 1.0, 0.3, 0.6, 0.6},
	} {
		th := w.lift + w.amp*math.Sin(phase+w.off)
		for _, side := range []float64{-1, 1} {
			hinge := V3{w.hx, side * 0.8, cz + 1.0}
			dir := V3{-0.35, side * math.Cos(th), math.Sin(th)}.Norm()
			c := hinge.Add(dir.Mul(w.span))
			rot := R.Mul(Yaw(-side * 0.3)).Mul(Roll(side * th))
			m := mStingW
			if wi == 1 {
				m = mStingWv
			}
			s.Add(Ellipsoid(x.p(c), x.v(V3{w.chord, w.span, 0.12}), rot, m))
		}
	}
	return s
}
