// Package art draws all game graphics procedurally as pixel art.
package art

import (
	"image"
	"image/color"
	"math"
)

func Hex(h uint32) color.RGBA { return color.RGBA{uint8(h >> 16), uint8(h >> 8), uint8(h), 255} }

// Palette: Endesga 32.
var (
	Ink      = Hex(0x181425)
	Navy     = Hex(0x262b44)
	Slate    = Hex(0x3a4466)
	Steel    = Hex(0x5a6988)
	Gray     = Hex(0x8b9bb4)
	Silver   = Hex(0xc0cbdc)
	White    = Hex(0xffffff)
	DeepTeal = Hex(0x193c3e)
	Forest   = Hex(0x265c42)
	Green    = Hex(0x3e8948)
	Lime     = Hex(0x63c74d)
	DeepBlue = Hex(0x124e89)
	Blue     = Hex(0x0099db)
	Cyan     = Hex(0x2ce8f5)
	Plum     = Hex(0x3e2731)
	Brown    = Hex(0x733e39)
	Clay     = Hex(0xb86f50)
	Tan      = Hex(0xe4a672)
	Sand     = Hex(0xead4aa)
	Maroon   = Hex(0xa22633)
	Red      = Hex(0xe43b44)
	Orange   = Hex(0xf77622)
	Amber    = Hex(0xfeae34)
	Yellow   = Hex(0xfee761)
	Purple   = Hex(0x68386c)
	Magenta  = Hex(0xb55088)
	Pink     = Hex(0xf6757a)
	Peach    = Hex(0xe8b796)
	Copper   = Hex(0xc28569)
	Rust     = Hex(0xbe4a2f)
	Ochre    = Hex(0xd77643)
	Crimson  = Hex(0xff0044)
	// Extra accents for the railgun (no enemy uses them).
	Violet = Hex(0x8a52d6)
	Lilac  = Hex(0xc9a6ff)
)

// Ramp is an ordered list of colours from dark to light.
type Ramp []color.RGBA

func (r Ramp) At(t float64) color.RGBA {
	i := int(math.Floor(t * float64(len(r))))
	if i < 0 {
		i = 0
	}
	if i >= len(r) {
		i = len(r) - 1
	}
	return r[i]
}

var (
	MetalRamp  = Ramp{Ink, Navy, Slate, Steel, Gray, Silver, White}
	DarkMetal  = Ramp{Ink, Navy, Slate, Steel, Gray}
	AlienRamp  = Ramp{Ink, DeepTeal, Forest, Green, Lime, Yellow}
	FleshRamp  = Ramp{Ink, Plum, Purple, Magenta, Pink}
	GlassRamp  = Ramp{Ink, DeepBlue, Blue, Cyan, White}
	EnergyRamp = Ramp{DeepBlue, Blue, Cyan, White}
	FireRamp   = Ramp{Maroon, Red, Orange, Amber, Yellow, White}
	EarthRamp  = Ramp{Ink, Plum, Brown, Clay, Copper}
	RockRamp   = Ramp{Ink, Navy, Slate, Steel, Gray, Silver}
	BoneRamp   = Ramp{Plum, Brown, Copper, Tan, Sand}
	GoldRamp   = Ramp{Plum, Brown, Ochre, Amber, Yellow}
	CrystalRmp = Ramp{DeepBlue, Blue, Cyan, White}
	PinkRamp   = Ramp{Plum, Purple, Magenta, Pink, Peach}
)

var light = func() [3]float64 {
	x, y, z := -0.5, -0.65, 0.58
	l := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / l, y / l, z / l}
}()

// Shade converts a surface normal into a ramp position (0..1).
func Shade(nx, ny, nz float64) float64 {
	d := nx*light[0] + ny*light[1] + nz*light[2]
	return 0.45 + 0.5*d
}

// Canvas is a small RGBA image with pixel-art drawing helpers.
type Canvas struct {
	Img  *image.RGBA
	W, H int
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{Img: image.NewRGBA(image.Rect(0, 0, w, h)), W: w, H: h}
}

func (c *Canvas) In(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H }

func (c *Canvas) Set(x, y int, col color.RGBA) {
	if !c.In(x, y) {
		return
	}
	i := c.Img.PixOffset(x, y)
	p := c.Img.Pix[i : i+4 : i+4]
	p[0], p[1], p[2], p[3] = col.R, col.G, col.B, col.A
}

func (c *Canvas) Get(x, y int) color.RGBA {
	if !c.In(x, y) {
		return color.RGBA{}
	}
	i := c.Img.PixOffset(x, y)
	p := c.Img.Pix[i : i+4 : i+4]
	return color.RGBA{p[0], p[1], p[2], p[3]}
}

// Blend composites an opaque colour with coverage a over the pixel.
func (c *Canvas) Blend(x, y int, col color.RGBA, a float64) {
	if !c.In(x, y) || a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := c.Img.PixOffset(x, y)
	p := c.Img.Pix[i : i+4 : i+4]
	k := 1 - a
	p[0] = uint8(float64(col.R)*a + float64(p[0])*k)
	p[1] = uint8(float64(col.G)*a + float64(p[1])*k)
	p[2] = uint8(float64(col.B)*a + float64(p[2])*k)
	p[3] = uint8(255*a + float64(p[3])*k)
}

func (c *Canvas) Solid(x, y int) bool { return c.Get(x, y).A >= 200 }

// bbox iterates over pixel centres inside a bounding box.
func (c *Canvas) bbox(cx, cy, r float64, f func(x, y int, px, py float64)) {
	x0, x1 := int(math.Floor(cx-r-1)), int(math.Ceil(cx+r+1))
	y0, y1 := int(math.Floor(cy-r-1)), int(math.Ceil(cy+r+1))
	for y := max(y0, 0); y <= min(y1, c.H-1); y++ {
		for x := max(x0, 0); x <= min(x1, c.W-1); x++ {
			f(x, y, float64(x)+0.5, float64(y)+0.5)
		}
	}
}

// Ball draws a shaded, rotated ellipse. flat (0..1) flattens the dome.
func (c *Canvas) Ball(cx, cy, rx, ry, rot float64, ramp Ramp, flat, bias float64) {
	cs, sn := math.Cos(rot), math.Sin(rot)
	c.bbox(cx, cy, math.Max(rx, ry), func(x, y int, px, py float64) {
		dx, dy := px-cx, py-cy
		u := (dx*cs + dy*sn) / rx
		v := (-dx*sn + dy*cs) / ry
		r2 := u*u + v*v
		if r2 > 1 {
			return
		}
		nz := math.Sqrt(1 - r2)
		nx := (u*cs - v*sn) * (1 - flat)
		ny := (u*sn + v*cs) * (1 - flat)
		l := math.Sqrt(nx*nx + ny*ny + nz*nz)
		c.Set(x, y, ramp.At(Shade(nx/l, ny/l, nz/l)+bias))
	})
}

// Pit draws a concave (inverted) shaded ellipse, like a hole or crater.
func (c *Canvas) Pit(cx, cy, rx, ry float64, ramp Ramp, bias float64) {
	c.bbox(cx, cy, math.Max(rx, ry), func(x, y int, px, py float64) {
		u, v := (px-cx)/rx, (py-cy)/ry
		r2 := u*u + v*v
		if r2 > 1 {
			return
		}
		nz := math.Sqrt(1 - r2)
		c.Set(x, y, ramp.At(Shade(-u*0.8, -v*0.8, nz)+bias))
	})
}

// Box draws a rotated rectangle with bevelled edges. chamfer cuts the corners.
func (c *Canvas) Box(cx, cy, w, h, rot float64, ramp Ramp, bevel, chamfer, bias float64) {
	cs, sn := math.Cos(rot), math.Sin(rot)
	hw, hh := w/2, h/2
	c.bbox(cx, cy, math.Hypot(hw, hh), func(x, y int, px, py float64) {
		dx, dy := px-cx, py-cy
		lx := dx*cs + dy*sn
		ly := -dx*sn + dy*cs
		ax, ay := math.Abs(lx), math.Abs(ly)
		if ax > hw || ay > hh {
			return
		}
		ed := math.Inf(1)
		if chamfer > 0 {
			ed = (hw + hh - chamfer - ax - ay) / math.Sqrt2
			if ed < 0 {
				return
			}
		}
		ex, ey := hw-ax, hh-ay
		var nlx, nly, nz float64 = 0, 0, 1
		switch {
		case ed < bevel && ed <= ex && ed <= ey:
			nlx, nly, nz = math.Copysign(0.53, lx), math.Copysign(0.53, ly), 0.66
		case ex < bevel && ex <= ey:
			nlx, nz = math.Copysign(0.75, lx), 0.66
		case ey < bevel:
			nly, nz = math.Copysign(0.75, ly), 0.66
		}
		nx := nlx*cs - nly*sn
		ny := nlx*sn + nly*cs
		c.Set(x, y, ramp.At(Shade(nx, ny, nz)+bias))
	})
}

// Rod draws a shaded capsule between two points.
func (c *Canvas) Rod(x0, y0, x1, y1, r float64, ramp Ramp, bias float64) {
	ax, ay := x1-x0, y1-y0
	l2 := ax*ax + ay*ay
	cx, cy := (x0+x1)/2, (y0+y1)/2
	c.bbox(cx, cy, math.Sqrt(l2)/2+r, func(x, y int, px, py float64) {
		t := 0.0
		if l2 > 0 {
			t = math.Max(0, math.Min(1, ((px-x0)*ax+(py-y0)*ay)/l2))
		}
		dx, dy := px-(x0+ax*t), py-(y0+ay*t)
		d := math.Hypot(dx, dy)
		if d > r {
			return
		}
		nx, ny := dx/r, dy/r
		nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
		c.Set(x, y, ramp.At(Shade(nx, ny, nz)+bias))
	})
}

// Disc fills an ellipse with a flat colour and coverage alpha.
func (c *Canvas) Disc(cx, cy, rx, ry float64, col color.RGBA, alpha float64) {
	c.bbox(cx, cy, math.Max(rx, ry), func(x, y int, px, py float64) {
		u, v := (px-cx)/rx, (py-cy)/ry
		if u*u+v*v <= 1 {
			if alpha >= 1 {
				c.Set(x, y, col)
			} else {
				c.Blend(x, y, col, alpha)
			}
		}
	})
}

// Ring draws a flat elliptical ring between radii r0 and r1 (as fractions of rx/ry scale).
func (c *Canvas) Ring(cx, cy, r0, r1 float64, col color.RGBA, alpha float64) {
	c.bbox(cx, cy, r1, func(x, y int, px, py float64) {
		d := math.Hypot(px-cx, py-cy)
		if d >= r0 && d <= r1 {
			c.Blend(x, y, col, alpha)
		}
	})
}

// Line draws a 1-pixel line with a flat colour.
func (c *Canvas) Line(x0, y0, x1, y1 float64, col color.RGBA) {
	n := int(math.Ceil(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		c.Set(int(math.Floor(x0+(x1-x0)*t)), int(math.Floor(y0+(y1-y0)*t)), col)
	}
}

func (c *Canvas) Rect(x, y, w, h int, col color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.Set(i, j, col)
		}
	}
}

// Poly fills a polygon (pixel centres, even-odd rule).
func (c *Canvas) Poly(pts [][2]float64, col color.RGBA) {
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		minY, maxY = math.Min(minY, p[1]), math.Max(maxY, p[1])
	}
	for y := max(int(minY), 0); y <= min(int(maxY)+1, c.H-1); y++ {
		py := float64(y) + 0.5
		var xs []float64
		for i := range pts {
			a, b := pts[i], pts[(i+1)%len(pts)]
			if (a[1] <= py) != (b[1] <= py) {
				xs = append(xs, a[0]+(py-a[1])/(b[1]-a[1])*(b[0]-a[0]))
			}
		}
		for i := 0; i+1 < len(xs); i++ {
			for j := i + 1; j < len(xs); j++ {
				if xs[j] < xs[i] {
					xs[i], xs[j] = xs[j], xs[i]
				}
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			for x := max(int(math.Ceil(xs[i]-0.5)), 0); x <= min(int(math.Floor(xs[i+1]-0.5)), c.W-1); x++ {
				c.Set(x, y, col)
			}
		}
	}
}

// Outline surrounds solid pixels with a 1-pixel border.
func (c *Canvas) Outline(col color.RGBA) {
	solid := make([]bool, c.W*c.H)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			solid[y*c.W+x] = c.Solid(x, y)
		}
	}
	at := func(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H && solid[y*c.W+x] }
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if solid[y*c.W+x] {
				continue
			}
			if at(x-1, y) || at(x+1, y) || at(x, y-1) || at(x, y+1) {
				c.Set(x, y, col)
			}
		}
	}
}

// Silhouette returns a white copy of the solid pixels (used for flashes).
func (c *Canvas) Silhouette() *Canvas {
	s := NewCanvas(c.W, c.H)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if a := c.Get(x, y).A; a > 0 {
				s.Set(x, y, color.RGBA{a, a, a, a})
			}
		}
	}
	return s
}

// rot rotates a local point around the canvas centre.
func rot(cx, cy, lx, ly, a float64) (float64, float64) {
	cs, sn := math.Cos(a), math.Sin(a)
	return cx + lx*cs - ly*sn, cy + lx*sn + ly*cs
}

// hash returns a deterministic pseudo-random value in [0,1).
func hash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h%100000) / 100000
}

// noise is smooth value noise in [0,1).
func noise(x, y float64, seed int) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int(x0), int(y0)
	a := hash(ix, iy, seed)
	b := hash(ix+1, iy, seed)
	c := hash(ix, iy+1, seed)
	d := hash(ix+1, iy+1, seed)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

// Color is the colour type used by all art.
type Color = color.RGBA
