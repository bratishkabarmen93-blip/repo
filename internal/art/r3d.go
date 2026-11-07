package art

import (
	"image/color"
	"math"
)

// The whole game uses one fixed 2.5D projection: x stays as it is, ground depth
// is squashed by GY and height rises by HZ. World units are pixels (32 per cell).
const (
	GY = 0.75
	HZ = 0.8
)

// Project maps a world point (z up) onto the projected plane.
func Project(x, y, z float64) (float64, float64) { return x, y*GY - z*HZ }

type V3 struct{ X, Y, Z float64 }

func V(x, y, z float64) V3           { return V3{x, y, z} }
func (a V3) Add(b V3) V3             { return V3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a V3) Sub(b V3) V3             { return V3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a V3) Mul(s float64) V3        { return V3{a.X * s, a.Y * s, a.Z * s} }
func (a V3) Dot(b V3) float64        { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a V3) Scale(b V3) V3           { return V3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a V3) Cross(b V3) V3           { return V3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X} }
func (a V3) Len() float64            { return math.Sqrt(a.Dot(a)) }
func (a V3) Neg() V3                 { return V3{-a.X, -a.Y, -a.Z} }
func (a V3) Lerp(b V3, t float64) V3 { return a.Add(b.Sub(a).Mul(t)) }
func (a V3) Norm() V3 {
	l := a.Len()
	if l == 0 {
		return a
	}
	return a.Mul(1 / l)
}

// M3 is a rotation matrix stored by rows (local to world when used on parts).
type M3 [3]V3

var Ident = M3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func (m M3) Apply(v V3) V3 { return V3{m[0].Dot(v), m[1].Dot(v), m[2].Dot(v)} }
func (m M3) T() M3 {
	return M3{{m[0].X, m[1].X, m[2].X}, {m[0].Y, m[1].Y, m[2].Y}, {m[0].Z, m[1].Z, m[2].Z}}
}
func (m M3) Mul(n M3) M3 {
	t := n.T()
	var r M3
	for i := 0; i < 3; i++ {
		r[i] = V3{m[i].Dot(t[0]), m[i].Dot(t[1]), m[i].Dot(t[2])}
	}
	return r
}

// Yaw turns around the vertical axis, in the same sense as game angles (x towards y).
func Yaw(a float64) M3 {
	c, s := math.Cos(a), math.Sin(a)
	return M3{{c, -s, 0}, {s, c, 0}, {0, 0, 1}}
}

// Pitch lifts the +x axis upwards.
func Pitch(a float64) M3 {
	c, s := math.Cos(a), math.Sin(a)
	return M3{{c, 0, -s}, {0, 1, 0}, {s, 0, c}}
}

// Roll turns around the x axis.
func Roll(a float64) M3 {
	c, s := math.Cos(a), math.Sin(a)
	return M3{{1, 0, 0}, {0, c, -s}, {0, s, c}}
}

// alongZ returns a rotation whose local z axis points along w.
func alongZ(w V3) M3 {
	w = w.Norm()
	a := V3{1, 0, 0}
	if math.Abs(w.X) > 0.9 {
		a = V3{0, 1, 0}
	}
	u := a.Cross(w).Norm()
	v := w.Cross(u)
	return M3{{u.X, v.X, w.X}, {u.Y, v.Y, w.Y}, {u.Z, v.Z, w.Z}}
}

var (
	viewDir = V3{0, -HZ, -GY}.Norm()
	toEye   = viewDir.Neg()
	// Light comes from the upper left and slightly from behind, so tops are bright,
	// front faces are mid-tones and shadows fall to the lower right.
	Light   = V3{-0.62, -0.32, 0.72}.Norm()
	halfVec = Light.Add(toEye).Norm()
)

// Mat describes how a surface is shaded.
type Mat struct {
	Ramp Ramp
	Bias float64
	Spec float64 // strength of the specular highlight
	Glow bool    // emissive: ignores light and shadow
}

func (m *Mat) shade(n V3, lit bool, d float64) color.RGBA {
	if m.Glow {
		return m.Ramp.At(m.Bias + 0.55 + 0.45*n.Dot(toEye) + d)
	}
	nl := n.Dot(Light)
	if !lit && nl > 0 {
		nl *= 0.1
	}
	l := 0.42 + 0.5*nl + m.Bias + d
	if m.Spec > 0 && lit {
		l += m.Spec * math.Pow(math.Max(0, n.Dot(halfVec)), 18)
	}
	return m.Ramp.At(l)
}

// span is the interval where a ray is inside a convex shape, with outward normals.
type span struct {
	t0, t1 float64
	n0, n1 V3
	cap0   bool // entry face comes from a clip plane
}

type shape interface {
	span(o, d V3) (span, bool)
	bound() float64 // radius of a bounding sphere around the local origin
}

type ellipsoid struct{ r V3 }

func (e ellipsoid) bound() float64 { return math.Max(e.r.X, math.Max(e.r.Y, e.r.Z)) }
func (e ellipsoid) span(o, d V3) (span, bool) {
	ir := V3{1 / e.r.X, 1 / e.r.Y, 1 / e.r.Z}
	oo, dd := o.Scale(ir), d.Scale(ir)
	a, b, c := dd.Dot(dd), oo.Dot(dd), oo.Dot(oo)-1
	disc := b*b - a*c
	if disc < 0 {
		return span{}, false
	}
	s := math.Sqrt(disc)
	t0, t1 := (-b-s)/a, (-b+s)/a
	ir2 := ir.Scale(ir)
	return span{t0: t0, t1: t1,
		n0: o.Add(d.Mul(t0)).Scale(ir2).Norm(), n1: o.Add(d.Mul(t1)).Scale(ir2).Norm()}, true
}

// cylinder stands on the local origin and rises along z.
type cylinder struct{ r, h float64 }

func (c cylinder) bound() float64 { return math.Hypot(c.r, c.h) }
func (c cylinder) span(o, d V3) (span, bool) {
	s := span{t0: math.Inf(-1), t1: math.Inf(1)}
	a := d.X*d.X + d.Y*d.Y
	if a > 1e-12 {
		b := o.X*d.X + o.Y*d.Y
		cc := o.X*o.X + o.Y*o.Y - c.r*c.r
		disc := b*b - a*cc
		if disc < 0 {
			return s, false
		}
		q := math.Sqrt(disc)
		s.t0, s.t1 = (-b-q)/a, (-b+q)/a
		p0, p1 := o.Add(d.Mul(s.t0)), o.Add(d.Mul(s.t1))
		s.n0, s.n1 = V3{p0.X, p0.Y, 0}.Norm(), V3{p1.X, p1.Y, 0}.Norm()
	} else if o.X*o.X+o.Y*o.Y > c.r*c.r {
		return s, false
	}
	return slab(s, o.Z, d.Z, 0, c.h)
}

func slab(s span, oz, dz, lo, hi float64) (span, bool) {
	if math.Abs(dz) < 1e-12 {
		return s, oz >= lo && oz <= hi && s.t0 <= s.t1
	}
	ta, tb := (lo-oz)/dz, (hi-oz)/dz
	na, nb := V3{0, 0, -1}, V3{0, 0, 1}
	if ta > tb {
		ta, tb, na, nb = tb, ta, nb, na
	}
	if ta > s.t0 {
		s.t0, s.n0 = ta, na
	}
	if tb < s.t1 {
		s.t1, s.n1 = tb, nb
	}
	return s, s.t0 <= s.t1
}

type plane struct {
	n V3
	d float64
}

// convex is an intersection of half-spaces n·p <= d. smooth (optional) replaces
// the faceted normal so polygonal lathes shade like round objects.
type convex struct {
	pl     []plane
	r      float64
	smooth func(p, n V3) V3
}

func (c *convex) bound() float64 { return c.r }
func (c *convex) span(o, d V3) (span, bool) {
	s := span{t0: math.Inf(-1), t1: math.Inf(1)}
	for _, p := range c.pl {
		den := p.n.Dot(d)
		dist := p.d - p.n.Dot(o)
		if math.Abs(den) < 1e-12 {
			if dist < 0 {
				return s, false
			}
			continue
		}
		t := dist / den
		if den < 0 {
			if t > s.t0 {
				s.t0, s.n0 = t, p.n
			}
		} else if t < s.t1 {
			s.t1, s.n1 = t, p.n
		}
		if s.t0 > s.t1 {
			return s, false
		}
	}
	if c.smooth != nil {
		s.n0 = c.smooth(o.Add(d.Mul(s.t0)), s.n0)
		s.n1 = c.smooth(o.Add(d.Mul(s.t1)), s.n1)
	}
	return s, true
}

// capsule is a rounded rod from the local origin along +z with length h.
type capsule struct{ r, h float64 }

func (c capsule) bound() float64 { return c.h + c.r }
func (c capsule) span(o, d V3) (span, bool) {
	best := span{t0: math.Inf(1), t1: math.Inf(-1)}
	found := false
	merge := func(s span, ok bool) {
		if !ok {
			return
		}
		found = true
		if s.t0 < best.t0 {
			best.t0, best.n0 = s.t0, s.n0
		}
		if s.t1 > best.t1 {
			best.t1, best.n1 = s.t1, s.n1
		}
	}
	merge(cylinder{c.r, c.h}.span(o, d))
	sp := ellipsoid{V3{c.r, c.r, c.r}}
	merge(sp.span(o, d))
	merge(sp.span(o.Sub(V3{0, 0, c.h}), d))
	return best, found
}

// Part is one primitive placed in the world, optionally clipped and carved.
type Part struct {
	sh    shape
	pos   V3
	rot   M3 // local -> world
	inv   M3
	M     *Mat
	Cap   *Mat               // faces created by clip planes
	Cut   *Mat               // faces carved by Sub parts
	Paint func(p, n V3) *Mat // optional pattern in local coordinates
	sub   []*Part
	clip  []plane
	bc    V3
	br    float64
}

func newPart(sh shape, pos V3, rot M3, m *Mat) *Part {
	p := &Part{sh: sh, pos: pos, rot: rot, inv: rot.T(), M: m}
	p.bc, p.br = pos, sh.bound()
	return p
}

// Clip keeps only the local half-space n·p <= d (n is normalised).
func (p *Part) Clip(n V3, d float64) *Part {
	l := n.Len()
	p.clip = append(p.clip, plane{n.Mul(1 / l), d / l})
	return p
}

// Sub carves the given parts out of p (their own materials are ignored).
func (p *Part) Sub(cut *Mat, q ...*Part) *Part {
	p.Cut = cut
	p.sub = append(p.sub, q...)
	return p
}

func (p *Part) WithPaint(f func(p, n V3) *Mat) *Part { p.Paint = f; return p }
func (p *Part) WithCap(m *Mat) *Part                 { p.Cap = m; return p }

func (p *Part) worldSpan(o, d V3) (span, bool) {
	oc := o.Sub(p.bc)
	b := oc.Dot(d)
	if b*b-(oc.Dot(oc)-p.br*p.br) < 0 {
		return span{}, false
	}
	lo, ld := p.inv.Apply(o.Sub(p.pos)), p.inv.Apply(d)
	s, ok := p.sh.span(lo, ld)
	if !ok {
		return s, false
	}
	for _, c := range p.clip {
		den := c.n.Dot(ld)
		dist := c.d - c.n.Dot(lo)
		if math.Abs(den) < 1e-12 {
			if dist < 0 {
				return s, false
			}
			continue
		}
		t := dist / den
		if den < 0 {
			if t > s.t0 {
				s.t0, s.n0, s.cap0 = t, c.n, true
			}
		} else if t < s.t1 {
			s.t1, s.n1 = t, c.n
		}
	}
	if s.t0 > s.t1 {
		return s, false
	}
	s.n0, s.n1 = p.rot.Apply(s.n0), p.rot.Apply(s.n1)
	return s, true
}

const eps = 1e-4

// trace finds the first surface of the part along the ray with t > tmin.
func (p *Part) trace(o, d V3, tmin float64) (float64, V3, *Mat, bool) {
	s, ok := p.worldSpan(o, d)
	if !ok || s.t1 < tmin {
		return 0, V3{}, nil, false
	}
	t, n, m := s.t0, s.n0, p.M
	if s.cap0 && p.Cap != nil {
		m = p.Cap
	}
	if t < tmin {
		t = tmin // the ray starts inside
	}
	for range len(p.sub) + 1 {
		moved := false
		for _, q := range p.sub {
			if qs, ok := q.worldSpan(o, d); ok && qs.t0 <= t && t <= qs.t1 {
				t, n, m, moved = qs.t1, qs.n1.Neg(), p.Cut, true
			}
		}
		if !moved {
			break
		}
	}
	if t > s.t1 {
		return 0, V3{}, nil, false
	}
	if m == nil {
		m = p.M
	}
	if p.Paint != nil && m == p.M {
		hp := o.Add(d.Mul(t))
		if pm := p.Paint(p.inv.Apply(hp.Sub(p.pos)), p.inv.Apply(n)); pm != nil {
			m = pm
		}
	}
	return t, n, m, true
}

// Primitive constructors (positions in world pixels, z up).

func Ellipsoid(c V3, r V3, rot M3, m *Mat) *Part { return newPart(ellipsoid{r}, c, rot, m) }
func Sphere(c V3, r float64, m *Mat) *Part       { return Ellipsoid(c, V3{r, r, r}, Ident, m) }

// Cylinder runs from a to b with radius r.
func Cylinder(a, b V3, r float64, m *Mat) *Part {
	return newPart(cylinder{r, b.Sub(a).Len()}, a, alongZ(b.Sub(a)), m)
}

// Rod is a capsule from a to b with radius r.
func Rod(a, b V3, r float64, m *Mat) *Part {
	return newPart(capsule{r, b.Sub(a).Len()}, a, alongZ(b.Sub(a)), m)
}

// Lathe is an n-sided frustum standing on c (radius r0 at the bottom, r1 at height h),
// shaded smoothly like a round object.
func Lathe(c V3, r0, r1, h float64, n int, rot M3, m *Mat) *Part {
	cv := &convex{r: math.Hypot(math.Max(r0, r1), h)}
	k := (r1 - r0) / h
	for i := 0; i < n; i++ {
		a := (float64(i) + 0.5) / float64(n) * 2 * math.Pi
		nx, ny := math.Cos(a), math.Sin(a)
		// Plane through the polygon edge; the polygon is circumscribed around the circle.
		nn := V3{nx, ny, -k}.Norm()
		cv.pl = append(cv.pl, plane{nn, r0 * nn.Dot(V3{nx, ny, 0})})
	}
	cv.pl = append(cv.pl, plane{V3{0, 0, 1}, h}, plane{V3{0, 0, -1}, 0})
	cv.smooth = func(p, fn V3) V3 {
		if math.Abs(fn.Z) > 0.999 {
			return fn
		}
		return V3{p.X, p.Y, -(r0 + k*p.Z) * k}.Norm()
	}
	return newPart(cv, c, rot, m)
}

// Prism is an n-sided faceted frustum (flat shaded) standing on c.
func Prism(c V3, r0, r1, h float64, n int, rot M3, m *Mat) *Part {
	p := Lathe(c, r0, r1, h, n, rot, m)
	p.sh.(*convex).smooth = nil
	return p
}

// Block is a box centred on c with half sizes hs, chamfered vertical edges and a bevelled top.
func Block(c, hs V3, rot M3, chamfer, bevel float64, m *Mat) *Part {
	cv := &convex{r: hs.Len()}
	cv.pl = []plane{
		{V3{1, 0, 0}, hs.X}, {V3{-1, 0, 0}, hs.X},
		{V3{0, 1, 0}, hs.Y}, {V3{0, -1, 0}, hs.Y},
		{V3{0, 0, 1}, hs.Z}, {V3{0, 0, -1}, hs.Z},
	}
	if chamfer > 0 {
		for _, s := range [][2]float64{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			n := V3{s[0], s[1], 0}.Norm()
			cv.pl = append(cv.pl, plane{n, (hs.X + hs.Y - chamfer) / math.Sqrt2})
		}
	}
	if bevel > 0 {
		for _, s := range []V3{{1, 0, 1}, {-1, 0, 1}, {0, 1, 1}, {0, -1, 1}} {
			n := s.Norm()
			ext := hs.X
			if s.Y != 0 {
				ext = hs.Y
			}
			cv.pl = append(cv.pl, plane{n, (ext + hs.Z - bevel) / math.Sqrt2})
		}
	}
	return newPart(cv, c, rot, m)
}

// Shard is a hexagonal crystal: a prism of height h topped by a pyramid of height tip.
func Shard(c V3, r, h, tip float64, rot M3, m *Mat) *Part {
	cv := &convex{r: h + tip + r}
	for i := 0; i < 6; i++ {
		a := float64(i) / 6 * 2 * math.Pi
		nx, ny := math.Cos(a), math.Sin(a)
		cv.pl = append(cv.pl, plane{V3{nx, ny, 0}, r})
		nn := V3{nx * tip, ny * tip, r}.Norm()
		cv.pl = append(cv.pl, plane{nn, nn.Dot(V3{0, 0, h + tip})})
	}
	cv.pl = append(cv.pl, plane{V3{0, 0, -1}, 0})
	return newPart(cv, c, rot, m)
}

// Scene is a set of parts rendered together.
type Scene struct{ Parts []*Part }

func (s *Scene) Add(ps ...*Part) { s.Parts = append(s.Parts, ps...) }

func (s *Scene) trace(o, d V3, tmin float64) (float64, V3, *Mat, bool) {
	best, bn, bm, hit := math.Inf(1), V3{}, (*Mat)(nil), false
	for _, p := range s.Parts {
		if t, n, m, ok := p.trace(o, d, tmin); ok && t < best {
			best, bn, bm, hit = t, n, m, true
		}
	}
	return best, bn, bm, hit
}

func (s *Scene) occluded(o, d V3) bool {
	for _, p := range s.Parts {
		if _, _, _, ok := p.trace(o, d, eps); ok {
			return true
		}
	}
	return false
}

// bounds returns the projected bounding box of the scene.
func (s *Scene) bounds() (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	ry := math.Hypot(GY, HZ)
	for _, p := range s.Parts {
		px, py := Project(p.bc.X, p.bc.Y, p.bc.Z)
		x0, x1 = math.Min(x0, px-p.br), math.Max(x1, px+p.br)
		y0, y1 = math.Min(y0, py-p.br*ry), math.Max(y1, py+p.br*ry)
	}
	return
}

// shadowBounds returns the projected box of the shadow cast onto z = 0.
func (s *Scene) shadowBounds() (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range s.Parts {
		g := p.bc.Sub(Light.Mul(p.bc.Z / Light.Z))
		r := p.br / Light.Z
		x0, x1 = math.Min(x0, g.X-r), math.Max(x1, g.X+r)
		y0, y1 = math.Min(y0, (g.Y-r)*GY), math.Max(y1, (g.Y+r)*GY)
	}
	return
}

// View maps canvas pixels to the projected plane: pixel (OX, OY) is the world origin.
// S is the number of canvas pixels per world pixel (0 means 1).
type View struct {
	W, H   int
	OX, OY float64
	S      float64
}

func (v View) scale() float64 {
	if v.S <= 0 {
		return 1
	}
	return v.S
}

func (v View) ray(i, j int) (V3, V3) {
	k := v.scale()
	sx := (float64(i) + 0.5 - v.OX) / k
	sy := (float64(j) + 0.5 - v.OY) / k
	const z = 600
	return V3{sx, (sy + z*HZ) / GY, z}, viewDir
}

// Opts tunes a sprite render.
type Opts struct {
	Inner   float64 // depth jump that draws an inner outline (0 = none)
	Shadows bool    // self shadowing
	Dither  float64
	NoLine  bool // skip the outer outline
}

// Draw renders the scene into a canvas of the given view.
func (s *Scene) Draw(v View, o Opts) *Canvas {
	c := NewCanvas(v.W, v.H)
	n := v.W * v.H
	depth := make([]float64, n)
	mats := make([]*Mat, n)
	for j := 0; j < v.H; j++ {
		for i := 0; i < v.W; i++ {
			k := j*v.W + i
			depth[k] = math.Inf(1)
			ro, rd := v.ray(i, j)
			t, nn, m, ok := s.trace(ro, rd, 0)
			if !ok {
				continue
			}
			lit := true
			if o.Shadows && !m.Glow && nn.Dot(Light) > 0 {
				p := ro.Add(rd.Mul(t)).Add(nn.Mul(0.05))
				lit = !s.occluded(p, Light)
			}
			d := 0.0
			if o.Dither > 0 {
				d = (dither(i, j) - 0.5) * o.Dither
			}
			depth[k], mats[k] = t, m
			c.Set(i, j, m.shade(nn, lit, d))
		}
	}
	if o.Inner > 0 {
		edge := make([]bool, n)
		for j := 0; j < v.H; j++ {
			for i := 0; i < v.W; i++ {
				k := j*v.W + i
				if mats[k] == nil {
					continue
				}
				for _, q := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					x, y := i+q[0], j+q[1]
					if x < 0 || y < 0 || x >= v.W || y >= v.H {
						continue
					}
					kk := y*v.W + x
					if mats[kk] != nil && depth[kk]-depth[k] > o.Inner {
						edge[kk] = true
					}
				}
			}
		}
		for k, e := range edge {
			if e && !mats[k].Glow {
				c.Set(k%v.W, k/v.W, mats[k].Ramp[0])
			}
		}
	}
	if !o.NoLine {
		c.Outline(Ink)
	}
	return c
}

// ShadowMask renders where the scene shadows the ground plane z = 0 (opaque black).
func (s *Scene) ShadowMask(v View) *Canvas {
	c := NewCanvas(v.W, v.H)
	for j := 0; j < v.H; j++ {
		for i := 0; i < v.W; i++ {
			sx := float64(i) + 0.5 - v.OX
			sy := float64(j) + 0.5 - v.OY
			if s.occluded(V3{sx, sy / GY, 0}, Light) {
				c.Set(i, j, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return c
}

// Sprite is a rendered model with its anchor (the canvas pixel of the model origin)
// and the shadow it casts on the ground plane.
type Sprite struct {
	C      *Canvas
	OX, OY int
	Sh     *Canvas
	SX, SY int
}

// Bake renders a scene with automatically sized canvases. fx, fy shift the model
// by a sub-pixel amount (used when stamping into the terrain).
func (s *Scene) Bake(o Opts, shadow bool, fx, fy float64) *Sprite {
	x0, y0, x1, y1 := s.bounds()
	sp := &Sprite{}
	sp.OX, sp.OY = int(math.Ceil(-x0))+2, int(math.Ceil(-y0))+2
	w, h := sp.OX+int(math.Ceil(x1))+2, sp.OY+int(math.Ceil(y1))+2
	sp.C = s.Draw(View{W: w, H: h, OX: float64(sp.OX) + fx, OY: float64(sp.OY) + fy}, o)
	sp.C, sp.OX, sp.OY = trim(sp.C, sp.OX, sp.OY)
	if shadow {
		x0, y0, x1, y1 = s.shadowBounds()
		sp.SX, sp.SY = int(math.Ceil(-x0))+1, int(math.Ceil(-y0))+1
		w, h = sp.SX+int(math.Ceil(x1))+1, sp.SY+int(math.Ceil(y1))+1
		sp.Sh = s.ShadowMask(View{W: w, H: h, OX: float64(sp.SX) + fx, OY: float64(sp.SY) + fy})
		sp.Sh, sp.SX, sp.SY = trim(sp.Sh, sp.SX, sp.SY)
	}
	return sp
}

// BakeScaled renders the scene at k canvas pixels per world pixel (no shadow),
// used for crisp UI portraits of any size.
func (s *Scene) BakeScaled(o Opts, k float64) *Sprite {
	x0, y0, x1, y1 := s.bounds()
	sp := &Sprite{}
	sp.OX, sp.OY = int(math.Ceil(-x0*k))+2, int(math.Ceil(-y0*k))+2
	w, h := sp.OX+int(math.Ceil(x1*k))+2, sp.OY+int(math.Ceil(y1*k))+2
	sp.C = s.Draw(View{W: w, H: h, OX: float64(sp.OX), OY: float64(sp.OY), S: k}, o)
	sp.C, sp.OX, sp.OY = trim(sp.C, sp.OX, sp.OY)
	return sp
}

// trim crops transparent borders and keeps the anchor in place.
func trim(c *Canvas, ox, oy int) (*Canvas, int, int) {
	x0, y0, x1, y1 := c.W, c.H, -1, -1
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if c.Get(x, y).A > 0 {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return NewCanvas(1, 1), 0, 0
	}
	out := NewCanvas(x1-x0+1, y1-y0+1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			out.Set(x-x0, y-y0, c.Get(x, y))
		}
	}
	return out, ox - x0, oy - y0
}

// Lift moves every part of the scene up by z.
func Lift(s *Scene, z float64) {
	for _, p := range s.Parts {
		p.move(V3{0, 0, z})
	}
}

func (p *Part) move(d V3) {
	p.pos, p.bc = p.pos.Add(d), p.bc.Add(d)
	for _, q := range p.sub {
		q.move(d)
	}
}
