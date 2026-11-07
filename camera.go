package main

import "math"

// Camera maps world pixels to screen pixels with smooth, cursor-anchored zoom.
type Camera struct {
	X, Y   float64 // world point shown at the view anchor
	Z, TZ  float64 // current and target zoom (screen px per world px)
	MapW   float64
	MapH   float64
	Fit    float64
	MinZ   float64
	MaxZ   float64
	Auto   bool // keep the fitted view when the window is resized
	homing bool

	ax, ay   float64 // view anchor on screen
	vw, vh   float64 // view size
	margin   float64
	zx, zy   float64 // zoom anchor on screen
	vx, vy   float64 // keyboard pan velocity in world px/s
	nx, ny   float64 // screen px of a smooth pan still to go
	inputVX  float64
	inputVY  float64
	keySpeed float64
}

func NewCamera(mapW, mapH float64) *Camera {
	return &Camera{MapW: mapW, MapH: mapH, Auto: true, X: mapW / 2, Y: mapH / 2}
}

// SetView updates the visible screen area; top is the height of the HUD bar.
func (c *Camera) SetView(sw, sh, top, margin float64) {
	c.SetViewRect(0, top, sw, sh-top, margin)
	c.keySpeed = math.Max(sw, sh) * 0.75
}

// SetViewRect shows the map in a w×h screen rectangle at (x, y).
func (c *Camera) SetViewRect(x, y, w, h, margin float64) {
	c.vw, c.vh = w, h
	c.ax, c.ay = x+w/2, y+h/2
	c.margin = margin
	c.keySpeed = math.Max(w, h) * 0.75
	fit := math.Min((c.vw-2*margin)/c.MapW, (c.vh-2*margin)/c.MapH)
	if fit <= 0.05 {
		fit = 0.05
	}
	c.Fit = fit
	c.MinZ = fit
	c.MaxZ = fit * 2.6
	if c.Z == 0 || c.Auto {
		c.Z, c.TZ = fit, fit
		c.X, c.Y = c.MapW/2, c.MapH/2
	}
	c.TZ = clampF(c.TZ, c.MinZ, c.MaxZ)
	c.Z = clampF(c.Z, c.MinZ*0.999, c.MaxZ*1.001)
	c.clamp()
}

func (c *Camera) ToScreen(wx, wy float64) (float64, float64) {
	return (wx-c.X)*c.Z + c.ax, (wy-c.Y)*c.Z + c.ay
}

func (c *Camera) ToWorld(sx, sy float64) (float64, float64) {
	return (sx-c.ax)/c.Z + c.X, (sy-c.ay)/c.Z + c.Y
}

// ZoomAt changes the target zoom by wheel steps, keeping the point under the cursor fixed.
func (c *Camera) ZoomAt(sx, sy, steps float64) {
	c.TZ = clampF(c.TZ*math.Pow(1.2, steps), c.MinZ, c.MaxZ)
	c.zx, c.zy = sx, sy
	c.homing = false
	c.nx, c.ny = 0, 0
	c.Auto = c.TZ <= c.Fit*1.0001
}

// Nudge smoothly moves the view content by a screen-space delta (when zoomed in).
func (c *Camera) Nudge(dx, dy float64) {
	if c.Z <= c.Fit*1.001 {
		return
	}
	c.nx, c.ny = dx, dy
	c.homing = false
}

// Pan moves the view by a screen-space delta (dragging).
func (c *Camera) Pan(dx, dy float64) {
	c.X -= dx / c.Z
	c.Y -= dy / c.Z
	c.homing = false
	c.Auto = false
	c.nx, c.ny = 0, 0
	c.clamp()
}

// Home animates back to the fitted overview.
func (c *Camera) Home() {
	c.TZ = c.Fit
	c.zx, c.zy = c.ax, c.ay
	c.homing = true
	c.Auto = true
	c.nx, c.ny = 0, 0
}

// SetKeyInput sets the keyboard pan direction (-1..1 per axis).
func (c *Camera) SetKeyInput(x, y float64) { c.inputVX, c.inputVY = x, y }

func (c *Camera) Update(dt float64) {
	k := 1 - math.Exp(-dt*14)
	if c.Z != c.TZ {
		wx, wy := c.ToWorld(c.zx, c.zy)
		lz := math.Log(c.Z) + (math.Log(c.TZ)-math.Log(c.Z))*k
		c.Z = math.Exp(lz)
		if math.Abs(c.Z/c.TZ-1) < 0.0015 {
			c.Z = c.TZ
		}
		c.X = wx - (c.zx-c.ax)/c.Z
		c.Y = wy - (c.zy-c.ay)/c.Z
	}
	if c.homing {
		c.X += (c.MapW/2 - c.X) * k
		c.Y += (c.MapH/2 - c.Y) * k
		if c.Z == c.TZ && math.Abs(c.X-c.MapW/2) < 0.05 && math.Abs(c.Y-c.MapH/2) < 0.05 {
			c.homing = false
		}
	}
	if c.nx != 0 || c.ny != 0 {
		sx, sy := c.nx*k, c.ny*k
		if math.Abs(c.nx) < 0.5 && math.Abs(c.ny) < 0.5 {
			sx, sy = c.nx, c.ny
		}
		c.X -= sx / c.Z
		c.Y -= sy / c.Z
		c.nx -= sx
		c.ny -= sy
		c.Auto = false
	}
	tvx, tvy := c.inputVX*c.keySpeed/c.Z, c.inputVY*c.keySpeed/c.Z
	kv := 1 - math.Exp(-dt*12)
	c.vx += (tvx - c.vx) * kv
	c.vy += (tvy - c.vy) * kv
	if math.Abs(c.vx) < 0.01 && tvx == 0 {
		c.vx = 0
	}
	if math.Abs(c.vy) < 0.01 && tvy == 0 {
		c.vy = 0
	}
	if c.vx != 0 || c.vy != 0 {
		c.X += c.vx * dt
		c.Y += c.vy * dt
		if tvx != 0 || tvy != 0 {
			c.Auto = false
			c.homing = false
		}
	}
	c.clamp()
}

// clamp keeps the map on screen; a map smaller than the view is centred.
func (c *Camera) clamp() {
	pad := c.margin / c.Z
	hw, hh := c.vw/2/c.Z, c.vh/2/c.Z
	if lo, hi := hw-pad, c.MapW-hw+pad; lo >= hi {
		c.X = c.MapW / 2
	} else {
		c.X = clampF(c.X, lo, hi)
	}
	if lo, hi := hh-pad, c.MapH-hh+pad; lo >= hi {
		c.Y = c.MapH / 2
	} else {
		c.Y = clampF(c.Y, lo, hi)
	}
}

func clampF(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
