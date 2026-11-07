package art

import "math"

// Turret types, in build-menu order (they match the game's tower IDs).
var TurretIDs = []string{"blaster", "cryo", "flamer", "railgun", "skyguard"}

// TurretLevels is the number of looks per turret: three stat levels and the super.
const TurretLevels = 4

// Turret materials. Each type has its own accent so they read apart at a glance:
// blaster cyan, cryo ice blue, flamer orange, railgun violet, skyguard red.
var (
	mFrostM  = mat(Ramp{Navy, Slate, Steel, Gray, Silver, White}, 0.02, 0.4)
	mIce     = mat(Ramp{Navy, DeepBlue, Blue, Cyan, White}, 0.02, 0.8)
	mIceG    = glow(Ramp{DeepBlue, Blue, Cyan, White}, -0.1)
	mIceTank = mat(Ramp{Ink, DeepBlue, Blue, Cyan, White}, -0.02, 0.7)
	mTank    = mat(Ramp{Ink, Plum, Maroon, Red, Pink}, -0.02, 0.45)
	mFireG   = glow(Ramp{Red, Orange, Amber, Yellow}, 0)
	mRail    = mat(Ramp{Ink, Navy, Slate, Steel, Gray, Silver}, -0.06, 0.45)
	mViolet  = mat(Ramp{Ink, Navy, Purple, Violet, Lilac}, 0, 0.45)
	mVioletG = glow(Ramp{Purple, Violet, Lilac, White}, -0.25)
	mSky     = mat(Ramp{Ink, Navy, Slate, Steel, Gray, Silver}, 0.04, 0.3)
	mSilver  = mat(Ramp{Navy, Slate, Steel, Gray, Silver, White}, 0.1, 0.5)
	mPlain   = mat(Ramp{Ink, Navy, Slate, Steel}, 0.02, 0.2)
	mMissile = mat(Ramp{Slate, Steel, Gray, Silver, White}, 0.05, 0.45)
	mRedTip  = mat(Ramp{Plum, Maroon, Red, Pink}, 0.06, 0.45)
	mRedTrim = mat(Ramp{Ink, Plum, Maroon, Red}, 0.02, 0.35)
	mGold    = mat(Ramp{Brown, Ochre, Amber, Yellow}, 0, 0.45)
)

// turretStyle is the per-type accent: level 2 trim and the glow of the super.
type turretStyle struct {
	trim, glow *Mat
	stud       *Mat
}

var turretStyles = map[string]turretStyle{
	"blaster":  {trims[1], glow(Ramp{DeepBlue, Blue, Cyan, White}, -0.38), mAmber},
	"cryo":     {mIceTank, glow(Ramp{DeepBlue, Blue, Cyan, White}, -0.3), mIce},
	"flamer":   {mTank, glow(Ramp{Maroon, Red, Orange, Amber, Yellow}, -0.3), mAmber},
	"railgun":  {mViolet, glow(Ramp{Purple, Violet, Lilac, White}, -0.3), mViolet},
	"skyguard": {mRedTrim, glow(Ramp{Plum, Maroon, Red, Pink}, -0.25), mRedTip},
}

// levelTrim is the ring colour of a level: plain, silver, gold, then the type
// colour glowing for the super.
func levelTrim(id string, lv int) *Mat {
	st := turretStyles[id]
	switch lv {
	case 0:
		return mPlain
	case 1:
		return mSilver
	case 2:
		return mGold
	}
	return st.glow
}

// TurretSpin is the rotational symmetry period of a turret head: heads that only
// spin (the cryo) look the same every quarter turn, so fewer angles are baked.
func TurretSpin(id string) float64 {
	if id == "cryo" {
		return math.Pi / 2
	}
	return 2 * math.Pi
}

// TurretTip is where shots leave the head: distance ahead of the pivot and height
// above the pad top.
func TurretTip(id string, lv int) (fwd, z float64) {
	switch id {
	case "cryo":
		return 0, PivotZ + 5
	case "flamer":
		return [4]float64{11, 12, 12.6, 12.6}[lv], PivotZ + MuzzleZ
	case "railgun":
		return [4]float64{16.4, 18, 19.6, 19.6}[lv], PivotZ + MuzzleZ
	case "skyguard":
		return 5.6, PivotZ + 6.4
	}
	return [4]float64{12.8, 13.2, 15.2, 15.2}[lv], PivotZ + MuzzleZ
}

// TurretBaseScene is the static mount of a turret standing on the pad top.
func TurretBaseScene(id string, lv int) *Scene {
	s := &Scene{}
	turretBase(s, id, lv, V3{})
	return s
}

// TurretHeadScene is the rotating part of a turret aiming at angle a.
func TurretHeadScene(id string, lv int, a float64) *Scene {
	s := &Scene{}
	turretHead(s, id, lv, a, V3{})
	return s
}

func TurretBase(id string, lv int) *Sprite {
	return TurretBaseScene(id, lv).Bake(spriteOpts, true, 0, 0)
}
func TurretHead(id string, lv int, a float64) *Sprite {
	return TurretHeadScene(id, lv, a).Bake(spriteOpts, true, 0, 0)
}

// PortraitAngle shows a turret aiming to the front-right.
const PortraitAngle = 0.55

// TurretPortrait renders a complete turret standing on its pad to fit a w x h box.
func TurretPortrait(id string, lv int, w, h float64, pad bool) *Sprite {
	s := &Scene{}
	o := V3{}
	if pad {
		s.Add(PadScene(0, 0).Parts...)
		o = V3{0, 0, PadH}
	}
	turretBase(s, id, lv, o)
	turretHead(s, id, lv, PortraitAngle, o)
	probe := s.BakeScaled(spriteOpts, 1)
	k := math.Min(w/float64(probe.C.W), h/float64(probe.C.H))
	return s.BakeScaled(Opts{Inner: 3 / k, Shadows: true}, k)
}

// --- mounts ----------------------------------------------------------------------

func turretBase(s *Scene, id string, lv int, o V3) {
	P := func(x, y, z float64) V3 { return o.Add(V3{x, y, z}) }
	st := turretStyles[id]
	trim := levelTrim(id, lv)
	r := 9.6 + 0.5*float64(min(lv, 2))
	column := mMetalLo
	if id == "cryo" {
		column = mFrostM
	}
	// Feet, a ring and the column the head turns on.
	switch id {
	case "railgun":
		for _, d := range [][2]float64{{-8, -8}, {8, -8}, {-8, 8}, {8, 8}} {
			s.Add(Block(P(d[0], d[1], 1.2), V3{3, 3, 1.2}, Ident, 1, 0.6, mDarker))
		}
		s.Add(Block(P(0, 0, 1.6), V3{r - 0.4, r - 0.4, 1.6}, Ident, 3.4, 0.8, mDark))
		frame := Block(P(0, 0, 3.2), V3{r - 1.6, r - 1.6, 0.6}, Ident, 3, 0.3, trim)
		frame.Sub(mDark, Block(P(0, 0, 3.2), V3{r - 3.2, r - 3.2, 1.2}, Ident, 2.4, 0, nil))
		s.Add(frame)
	case "flamer":
		for _, d := range [][2]float64{{-7.6, -7.6}, {7.6, -7.6}, {-7.6, 7.6}, {7.6, 7.6}} {
			s.Add(Block(P(d[0], d[1], 1.1), V3{2.6, 2.6, 1.1}, Yaw(math.Pi/4), 0, 0.6, mDarker))
		}
		s.Add(Prism(P(0, 0, 0), r+0.8, r-0.2, 3, 6, Yaw(math.Pi/6), mDark))
		s.Add(Prism(P(0, 0, 2.2), r+0.5, r+0.1, 1.2, 6, Yaw(math.Pi/6), trim))
	default:
		for _, d := range [][2]float64{{-7.6, -7.6}, {7.6, -7.6}, {-7.6, 7.6}, {7.6, 7.6}} {
			s.Add(Block(P(d[0], d[1], 1.1), V3{2.6, 2.6, 1.1}, Yaw(math.Pi/4), 0, 0.6, mDarker))
		}
		s.Add(Lathe(P(0, 0, 0), r+0.6, r-0.4, 3, 24, Ident, mDark))
		s.Add(Lathe(P(0, 0, 2.2), r+0.3, r-0.1, 1.2, 24, Ident, trim))
	}
	s.Add(Lathe(P(0, 0, 3), 5.6, 4.2, PivotZ-3, 16, Ident, column))
	s.Add(Lathe(P(0, 0, PivotZ-1.6), 5.0, 4.8, 1.2, 16, Ident, mDark))

	// Studs show the third level; they light up with the super.
	stud := st.stud
	if lv == 3 {
		stud = st.glow
	}
	switch id {
	case "cryo":
		// Coolant tanks around the column.
		n := 2
		if lv >= 1 {
			n = 4
		}
		tank := mIceTank
		if lv == 3 {
			tank = mIceG
		}
		for i := 0; i < n; i++ {
			a := float64(i)/float64(n)*2*math.Pi + math.Pi/4
			cx, cy := math.Cos(a)*7.4, math.Sin(a)*7.4
			s.Add(Rod(P(cx, cy, 2.6), P(cx, cy, 4.6+0.4*float64(min(lv, 2))), 1.7, tank))
			s.Add(Lathe(P(cx, cy, 1.6), 2.2, 2.0, 1.2, 10, Ident, mDark))
		}
	case "skyguard":
		// A sensor mast behind the turret.
		h := 11.0 + 1.5*float64(min(lv, 2))
		s.Add(Lathe(P(-7.4, -7.4, 1), 1.6, 1.2, 2.4, 8, Ident, mDark))
		s.Add(Cylinder(P(-7.4, -7.4, 3), P(-7.4, -7.4, h), 0.55, mMetal))
		s.Add(Sphere(P(-7.4, -7.4, h+0.4), 0.95, mRedG))
		if lv >= 2 {
			for i := 0; i < 4; i++ {
				a := float64(i)/4*2*math.Pi + math.Pi/4
				s.Add(Sphere(P(math.Cos(a)*7.2, math.Sin(a)*7.2, 3.2), 1.1, stud))
			}
		}
	case "railgun":
		if lv >= 1 {
			for _, d := range [][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				c := P(d[0]*(r-0.3), d[1]*(r-0.3), 1.7)
				hs := V3{0.4 + math.Abs(d[1])*3.4, 0.4 + math.Abs(d[0])*3.4, 0.5}
				m := mViolet
				if lv >= 2 {
					m = mVioletG
				}
				s.Add(Block(c, hs, Ident, 0, 0, m))
			}
		}
		if lv >= 2 {
			for _, d := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
				s.Add(Sphere(P(d[0]*6.2, d[1]*6.2, 3.8), 1.1, stud))
			}
		}
	case "flamer":
		if lv >= 1 {
			// Spare fuel canisters on the mount.
			for _, y := range []float64{-5.6, 5.6} {
				s.Add(Rod(P(-7.6, y, 4.4), P(-7.6, y*0.55, 4.4), 1.3, mTank))
			}
		}
		if lv >= 2 {
			for i := 0; i < 3; i++ {
				a := float64(i)/3*2*math.Pi + math.Pi/3
				s.Add(Sphere(P(math.Cos(a)*7.6, math.Sin(a)*7.6, 3.4), 1.0, stud))
			}
		}
	default:
		if lv >= 2 {
			for i := 0; i < 4; i++ {
				a := float64(i)/4*2*math.Pi + math.Pi/4
				s.Add(Sphere(P(math.Cos(a)*7.2, math.Sin(a)*7.2, 3.2), 1.1, stud))
			}
		}
	}
}

// --- heads -------------------------------------------------------------------------

func turretHead(s *Scene, id string, lv int, a float64, o V3) {
	x := xf{Yaw(a), o.Add(V3{0, 0, PivotZ})}
	switch id {
	case "cryo":
		cryoHead(s, x, lv)
	case "flamer":
		flamerHead(s, x, lv)
	case "railgun":
		railHead(s, x, lv)
	case "skyguard":
		skyHead(s, x, lv)
	default:
		blasterHead(s, x, lv)
	}
}

// muzzleRing adds a barrel end with a dark bore along local x at (y, z).
func muzzleRing(s *Scene, x xf, y, z, x0, x1, r float64, m *Mat) {
	p := Cylinder(x.p(V3{x0, y, z}), x.p(V3{x1, y, z}), r, m)
	p.Sub(mBore, Cylinder(x.p(V3{x1 - 0.8, y, z}), x.p(V3{x1 + 1, y, z}), r*0.45, nil))
	s.Add(p)
}

func blasterHead(s *Scene, x xf, lv int) {
	R := x.R
	mz := MuzzleZ
	barrel := func(y, r, x0, x1, mr, mlen float64) {
		s.Add(Cylinder(x.p(V3{x0, y, mz}), x.p(V3{x1, y, mz}), r, mMetal))
		muzzleRing(s, x, y, mz, x1-mlen, x1+0.4, mr, mDark)
	}
	switch lv {
	case 0:
		s.Add(Ellipsoid(x.p(V3{-0.6, 0, 2.6}), V3{6.2, 5.6, 3.9}, R, mMetal))
		s.Add(Block(x.p(V3{-4.6, 0, 2.9}), V3{2.6, 4.0, 2.4}, R, 1, 0.8, mMetalLo))
		barrel(0, 1.5, 2, 12.4, 2.1, 2.2)
		s.Add(Sphere(x.p(V3{2.2, -2.7, 5.2}), 0.9, mCyanG))
	case 1:
		s.Add(Ellipsoid(x.p(V3{-0.6, 0, 2.7}), V3{6.8, 6.4, 4.1}, R, mMetal))
		s.Add(Block(x.p(V3{-5.0, 0, 3.0}), V3{2.6, 5.2, 2.6}, R, 1.2, 0.8, mMetalLo))
		for _, y := range []float64{-2.3, 2.3} {
			barrel(y, 1.2, 2, 12.8, 1.65, 1.8)
		}
		s.Add(Rod(x.p(V3{-1.6, -5.6, 3.4}), x.p(V3{-1.6, 5.6, 3.4}), 1.15, trims[1]))
		s.Add(Sphere(x.p(V3{2.4, 0, 6.3}), 0.9, mCyanG))
	default:
		for _, y := range []float64{-6.4, 6.4} {
			s.Add(Block(x.p(V3{-1.2, y, 2.6}), V3{4.6, 1.5, 2.6}, R, 1, 0.8, mDarker))
		}
		s.Add(Ellipsoid(x.p(V3{-0.6, 0, 2.9}), V3{7.4, 6.6, 4.4}, R, mMetal))
		s.Add(Block(x.p(V3{-5.4, 0, 3.2}), V3{2.6, 5.6, 2.8}, R, 1.4, 0.8, mMetalLo))
		s.Add(Cylinder(x.p(V3{2, 0, mz}), x.p(V3{13.4, 0, mz}), 2.1, mMetal))
		for _, k := range []float64{6, 8.6, 11.2} {
			s.Add(Cylinder(x.p(V3{k - 0.5, 0, mz}), x.p(V3{k + 0.5, 0, mz}), 2.75, mCyanG))
		}
		muzzleRing(s, x, 0, mz, 12.6, 15.4, 2.6, mDark)
		if lv == 2 {
			s.Add(Rod(x.p(V3{-1.8, -6.2, 4.4}), x.p(V3{-1.8, 6.2, 4.4}), 1.2, mGold))
			s.Add(Sphere(x.p(V3{2.6, 0, 6.9}), 1.0, mCyanG))
			return
		}
		// Ricochet: deflector prongs on both sides and a glowing prism on top.
		for _, sg := range []float64{-1, 1} {
			s.Add(Block(x.p(V3{-1.2, sg * 7.95, 2.8}), V3{3.2, 0.1, 1.0}, R, 0, 0, mCyanG))
			a0 := V3{1.5, sg * 4.6, mz - 0.9}
			dir := V3{math.Cos(0.16), sg * math.Sin(0.16), 0}
			s.Add(Cylinder(x.p(a0), x.p(a0.Add(dir.Mul(8.2))), 0.95, mMetal))
			tip := a0.Add(dir.Mul(8.2))
			s.Add(Sphere(x.p(tip), 1.25, mCyanG))
		}
		s.Add(Rod(x.p(V3{-1.8, -6.2, 4.4}), x.p(V3{-1.8, 6.2, 4.4}), 1.2, mCyanG))
		s.Add(Shard(x.p(V3{0.8, 0, 6.2}), 1.5, 1.2, 2.4, R, mCyanG))
	}
}

func cryoHead(s *Scene, x xf, lv int) {
	R := x.R
	k := 1 + 0.07*float64(min(lv, 2))
	// An emitter drum with a coloured band and four fins (they show the spin).
	s.Add(Lathe(x.p(V3{0, 0, -0.6}), 5.0*k, 4.5*k, 3.4, 16, R, mFrostM))
	s.Add(Lathe(x.p(V3{0, 0, 0.5}), 5.25*k, 5.2*k, 1.0, 16, R, levelTrim("cryo", lv)))
	for i := 0; i < 4; i++ {
		t := float64(i) / 4 * 2 * math.Pi
		c, sn := math.Cos(t), math.Sin(t)
		s.Add(Block(x.p(V3{c * 5.6 * k, sn * 5.6 * k, 1.4}), V3{1.7, 0.6, 1.5}, R.Mul(Yaw(t)), 0, 0.5, mDark))
	}
	core := mIce
	if lv >= 2 {
		core = mIceG
	}
	h := [4]float64{2.6, 3.2, 4.0, 4.6}[lv]
	r := [4]float64{2.2, 2.4, 2.7, 2.9}[lv]
	s.Add(Shard(x.p(V3{0, 0, 2.4}), r, h, 2.2+0.3*float64(lv), R, core))
	if lv >= 1 {
		// Smaller crystals lean out of the drum.
		for i := 0; i < 4; i++ {
			t := float64(i)/4*2*math.Pi + math.Pi/4
			c, sn := math.Cos(t), math.Sin(t)
			dir := R.Apply(V3{c * 0.6, sn * 0.6, 1})
			sz := 0.85 + 0.15*float64(lv)
			s.Add(Shard(x.p(V3{c * 3.3 * k, sn * 3.3 * k, 2.2}), sz, 1.2+0.4*float64(lv), 1.3, alongZ(dir), mIce))
		}
	}
	if lv == 3 {
		// Deep freeze: a halo floats around the crystal.
		ring := Lathe(x.p(V3{0, 0, 7.4}), 6.2, 6.2, 0.8, 16, R, mIceG)
		ring.Sub(mIceG, Lathe(x.p(V3{0, 0, 7}), 5.0, 5.0, 1.6, 16, R, nil))
		s.Add(ring)
	}
}

func flamerHead(s *Scene, x xf, lv int) {
	R := x.R
	mz := MuzzleZ
	hazard := func(p, n V3) *Mat {
		if n.Z < 0.6 {
			return nil
		}
		if math.Mod(p.X+p.Y+40, 2.6) < 1.3 {
			return mAmber
		}
		return mInk
	}
	pilot := func(xx, y float64) {
		s.Add(Sphere(x.p(V3{xx, y, mz - 1.9}), 0.75, mFireG))
	}
	switch lv {
	case 0:
		s.Add(Block(x.p(V3{-0.8, 0, 2.6}), V3{5.0, 4.4, 2.8}, R, 1.4, 1.0, mMetal))
		s.Add(Block(x.p(V3{-0.4, 0, 5.5}), V3{3.0, 2.6, 0.3}, R, 0.6, 0, mInk).WithPaint(hazard))
		s.Add(Rod(x.p(V3{-5.8, -3.2, 3.6}), x.p(V3{-5.8, 3.2, 3.6}), 2.4, mTank))
		s.Add(Cylinder(x.p(V3{3, 0, mz}), x.p(V3{9.4, 0, mz}), 1.3, mMetal))
		muzzleRing(s, x, 0, mz, 8.8, 11, 1.9, mDark)
		pilot(10.2, 0)
	case 1:
		s.Add(Block(x.p(V3{-0.8, 0, 2.7}), V3{5.4, 4.8, 3.0}, R, 1.4, 1.0, mMetal))
		s.Add(Block(x.p(V3{-0.2, 0, 5.8}), V3{3.4, 3.0, 0.3}, R, 0.6, 0, mInk).WithPaint(hazard))
		for _, y := range []float64{-3.0, 3.0} {
			s.Add(Rod(x.p(V3{-6.4, y, 1.2}), x.p(V3{-6.4, y, 5.8}), 2.1, mTank))
		}
		s.Add(Cylinder(x.p(V3{3, 0, mz}), x.p(V3{10.4, 0, mz}), 1.4, mMetal))
		s.Add(Cylinder(x.p(V3{5.2, 0, mz}), x.p(V3{6.4, 0, mz}), 2.3, mMetalLo))
		muzzleRing(s, x, 0, mz, 9.8, 12, 2.0, mDark)
		pilot(11.2, 0)
	default:
		for _, y := range []float64{-5.4, 5.4} {
			s.Add(Block(x.p(V3{-0.6, y, 2.7}), V3{4.4, 1.2, 2.6}, R, 0.8, 0.8, mDarker))
		}
		s.Add(Block(x.p(V3{-0.8, 0, 2.9}), V3{5.6, 4.6, 3.2}, R, 1.6, 1.0, mMetal))
		s.Add(Block(x.p(V3{-0.2, 0, 6.2}), V3{3.6, 3.0, 0.3}, R, 0.6, 0, mInk).WithPaint(hazard))
		band := func(p, n V3) *Mat {
			if p.Z > 1.6 && p.Z < 2.8 {
				return mGold
			}
			return nil
		}
		for _, y := range []float64{-3.2, 3.2} {
			s.Add(Rod(x.p(V3{-6.8, y, 1.2}), x.p(V3{-6.8, y, 6.4}), 2.2, mTank).WithPaint(band))
		}
		if lv == 3 {
			// Napalm: a glowing reservoir between the tanks and burning nozzle rings.
			s.Add(Rod(x.p(V3{-7.4, 0, 2.2}), x.p(V3{-7.4, 0, 7.4}), 1.6, mFireG))
		}
		for _, y := range []float64{-1.9, 1.9} {
			s.Add(Cylinder(x.p(V3{3, y, mz}), x.p(V3{10.8, y, mz}), 1.1, mMetal))
			ring := mDark
			if lv == 3 {
				ring = mFireG
			}
			s.Add(Cylinder(x.p(V3{6.2, y, mz}), x.p(V3{7.2, y, mz}), 1.75, ring))
			muzzleRing(s, x, y, mz, 10.4, 12.4, 1.6, mDark)
			pilot(11.6, y)
		}
	}
}

func railHead(s *Scene, x xf, lv int) {
	R := x.R
	mz := MuzzleZ
	length := [4]float64{16, 17.6, 19.2, 19.2}[lv]
	coils := [4][]float64{{5.5, 10.5}, {5, 9.5, 14}, {5, 9.4, 13.8}, {4.6, 8.4, 12.2, 16}}[lv]
	bw := [4]float64{5.4, 5.8, 6.2, 6.2}[lv]
	// Body with a capacitor at the back.
	s.Add(Block(x.p(V3{-2.4, 0, 2.6}), V3{bw, 4.4 + 0.2*float64(min(lv, 2)), 2.8}, R, 1.2, 1.0, mRail))
	capM := mViolet
	if lv == 3 {
		capM = mVioletG
	}
	s.Add(Cylinder(x.p(V3{-6.6 - 0.3*float64(lv), -3.4, 4.2}), x.p(V3{-6.6 - 0.3*float64(lv), 3.4, 4.2}), 2.0, capM))
	// Two rails with coils around them.
	for _, y := range []float64{-1.75, 1.75} {
		s.Add(Block(x.p(V3{length / 2, y, mz}), V3{length / 2, 0.75, 1.5}, R, 0, 0.4, mMetalLo))
	}
	for _, c := range coils {
		s.Add(Block(x.p(V3{c, 0, mz}), V3{0.7, 2.9, 2.3}, R, 0.6, 0, mVioletG))
	}
	if lv >= 1 {
		for _, y := range []float64{-4.9, 4.9} {
			s.Add(Rod(x.p(V3{-5.4, y, 2.4}), x.p(V3{0.8, y, 2.4}), 1.2, mViolet))
		}
	}
	if lv >= 2 {
		// An armoured shroud over the rear of the rails and cooling fins.
		s.Add(Block(x.p(V3{1.6, 0, mz + 2.1}), V3{3.2, 2.8, 0.8}, R, 0.8, 0.6, mRail))
		for _, k := range []float64{-4.4, -2.6, -0.8} {
			s.Add(Block(x.p(V3{k, 0, 5.9}), V3{0.35, 3.4, 0.9}, R, 0, 0.2, mDarker))
		}
		trim := mGold
		if lv == 3 {
			trim = mVioletG
		}
		s.Add(Rod(x.p(V3{-3, -5.2, 4.6}), x.p(V3{-3, 5.2, 4.6}), 0.9, trim))
	}
	if lv == 3 {
		// Overcharge: a plasma channel glows between the rails.
		s.Add(Block(x.p(V3{length/2 + 0.8, 0, mz}), V3{length/2 - 1.2, 0.55, 0.55}, R, 0, 0, mVioletG))
		s.Add(Sphere(x.p(V3{-2.2, 0, 6.3}), 1.5, mVioletG))
	} else {
		s.Add(Sphere(x.p(V3{-1.2, -3.4, 5.6}), 0.8, mVioletG))
	}
}

func skyHead(s *Scene, x xf, lv int) {
	R := x.R
	const pitch = 0.42
	P := R.Mul(Pitch(pitch))
	s.Add(Lathe(x.p(V3{0, 0, -0.4}), 5.6, 5.2, 1.8, 16, R, mDark))
	hw := [4]float64{2.8, 3.0, 3.2, 3.2}[lv]
	s.Add(Block(x.p(V3{-0.8, 0, 3.0}), V3{4.0, hw, 3.0}, R, 1.2, 1.0, mSky))
	// The launch frame tilts up; missiles ride on rails either side of the body.
	px := xf{P, x.p(V3{0.6, 0, 4.4})}
	missile := func(y, z float64) {
		const l = 9.0
		s.Add(Block(px.p(V3{-0.2, y, z - 1.25}), V3{4.4, 0.5, 0.3}, P, 0, 0, mDark))
		s.Add(Rod(px.p(V3{-l / 2, y, z}), px.p(V3{l/2 - 1.6, y, z}), 1.05, mMissile))
		s.Add(Ellipsoid(px.p(V3{l/2 - 1.3, y, z}), V3{1.7, 1.02, 1.02}, P, mRedTip))
		s.Add(Block(px.p(V3{-l/2 + 0.6, y, z}), V3{0.7, 0.18, 1.8}, P, 0, 0, mRedTrim))
		s.Add(Block(px.p(V3{-l/2 + 0.6, y, z}), V3{0.7, 1.8, 0.18}, P, 0, 0, mRedTrim))
	}
	for _, sg := range []float64{-1, 1} {
		s.Add(Block(px.p(V3{-1.2, sg * (hw + 0.5), 0}), V3{2.4, 0.6, 1.6}, P, 0.4, 0.4, mSky))
		missile(sg*(hw+1.9), 0)
		if lv >= 1 {
			missile(sg*(hw+4.1), 0)
		}
		if lv == 3 {
			missile(sg*(hw+3.0), 2.0)
		}
	}
	eye := mRedG
	s.Add(Sphere(x.p(V3{2.6, 0, 4.6}), 1.1, eye))
	if lv >= 2 {
		trim := mGold
		if lv == 3 {
			trim = mRedG
		}
		s.Add(Block(x.p(V3{-1.2, 0, 6.2}), V3{2.4, hw - 0.8, 0.3}, R, 0.5, 0, trim))
		// Tracking radar on top.
		s.Add(Cylinder(x.p(V3{-2.6, 0, 6}), x.p(V3{-2.6, 0, 8.6}), 0.45, mMetal))
		s.Add(Lathe(x.p(V3{-2.6, 0, 8}), 0.6, 2.8, 1.1, 12, R.Mul(Pitch(-0.5)), mMetalLo))
	}
}

// TurretIcon renders a turret zoom times larger than a w x h portrait and crops it
// to the box, keeping the head; the cut-off foot of the base fades out.
func TurretIcon(id string, lv int, w, h int, zoom float64) *Sprite {
	s := &Scene{}
	turretBase(s, id, lv, V3{})
	turretHead(s, id, lv, PortraitAngle, V3{})
	probe := s.BakeScaled(spriteOpts, 1)
	k := math.Min(float64(w)/float64(probe.C.W), float64(h)/float64(probe.C.H)) * zoom
	sp := s.BakeScaled(Opts{Inner: 3 / k, Shadows: true}, k)
	cw, ch := min(w, sp.C.W), min(h, sp.C.H)
	x0 := (sp.C.W - cw) / 2
	out := NewCanvas(cw, ch)
	fade := 0
	if sp.C.H > ch {
		fade = 4
	}
	for y := 0; y < ch; y++ {
		a := 1.0
		if d := ch - y; d <= fade {
			a = float64(d) / float64(fade+1)
		}
		for x := 0; x < cw; x++ {
			c := sp.C.Get(x0+x, y)
			c.A = uint8(float64(c.A) * a)
			c.R, c.G, c.B = uint8(float64(c.R)*a), uint8(float64(c.G)*a), uint8(float64(c.B)*a)
			out.Set(x, y, c)
		}
	}
	return &Sprite{C: out, OX: sp.OX - x0, OY: sp.OY}
}
