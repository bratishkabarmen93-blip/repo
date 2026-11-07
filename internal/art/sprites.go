package art

import (
	"image/color"
	"math"
)

const CellPx = 32

// Oval fills a rotated ellipse with a flat colour.
func (c *Canvas) Oval(cx, cy, rx, ry, r float64, col color.RGBA, alpha float64) {
	cs, sn := math.Cos(r), math.Sin(r)
	c.bbox(cx, cy, math.Max(rx, ry), func(x, y int, px, py float64) {
		dx, dy := px-cx, py-cy
		u := (dx*cs + dy*sn) / rx
		v := (-dx*sn + dy*cs) / ry
		if u*u+v*v <= 1 {
			c.Blend(x, y, col, alpha)
		}
	})
}

// Stamp copies the visible pixels of src onto c at (x, y).
func (c *Canvas) Stamp(src *Canvas, x, y int) {
	for j := 0; j < src.H; j++ {
		for i := 0; i < src.W; i++ {
			p := src.Get(i, j)
			if p.A == 0 {
				continue
			}
			if p.A == 255 {
				c.Set(x+i, y+j, p)
				continue
			}
			a := float64(p.A) / 255
			// src is premultiplied; un-premultiply for Blend.
			c.Blend(x+i, y+j, color.RGBA{uint8(float64(p.R) / a), uint8(float64(p.G) / a), uint8(float64(p.B) / a), 255}, a)
		}
	}
}

// Muzzle is a short-lived muzzle flash (14x14), meant for additive blending.
func Muzzle() *Canvas {
	c := NewCanvas(14, 14)
	c.Rod(1.5, 7, 12.5, 7, 0.8, Ramp{Blue, Cyan, White}, 0)
	c.Rod(7, 1.5, 7, 12.5, 0.8, Ramp{Blue, Cyan, White}, 0)
	c.Disc(7, 7, 2.8, 2.8, Cyan, 1)
	c.Disc(7, 7, 1.6, 1.6, White, 1)
	return c
}

// Bolt is a plasma projectile (16x16).
func Bolt(level int, a float64) *Canvas {
	c := NewCanvas(16, 16)
	s := 1 + 0.18*float64(level)
	c.Oval(8, 8, 3.8*s, 1.8*s, a, Blue, 1)
	c.Oval(8, 8, 2.9*s, 1.15*s, a, Cyan, 1)
	c.Oval(8, 8, 1.6*s, 0.6*s, a, White, 1)
	return c
}

// Plus is a small cross used for repair particles (5x5).
func Plus(col color.RGBA) *Canvas {
	c := NewCanvas(5, 5)
	c.Rect(2, 0, 1, 5, col)
	c.Rect(0, 2, 5, 1, col)
	return c
}

// Shield is the small armour badge drawn next to health bars (5x6).
func Shield() *Canvas {
	c := NewCanvas(7, 8)
	c.Poly([][2]float64{{0.5, 0.5}, {6.5, 0.5}, {6.5, 4.2}, {3.5, 7.6}, {0.5, 4.2}}, Ink)
	c.Poly([][2]float64{{1.5, 1.5}, {5.5, 1.5}, {5.5, 4}, {3.5, 6.2}, {1.5, 4}}, Gray)
	c.Poly([][2]float64{{1.5, 1.5}, {3.5, 1.5}, {3.5, 6.2}, {1.5, 4}}, Silver)
	return c
}

// Glow is a soft round light (32x32), meant for additive blending.
func Glow(col color.RGBA) *Canvas {
	c := NewCanvas(32, 32)
	for r := 15.0; r > 0; r -= 3 {
		c.Disc(16, 16, r, r, col, 0.22)
	}
	return c
}

// Icon draws small HUD icons (11x11).
func Icon(name string) *Canvas {
	c := NewCanvas(11, 11)
	switch name {
	case "hp":
		c.Poly([][2]float64{{1.5, 1.5}, {9.5, 1.5}, {9.5, 5.5}, {5.5, 9.8}, {1.5, 5.5}}, Red)
		c.Poly([][2]float64{{2.5, 2.5}, {5.5, 2.5}, {5.5, 7.5}, {2.5, 5}}, Pink)
		c.Set(3, 3, White)
	case "cr":
		c.Poly([][2]float64{{5.5, 0.8}, {9.6, 3.2}, {9.6, 7.8}, {5.5, 10.2}, {1.4, 7.8}, {1.4, 3.2}}, Amber)
		c.Poly([][2]float64{{5.5, 0.8}, {1.4, 3.2}, {1.4, 7.8}, {5.5, 5.5}}, Yellow)
		c.Poly([][2]float64{{5.5, 10.2}, {9.6, 7.8}, {5.5, 5.5}, {1.4, 7.8}}, Ochre)
		c.Rect(5, 3, 1, 5, Brown)
	case "wave":
		c.Ball(5.5, 6, 4.3, 3.8, 0, AlienRamp, 0.1, 0)
		c.Set(3, 6, Yellow)
		c.Set(7, 6, Yellow)
		c.Set(2, 1, Lime)
		c.Set(3, 2, Lime)
		c.Set(8, 1, Lime)
		c.Set(7, 2, Lime)
	case "time":
		c.Disc(5.5, 5.5, 4.6, 4.6, Silver, 1)
		c.Disc(5.5, 5.5, 3.6, 3.6, Navy, 1)
		c.Rect(5, 2, 1, 4, White)
		c.Rect(5, 5, 3, 1, White)
	case "star", "star0":
		body, hi := Amber, Yellow
		if name == "star0" {
			body, hi = Slate, Steel
		}
		pts := make([][2]float64, 10)
		for i := range pts {
			r := 5.2
			if i%2 == 1 {
				r = 2.3
			}
			a := -math.Pi/2 + float64(i)*math.Pi/5
			pts[i] = [2]float64{5.5 + r*math.Cos(a), 5.9 + r*math.Sin(a)}
		}
		c.Poly(pts, body)
		c.Poly([][2]float64{pts[9], pts[0], pts[1], {5.5, 5.9}}, hi)
		if name == "star" {
			c.Set(5, 3, White)
		}
	case "lock":
		c.Rect(3, 1, 5, 1, Gray)
		c.Rect(2, 2, 1, 4, Gray)
		c.Rect(8, 2, 1, 4, Gray)
		c.Rect(1, 5, 9, 5, Steel)
		c.Rect(1, 5, 9, 1, Silver)
		c.Rect(5, 7, 1, 2, Ink)
	case "info":
		c.Disc(5.5, 5.5, 5, 5, Blue, 1)
		c.Disc(5.5, 5.2, 3.8, 3.8, Hex(0x1aaff0), 1)
		c.Rect(5, 2, 1, 1, White)
		c.Rect(4, 4, 2, 1, White)
		c.Rect(5, 5, 1, 3, White)
		c.Rect(4, 8, 3, 1, White)
	case "target":
		c.Disc(5.5, 5.5, 4.8, 4.8, Red, 1)
		c.Disc(5.5, 5.5, 3.6, 3.6, Ink, 1)
		c.Disc(5.5, 5.5, 2.2, 2.2, Red, 1)
		c.Disc(5.5, 5.5, 1.1, 1.1, White, 1)
	case "entrances":
		// Two roads, A and B, merging into one towards the base.
		c.Rod(1.2, 1.8, 5, 5.5, 0.95, Ramp{Ochre, Amber}, 0)
		c.Rod(1.2, 9.2, 5, 5.5, 0.95, Ramp{Purple, Lilac}, 0)
		c.Rod(5, 5.5, 7, 5.5, 0.95, Ramp{Gray, Silver}, 0)
		c.Poly([][2]float64{{6.6, 2.4}, {10.4, 5.5}, {6.6, 8.6}}, White)
	case "crystal":
		// A pink crystal: the currency of the colonies.
		c.Poly([][2]float64{{1.2, 6.8}, {2.6, 5.2}, {3.8, 7}, {3.6, 9.9}, {1.4, 9.7}}, Purple)
		c.Poly([][2]float64{{1.2, 6.8}, {2.6, 5.2}, {2.6, 9.8}, {1.4, 9.7}}, Magenta)
		top, ul, ur := [2]float64{6.2, 0.6}, [2]float64{3.5, 3.2}, [2]float64{8.9, 3.2}
		ll, lr, bot := [2]float64{3.5, 7.8}, [2]float64{8.9, 7.8}, [2]float64{6.2, 10.3}
		c.Poly([][2]float64{top, ur, lr, bot}, Magenta)
		c.Poly([][2]float64{top, ul, ll, bot}, Pink)
		c.Poly([][2]float64{top, ul, {6.2, 4.4}}, Peach)
		c.Set(5, 2, White)
		c.Set(4, 4, White)
	case "up":
		// An arrow: the next stage of a colony building can be bought.
		c.Poly([][2]float64{{5.5, 0.8}, {10.2, 5.6}, {0.8, 5.6}}, Lime)
		c.Rect(4, 5, 3, 5, Lime)
		c.Rect(4, 5, 1, 5, Hex(0x8fe07a))
		c.Set(5, 2, White)
	case "trophy":
		// A gold cup: the trophies on the bridge.
		for y, r := range [][2]int{{2, 8}, {2, 8}, {3, 7}, {3, 7}, {4, 6}} {
			for x := r[0]; x <= r[1]; x++ {
				col := Amber
				switch {
				case y == 0 || x == r[0]:
					col = Yellow
				case x == r[1]:
					col = Ochre
				}
				c.Set(x, y+1, col)
			}
		}
		for _, p := range [][2]int{{1, 1}, {0, 2}, {0, 3}, {1, 4}} {
			c.Set(p[0], p[1], Amber)
			c.Set(10-p[0], p[1], Ochre)
		}
		c.Rect(5, 6, 1, 2, Ochre)
		c.Rect(3, 8, 5, 1, Amber)
		c.Rect(2, 9, 7, 1, Ochre)
		c.Set(3, 2, White)
	case "check":
		c.Poly([][2]float64{{0.8, 5.4}, {2.6, 3.8}, {4.4, 5.8}, {8.6, 1}, {10.3, 2.6}, {4.4, 9.4}}, Lime)
	case "super":
		// A cut gem, the mark of super upgrades.
		top, l, r, b := [2]float64{5.5, 0.6}, [2]float64{0.7, 5.5}, [2]float64{10.3, 5.5}, [2]float64{5.5, 10.4}
		mid := [2]float64{5.5, 5.5}
		c.Poly([][2]float64{top, r, b, l}, Violet)
		c.Poly([][2]float64{top, l, mid}, Lilac)
		c.Poly([][2]float64{b, r, mid}, Purple)
		c.Set(4, 3, White)
		c.Set(3, 4, White)
	}
	c.Outline(Ink)
	return c
}

// Missile is a small rocket seen from above, pointing along a (14x14).
func Missile(a float64) *Canvas {
	c := NewCanvas(14, 14)
	dx, dy := math.Cos(a), math.Sin(a)
	px, py := -dy, dx
	x0, y0 := 7-dx*4.2, 7-dy*4.2
	// Fins at the tail.
	c.Rod(x0-px*1.9, y0-py*1.9, x0+px*1.9, y0+py*1.9, 0.7, Ramp{Maroon, Red}, 0)
	c.Rod(x0, y0, 7+dx*2.6, 7+dy*2.6, 1.25, Ramp{Steel, Gray, Silver, White}, 0)
	c.Disc(7+dx*3.3, 7+dy*3.3, 1.2, 1.2, Red, 1)
	c.Outline(Ink)
	return c
}

// Flake is a tiny snowflake for frost effects (5x5).
func Flake() *Canvas {
	c := NewCanvas(5, 5)
	c.Rect(2, 0, 1, 5, White)
	c.Rect(0, 2, 5, 1, White)
	c.Set(1, 1, White)
	c.Set(3, 3, White)
	c.Set(3, 1, White)
	c.Set(1, 3, White)
	return c
}
