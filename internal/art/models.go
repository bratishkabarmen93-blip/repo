package art

import (
	"image/color"
	"math"
	"math/rand"
)

// Heights shared with the game renderer (world pixels).
const (
	PadH     = 4.0  // top of a build pad
	PivotZ   = 9.0  // turret head pivot above the pad top
	MuzzleZ  = 3.6  // barrel height above the pivot
	BaseTopZ = 21.0 // roof of the command center
)

// Command center details (relative to its centre on the ground).
var (
	RadarPos   = V3{27, -27, 31}
	BaseLights = []V3{{-27, -27, 27}, {-27, 27, 27}, {27, 27, 27}}
	DomeCenter = V3{0, 0, 27}
)

func mat(r Ramp, bias, spec float64) *Mat { return &Mat{Ramp: r, Bias: bias, Spec: spec} }
func glow(r Ramp, bias float64) *Mat      { return &Mat{Ramp: r, Bias: bias, Glow: true} }

var (
	mDark    = mat(DarkMetal, -0.02, 0.15)
	mDarker  = mat(DarkMetal, -0.14, 0.1)
	mMetal   = mat(MetalRamp, -0.13, 0.28)
	mMetalLo = mat(MetalRamp, -0.2, 0.25)
	mInk     = mat(Ramp{Ink, Ink, Navy}, 0, 0)
	mBore    = mat(Ramp{Ink}, 0, 0)
	mAmber   = mat(Ramp{Brown, Ochre, Amber, Yellow}, 0.05, 0.2)
	mGlass   = mat(GlassRamp, -0.02, 0.9)
	mCyanG   = glow(Ramp{Blue, Cyan, White}, 0)
	mYellowG = glow(Ramp{Amber, Yellow, White}, -0.1)
	mRedG    = glow(Ramp{Maroon, Red, Pink}, 0)
	mLamp    = mat(Ramp{Ink, Navy, Slate, Steel}, -0.05, 0.4)
	mAlien   = mat(AlienRamp, -0.02, 0.35)
	mAlienHd = mat(AlienRamp, -0.14, 0.3)
	mAlienSt = mat(Ramp{Ink, DeepTeal, Forest, Green}, -0.05, 0.2)
	mLeg     = mat(Ramp{Ink, DeepTeal, Forest, Green}, -0.05, 0.1)
	mBone    = mat(BoneRamp, 0.05, 0.3)
	mFlesh   = mat(FleshRamp, 0.02, 0.25)
	mFleshIn = mat(Ramp{Ink, Ink, Plum, Purple}, -0.1, 0)
	mPinkSp  = mat(Ramp{Purple, Magenta, Pink, Peach}, 0.1, 0.3)
	mRock    = mat(RockRamp, -0.1, 0.1)
	mCrystal = mat(Ramp{Plum, Purple, Magenta, Pink, Peach}, 0.02, 0.6)
	mShrub   = mat(PinkRamp, -0.05, 0.2)
	mBulb    = mat(Ramp{Magenta, Pink, Peach, White}, 0, 0.5)
	mPadTop  = mat(Ramp{Ink, Navy, Slate}, 0, 0)
	mBolt    = mat(Ramp{Slate, Steel, Gray, Silver}, 0, 0.3)
)

var trims = []*Mat{
	mat(Ramp{Navy, Slate, Steel, Gray, Silver}, 0, 0.3),
	mat(Ramp{DeepBlue, Blue, Cyan, White}, 0, 0.4),
	mat(Ramp{Brown, Ochre, Amber, Yellow}, 0, 0.4),
}

var spriteOpts = Opts{Inner: 3, Shadows: true}

// xf places local model coordinates (x forward) at a pivot with a heading.
type xf struct {
	R M3
	T V3
}

func (x xf) p(v V3) V3 { return x.R.Apply(v).Add(x.T) }

// PadScene is the metal build pad standing on the ground at (cx, cy).
func PadScene(cx, cy float64) *Scene {
	s := &Scene{}
	c := V3{cx, cy, 0}
	top := V3{cx, cy, PadH}
	body := Block(c.Add(V3{0, 0, PadH / 2}), V3{13.5, 13.5, PadH / 2}, Ident, 5, 1, mDark)
	recess := Block(top, V3{7, 7, 1}, Ident, 0, 0, nil)
	body.Sub(mPadTop, recess)
	body.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		ax, ay := math.Abs(p.X), math.Abs(p.Y)
		if (ax > 10.2 && ax < 11.6 && ay < 3.2) || (ay > 10.2 && ay < 11.6 && ax < 3.2) {
			return mAmber
		}
		return nil
	})
	s.Add(body)
	s.Add(Cylinder(top.Add(V3{0, 0, -1}), top.Add(V3{0, 0, -0.4}), 3, mLamp))
	for _, d := range [][2]float64{{-9.5, -9.5}, {9.5, -9.5}, {-9.5, 9.5}, {9.5, 9.5}} {
		s.Add(Sphere(top.Add(V3{d[0], d[1], -0.2}), 0.9, mBolt))
	}
	return s
}

// CrawlerScene is the basic alien bug walking along heading at animation frame f of n.
func CrawlerScene(heading float64, f, n int) *Scene {
	s := &Scene{}
	x := xf{Yaw(heading), V3{}}
	R := x.R
	phase := float64(f) / float64(n) * 2 * math.Pi
	type leg struct{ hx, fx, fy float64 }
	legs := []leg{{2.6, 6.6, 5.6}, {0.6, 1.2, 6.8}, {-1.6, -4.6, 6.2}}
	for li, lg := range legs {
		for _, side := range []float64{-1, 1} {
			g := li % 2
			if side > 0 {
				g = 1 - g
			}
			sw, lift := math.Sin(phase)*1.7, math.Max(0, math.Cos(phase))*1.6
			if g == 1 {
				sw, lift = -sw, math.Max(0, -math.Cos(phase))*1.6
			}
			hip := V3{lg.hx, side * 2.0, 4.2}
			foot := V3{lg.fx + sw, side * lg.fy, 0.6 + lift}
			knee := V3{(lg.hx+foot.X)/2 + 0.3*sw, side * (lg.fy*0.62 + 1.2), 6.6 + lift*0.5}
			s.Add(Rod(x.p(hip), x.p(knee), 0.8, mLeg))
			s.Add(Rod(x.p(knee), x.p(foot), 0.7, mLeg))
		}
	}
	ab := Ellipsoid(x.p(V3{-4.6, 0, 4.6}), V3{6.0, 4.8, 3.8}, R, mAlien)
	ab.WithPaint(func(p, n V3) *Mat {
		if d := math.Mod(p.X+20, 2.8); d < 0.9 && p.Z > 0.5 {
			return mAlienSt
		}
		return nil
	})
	s.Add(ab)
	s.Add(Ellipsoid(x.p(V3{1.2, 0, 4.8}), V3{3.2, 3.4, 2.9}, R, mAlien))
	s.Add(Ellipsoid(x.p(V3{4.8, 0, 4.3}), V3{2.7, 2.8, 2.3}, R, mAlienHd))
	for _, side := range []float64{-1, 1} {
		a := x.p(V3{6.2, side * 1.5, 3.6})
		b := x.p(V3{8.2, side * 2.2, 3.1})
		c := x.p(V3{9.3, side * 0.8, 2.9})
		s.Add(Rod(a, b, 0.65, mBone), Rod(b, c, 0.55, mBone))
		s.Add(Sphere(x.p(V3{5.9, side * 1.35, 5.3}), 0.75, mYellowG))
	}
	return s
}

// BaseScene is the command center, centred on the origin (96x96 footprint),
// with a gate on the west side and, when east is set, one on the east side.
func BaseScene(east bool) *Scene {
	s := &Scene{}
	found := Prism(V3{0, 0, 0}, 47.5, 46.5, 5, 8, Yaw(-math.Pi/8), mDark)
	found.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		ax, ay := math.Abs(p.X), math.Abs(p.Y)
		if (ax > 42.5 || ay > 42.5) && ax+ay < 62 && int(math.Floor((p.X+p.Y+200)/3))%2 == 0 {
			return mAmber
		}
		return nil
	})
	s.Add(found)
	main := Block(V3{0, 0, 11}, V3{36, 36, 6}, Ident, 12, 2, mMetalLo)
	main.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		ax, ay := math.Abs(p.X), math.Abs(p.Y)
		if math.Hypot(p.X, p.Y) > 23 && ((ax > 11.4 && ax < 12.6) || (ay > 11.4 && ay < 12.6)) {
			return mDarker
		}
		return nil
	})
	s.Add(main)
	for _, d := range [][2]float64{{-27, -27}, {27, -27}, {-27, 27}, {27, 27}} {
		s.Add(Block(V3{d[0], d[1], 21}, V3{7, 7, 4}, Ident, 3, 1, mMetal))
	}
	for _, l := range BaseLights {
		s.Add(Cylinder(l.Add(V3{0, 0, -2}), l.Add(V3{0, 0, -0.5}), 2.4, mLamp))
	}
	s.Add(Cylinder(V3{RadarPos.X, RadarPos.Y, 25}, RadarPos, 1.3, mDark))
	s.Add(Lathe(V3{0, 0, 17}, 23, 21.5, 4.5, 32, Ident, mDark))
	dome := Ellipsoid(V3{0, 0, 21.5}, V3{17.5, 17.5, 13}, Ident, mGlass)
	dome.Clip(V3{0, 0, -1}, 0)
	s.Add(dome)
	// Gates where the roads come in.
	sides := []float64{-1}
	if east {
		sides = append(sides, 1)
	}
	for _, sx := range sides {
		for _, y := range []float64{-14.5, 14.5} {
			s.Add(Block(V3{sx * 44, y, 9}, V3{3, 2.6, 9}, Ident, 0.8, 0.6, mMetal))
		}
		beam := Block(V3{sx * 44, 0, 19.5}, V3{3.4, 17, 2}, Ident, 0, 0.6, mMetalLo)
		beam.WithPaint(func(p, n V3) *Mat {
			if n.Y > 0.5 && int(math.Floor((p.Y+p.Z*0.6+40)/2.5))%2 == 0 {
				return mAmber
			}
			return nil
		})
		s.Add(beam)
		s.Add(Block(V3{sx * 45.5, 0, 6.5}, V3{1, 11.8, 6.5}, Ident, 0, 0, mInk))
	}
	return s
}

// RadarScene is the rotating dish on the command center (origin at the mast top).
func RadarScene(a float64) *Scene {
	s := &Scene{}
	R := Yaw(a).Mul(Pitch(0.45))
	x := xf{R, V3{0, 0, 2}}
	dish := Sphere(x.p(V3{0, 0, 0}), 6.4, mMetal)
	dish.Sub(mLamp, Sphere(x.p(V3{2.4, 0, 0}), 6.4, nil))
	dish.Clip(R.Apply(V3{1, 0, 0}), 0.5)
	s.Add(dish)
	s.Add(Rod(x.p(V3{-3.2, 0, 0}), x.p(V3{3.4, 0, 0}), 0.55, mDark))
	s.Add(Sphere(x.p(V3{3.6, 0, 0}), 0.9, mRedG))
	s.Add(Cylinder(V3{0, 0, 0}, V3{0, 0, 2}, 1.5, mDark))
	return s
}

// BurrowScene is the alien hive hole where enemies emerge.
func BurrowScene() *Scene {
	s := &Scene{}
	r := rand.New(rand.NewSource(11))
	for i := 0; i < 7; i++ {
		a := float64(i)/7*2*math.Pi + r.Float64()*0.5
		s.Add(Rod(V3{math.Cos(a) * 11, math.Sin(a) * 9, 3.5}, V3{math.Cos(a+0.25) * 19, math.Sin(a+0.25) * 16, 0.3}, 1.4, mFlesh))
	}
	mound := Ellipsoid(V3{0, 0, -1}, V3{16, 13, 7.5}, Ident, mFlesh)
	mound.Sub(mFleshIn, Ellipsoid(V3{0, 0, 7.5}, V3{10, 8.5, 9.5}, Ident, nil))
	mound.WithPaint(func(p, n V3) *Mat {
		a := math.Atan2(p.Y, p.X)
		if hash(int(math.Floor(a*7)), int(math.Floor(p.Z*1.2)), 5) < 0.12 {
			return mPinkSp
		}
		return nil
	})
	s.Add(mound)
	return s
}

// Decor builders used by the terrain (positions on the ground).

// capped gives a rock a cap of snow in biomes that have it.
func capped(p *Part, bio *biome) *Part {
	if bio.snow == nil {
		return p
	}
	return p.WithPaint(func(_, n V3) *Mat {
		if n.Z > 0.8 {
			return bio.snow
		}
		return nil
	})
}

func rockScene(r *rand.Rand, x, y float64, bio *biome) *Scene {
	s := &Scene{}
	n := 1 + r.Intn(3)
	for i := 0; i < n; i++ {
		rx := 1.8 + r.Float64()*2.6
		ry := rx * (0.65 + r.Float64()*0.3)
		rz := rx * (0.55 + r.Float64()*0.3)
		c := V3{x + r.Float64()*4 - 2, y + r.Float64()*3 - 1.5, rz * 0.35}
		s.Add(capped(Ellipsoid(c, V3{rx, ry, rz}, Yaw(r.Float64()*math.Pi), bio.rock), bio))
	}
	return s
}

func boulderScene(r *rand.Rand, x, y float64, bio *biome) *Scene {
	s := &Scene{}
	n := 2 + r.Intn(2)
	for i := 0; i < n; i++ {
		rx := 3.6 + r.Float64()*2.4 - float64(i)*0.8
		c := V3{x + r.Float64()*5 - 2.5, y + r.Float64()*3 - 1.5, float64(i)*3.2 + rx*0.5}
		s.Add(capped(Ellipsoid(c, V3{rx, rx * (0.75 + r.Float64()*0.2), rx * (0.8 + r.Float64()*0.3)}, Yaw(r.Float64()*math.Pi), bio.rock), bio))
	}
	return s
}

func crystalScene(r *rand.Rand, x, y float64, bio *biome) *Scene {
	s := &Scene{}
	n := 2 + r.Intn(2)
	for i := 0; i < n; i++ {
		bx := x + float64(i-n/2)*3.0 + r.Float64()
		h := 3 + r.Float64()*5
		rad := 1.2 + r.Float64()*0.7
		rot := Yaw(r.Float64() * math.Pi).Mul(Pitch((r.Float64() - 0.5) * 0.7))
		s.Add(Shard(V3{bx, y + r.Float64()*2 - 1, -0.5}, rad, h, rad*1.6, rot, bio.crystal))
	}
	return s
}

func shrubScene(r *rand.Rand, x, y float64, bio *biome) *Scene {
	s := &Scene{}
	n := 3 + r.Intn(3)
	for i := 0; i < n; i++ {
		a := r.Float64() * 2 * math.Pi
		l := 2 + r.Float64()*3
		h := 3 + r.Float64()*4
		tip := V3{x + math.Cos(a)*l, y + math.Sin(a)*l*0.8, h}
		s.Add(Rod(V3{x, y, 0}, tip, 0.6, bio.shrub))
		s.Add(Sphere(tip, 1.0, bio.bulb))
	}
	return s
}

// PadMarker is the bracket overlay drawn around a pad top (anchor at its centre).
func PadMarker() (*Canvas, int, int) {
	hw, hh := 16, int(math.Round(16*GY))
	c := NewCanvas(2*hw+2, 2*hh+2)
	for _, p := range [][2]int{{0, 0}, {2*hw + 1, 0}, {0, 2*hh + 1}, {2*hw + 1, 2*hh + 1}} {
		dx, dy := 1, 1
		if p[0] > hw {
			dx = -1
		}
		if p[1] > hh {
			dy = -1
		}
		for i := 0; i < 6; i++ {
			c.Set(p[0]+dx*i, p[1], White)
			if i < 5 {
				c.Set(p[0], p[1]+dy*i, White)
			}
		}
	}
	return c, hw + 1, hh + 1
}

// RangeRing visualises a turret range of r world pixels as a ground ellipse.
func RangeRing(r int, col color.RGBA) (*Canvas, int, int) {
	fr := float64(r)
	ry := fr * GY
	w, h := 2*r+4, int(math.Ceil(2*ry))+4
	c := NewCanvas(w, h)
	cx, cy := float64(w)/2, float64(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5-cx, (float64(y)+0.5-cy)/GY
			d := math.Hypot(dx, dy)
			switch {
			case d <= fr-1.2:
				c.Blend(x, y, col, 0.08)
			case d <= fr:
				a := math.Atan2(dy, dx)
				if int(math.Floor((a+math.Pi)*fr/5))%2 == 0 {
					c.Blend(x, y, col, 0.85)
				} else {
					c.Blend(x, y, col, 0.3)
				}
			}
		}
	}
	return c, w / 2, h / 2
}

// Splat is a green goo decal on the ground (anchor at its centre).
func Splat(variant int) *Canvas { return GooSplat(variant, Forest, Green, Lime) }

// GooSplat is a goo decal in the given dark, mid and highlight colours.
func GooSplat(variant int, dark, mid, hi color.RGBA) *Canvas {
	c := NewCanvas(26, 20)
	r := rand.New(rand.NewSource(int64(variant*7919 + 3)))
	oval := func(x, y, rx, ry, rot float64, col color.RGBA) {
		cs, sn := math.Cos(rot), math.Sin(rot)
		for py := 0; py < c.H; py++ {
			for px := 0; px < c.W; px++ {
				dx, dy := float64(px)+0.5-13-x, (float64(py)+0.5-10)/GY-y
				u := (dx*cs + dy*sn) / rx
				v := (-dx*sn + dy*cs) / ry
				if u*u+v*v <= 1 {
					c.Set(px, py, col)
				}
			}
		}
	}
	oval(0, 0, 5+r.Float64()*1.5, 3.8+r.Float64()*1.5, r.Float64()*math.Pi, dark)
	for i := 0; i < 7; i++ {
		a := r.Float64() * 2 * math.Pi
		d := 4 + r.Float64()*6
		s := 0.8 + r.Float64()*1.6
		oval(math.Cos(a)*d, math.Sin(a)*d, s, s, 0, dark)
	}
	oval(-1, -1, 2.6, 1.6, 0.3, mid)
	c.Set(11, 9, hi)
	return c
}

// Scorch is the burnt mark and scrap left by a destroyed robot (anchor at its centre).
func Scorch(variant int) *Canvas {
	c := NewCanvas(30, 22)
	r := rand.New(rand.NewSource(int64(variant*104729 + 11)))
	for py := 0; py < c.H; py++ {
		for px := 0; px < c.W; px++ {
			dx, dy := float64(px)+0.5-15, (float64(py)+0.5-11)/GY
			d := math.Hypot(dx/12.5, dy/10.5) + (noise(float64(px)*0.45, float64(py)*0.45, variant+5)-0.5)*0.5
			switch {
			case d < 0.55:
				c.Set(px, py, Ink)
			case d < 0.95:
				if dither(px, py) < 1-(d-0.55)/0.4 {
					c.Set(px, py, Plum)
				}
			}
		}
	}
	// Scattered scrap plates and a couple of glowing embers.
	for i := 0; i < 7; i++ {
		a := r.Float64() * 2 * math.Pi
		d := 2 + r.Float64()*9
		x, y := 15+math.Cos(a)*d, 11+math.Sin(a)*d*GY
		col := []color.RGBA{Rust, Steel, Slate, Brown}[r.Intn(4)]
		c.Rect(int(x), int(y), 1+r.Intn(2), 1, col)
	}
	for i := 0; i < 2; i++ {
		c.Set(12+r.Intn(7), 9+r.Intn(4), Orange)
	}
	return c
}

// IconArt renders the turret used for the application icon.
func IconArt() *Canvas {
	b := TurretBaseScene("blaster", 1).Bake(spriteOpts, false, 0, 0)
	h := TurretHeadScene("blaster", 1, -math.Pi/4).Bake(spriteOpts, false, 0, 0)
	ox, oy := max(b.OX, h.OX), max(b.OY, h.OY)
	w := max(b.C.W-b.OX, h.C.W-h.OX) + ox
	hh := max(b.C.H-b.OY, h.C.H-h.OY) + oy
	c := NewCanvas(w, hh)
	c.Stamp(b.C, ox-b.OX, oy-b.OY)
	c.Stamp(h.C, ox-h.OX, oy-h.OY)
	return c
}

// Sprite bakes for the game.

func Crawler(heading float64, f, n int) *Sprite {
	return CrawlerScene(heading, f, n).Bake(Opts{Inner: 2.5, Shadows: true}, true, 0, 0)
}
func Base(east bool) *Sprite  { return BaseScene(east).Bake(Opts{Inner: 3, Shadows: true}, false, 0, 0) }
func Radar(a float64) *Sprite { return RadarScene(a).Bake(Opts{Inner: 2}, false, 0, 0) }
