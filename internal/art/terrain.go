package art

import (
	"math"
	"math/rand"
	"sort"

	"outpost/internal/game"
)

var bayer4 = [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

func dither(x, y int) float64 { return (bayer4[(y&3)*4+(x&3)] + 0.5) / 16 }

// Muted ground tones keep the battlefield calm so units stay readable.
var (
	groundDark  = Hex(0x5b3333)
	groundLight = Hex(0x86473a)
	speckLight  = Hex(0x9a5642)
	groundRamp  = Ramp{Ink, Plum, groundDark, Brown, groundLight, speckLight}
	roadRamp    = Ramp{Plum, Brown, Clay, Copper, Tan, Sand}
)

// biome is the look of a world: the colours of the ground, the road and the
// front cliff, and the decorations scattered over the plateau.
type biome struct {
	ground, road Ramp
	rim          [2]Color // the top edge of the front cliff
	cliff        Ramp     // cliff bands (levels 1..4)
	rock         *Mat
	snow         *Mat // cap on the top of rocks (nil: bare rock)
	crystal      *Mat
	shrub, bulb  *Mat
}

var desertBiome = &biome{
	ground: groundRamp, road: roadRamp, rim: [2]Color{groundLight, Brown}, cliff: groundRamp,
	rock: mRock, crystal: mCrystal, shrub: mShrub, bulb: mBulb,
}

// The frozen moon of chapter 2: blue-grey snow, a trench of packed ice.
var iceBiome = &biome{
	ground:  Ramp{Ink, Hex(0x323852), Hex(0x48516d), Hex(0x55607e), Hex(0x616d8c), Hex(0x7682a0)},
	road:    Ramp{Hex(0x323852), Hex(0x4f5a78), Hex(0x75819e), Hex(0x8e9ab4), Hex(0xa9b4c9), Hex(0xc8d2e0)},
	rim:     [2]Color{Hex(0x8e9ab4), Hex(0x616d8c)},
	cliff:   Ramp{Ink, Navy, Hex(0x2f3c5e), Hex(0x3e5378), Hex(0x55709a)},
	rock:    mat(Ramp{Ink, Ink, Navy, Slate, Steel}, -0.05, 0.15),
	snow:    mat(Ramp{Steel, Gray, Silver, White}, 0.05, 0.2),
	crystal: mat(GlassRamp, 0.02, 0.8),
	shrub:   mat(Ramp{Ink, DeepTeal, Slate, Steel}, -0.05, 0.2),
	bulb:    mat(Ramp{Navy, DeepBlue, Blue, Cyan}, 0.05, 0.6),
}

func biomeOf(l *game.Level) *biome {
	switch l.Spec.Biome {
	case "ice":
		return iceBiome
	case "titan":
		return titanBiome
	}
	return desertBiome
}

const (
	// lakeLevel is the height of the frozen lakes (just below the ground).
	lakeLevel = -2.6
	// PathHalfWidth is the half width of the road floor in world pixels.
	PathHalfWidth = 12.5
	// RoadDepth is how deep the road trench is worn into the ground.
	RoadDepth = 4.0
	cliffMax  = 31.0
	topPad    = 16
)

// TerrainInfo describes the baked ground image and its height field.
type TerrainInfo struct {
	OY   int // canvas row of the projected line y = 0
	W, H int // world size in pixels
	hg   []float32
	lake []float32 // inside a frozen lake: distance to its shore (px), negative outside
	hole [][2]float64
}

// Ice colours of the frozen lakes: dark, clear ice over deep water.
var lakeRamp = Ramp{Ink, Hex(0x1a2546), Hex(0x223764), Hex(0x2d4c82), Hex(0x4772a6), Hex(0x9cc3e6), Hex(0xe2f2ff)}

// HoleR is the radius of an ice hole in world pixels.
const HoleR = 10.0

// lakeAt is the frozen lake value at a world pixel (see TerrainInfo.lake).
func (t *TerrainInfo) lakeAt(x, y float64) float64 {
	if t.lake == nil {
		return -99
	}
	ix, iy := max(0, min(int(x), t.W-1)), max(0, min(int(y), t.H-1))
	return float64(t.lake[iy*t.W+ix])
}

// holeAt is the distance from a world pixel to the nearest ice hole centre.
func (t *TerrainInfo) holeAt(x, y float64) float64 {
	best := math.Inf(1)
	for _, h := range t.hole {
		best = math.Min(best, math.Hypot(x-h[0], y-h[1]))
	}
	return best
}

// Height returns the ground height at a world point (bilinear).
func (t *TerrainInfo) Height(x, y float64) float64 {
	fx, fy := x-0.5, y-0.5
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	at := func(x, y int) float64 {
		x = max(0, min(x, t.W-1))
		y = max(0, min(y, t.H-1))
		return float64(t.hg[y*t.W+x])
	}
	a := at(x0, y0) + (at(x0+1, y0)-at(x0, y0))*tx
	b := at(x0, y0+1) + (at(x0+1, y0+1)-at(x0, y0+1))*tx
	return a + (b-a)*ty
}

func (t *TerrainInfo) normal(x, y float64) V3 {
	dx := (t.Height(x+1, y) - t.Height(x-1, y)) / 2
	dy := (t.Height(x, y+1) - t.Height(x, y-1)) / 2
	return V3{-dx, -dy, 1}.Norm()
}

type decor struct {
	y  float64
	sp *Sprite
}

// Terrain renders the static ground layer of a level: plateau, road trench,
// decorations, build pads, the alien burrow and the base shadow.
func Terrain(l *game.Level) (*Canvas, *TerrainInfo) {
	W, H := l.W*CellPx, l.H*CellPx
	ti := &TerrainInfo{OY: topPad, W: W, H: H, hg: make([]float32, W*H)}
	CW := W
	CH := topPad + int(math.Ceil(float64(H)*GY+cliffMax*HZ)) + 2
	c := NewCanvas(CW, CH)

	// Road: distance field around the smoothed path.
	dist := make([]float64, W*H)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	reach := PathHalfWidth + 6
	seg := func(a, b game.Vec) {
		ax, ay := a.X*CellPx, a.Y*CellPx
		bx, by := b.X*CellPx, b.Y*CellPx
		x0, x1 := int(math.Min(ax, bx)-reach), int(math.Max(ax, bx)+reach)
		y0, y1 := int(math.Min(ay, by)-reach), int(math.Max(ay, by)+reach)
		dx, dy := bx-ax, by-ay
		l2 := dx*dx + dy*dy
		for y := max(y0, 0); y <= min(y1, H-1); y++ {
			for x := max(x0, 0); x <= min(x1, W-1); x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				t := 0.0
				if l2 > 0 {
					t = math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l2))
				}
				d := math.Hypot(px-ax-dx*t, py-ay-dy*t)
				if d < dist[y*W+x] {
					dist[y*W+x] = d
				}
			}
		}
	}
	for _, rt := range l.Routes {
		pts := rt.Path.Pts
		for i := 1; i < len(pts); i++ {
			seg(pts[i-1], pts[i])
		}
	}
	// Where roads merge, their curves would leave a notch in the outer edge:
	// square those corners off.
	for _, rt := range l.Routes {
		if rt.Join >= len(rt.Cells) {
			continue
		}
		m := rt.Cells[rt.Join]
		for _, r2 := range l.Routes {
			for i, c := range r2.Cells {
				if c != m {
					continue
				}
				for _, j := range []int{i - 1, i + 1} {
					if j >= 0 && j < len(r2.Cells) {
						seg(c.Center(), r2.Cells[j].Center())
					}
				}
			}
		}
	}
	hw := PathHalfWidth
	bio := biomeOf(l)

	// Frozen lakes along the stretches of road between paired ice holes.
	if len(l.Holes) > 0 {
		ti.lake = make([]float32, W*H)
		for i := range ti.lake {
			ti.lake[i] = -99
		}
		for _, h := range l.Holes {
			c := h.Center()
			ti.hole = append(ti.hole, [2]float64{c.X * CellPx, c.Y * CellPx})
		}
		for _, rt := range l.Routes {
			for _, sp := range rt.Ice {
				for d := sp[0] - 0.1; d <= sp[1]+0.1; d += 0.08 {
					p, _ := rt.Path.At(d)
					px, py := p.X*CellPx, p.Y*CellPx
					r := hw + 5 + 5*noise(px/23, py/23, 61)
					// The shore swells round the holes at both ends.
					if e := math.Min(d-sp[0], sp[1]-d); e < 0.6 {
						r += 4 * (1 - e/0.6)
					}
					R := int(r) + 2
					for y := max(0, int(py)-R); y <= min(H-1, int(py)+R); y++ {
						for x := max(0, int(px)-R); x <= min(W-1, int(px)+R); x++ {
							v := float32(r - math.Hypot(float64(x)+0.5-px, float64(y)+0.5-py))
							if v > ti.lake[y*W+x] {
								ti.lake[y*W+x] = v
							}
						}
					}
				}
			}
		}
	}

	// Decorations away from the road, pads, base and spawn.
	r := rand.New(rand.NewSource(42))
	type spot struct{ x, y float64 }
	type crater struct{ x, y, r float64 }
	var placed []spot
	var craters []crater
	var objs []decor
	free := func(x, y, rad float64) bool {
		ix, iy := int(x), int(y)
		if ix < 6 || iy < 6 || ix >= W-6 || iy >= H-6 {
			return false
		}
		for oy := -1; oy <= 1; oy++ {
			for ox := -1; ox <= 1; ox++ {
				px, py := ix+ox*int(rad), iy+oy*int(rad)
				if px >= 0 && py >= 0 && px < W && py < H && dist[py*W+px] < hw+6 {
					return false
				}
			}
		}
		for _, s := range l.Slots {
			if math.Abs(x-(float64(s.X)+0.5)*CellPx) < 18+rad && math.Abs(y-(float64(s.Y)+0.5)*CellPx) < 18+rad {
				return false
			}
		}
		bc := l.BaseCenter()
		half := float64(l.BaseSize)*CellPx/2 + 6
		if math.Abs(x-bc.X*CellPx) < half+rad && math.Abs(y-bc.Y*CellPx) < half+rad {
			return false
		}
		for _, rt := range l.Routes {
			bx, by := BurrowPos(l, rt.Spawn)
			if math.Hypot(x-bx, y-by) < 34 {
				return false
			}
		}
		for _, p := range placed {
			if math.Hypot(p.x-x, p.y-y) < 16+rad {
				return false
			}
		}
		if ti.lakeAt(x, y) > -8-rad {
			return false
		}
		return true
	}
	var scenes []struct {
		y float64
		s *Scene
	}
	for i := 0; i < 420; i++ {
		x, y := r.Float64()*float64(W), r.Float64()*float64(H)
		kind := r.Float64()
		rad := 4.0
		if kind < 0.24 {
			rad = 7
		}
		if !free(x, y, rad) {
			continue
		}
		placed = append(placed, spot{x, y})
		var s *Scene
		switch {
		case kind < 0.18:
			craters = append(craters, crater{x, y, 4.5 + r.Float64()*3})
		case kind < 0.24:
			s = boulderScene(r, x, y, bio)
		case kind < 0.55:
			s = rockScene(r, x, y, bio)
		case kind < 0.72:
			s = crystalScene(r, x, y, bio)
		default:
			s = shrubScene(r, x, y, bio)
		}
		if s != nil {
			scenes = append(scenes, struct {
				y float64
				s *Scene
			}{y, s})
		}
	}
	for _, s := range l.Slots {
		cx, cy := (float64(s.X)+0.5)*CellPx, (float64(s.Y)+0.5)*CellPx
		scenes = append(scenes, struct {
			y float64
			s *Scene
		}{cy, PadScene(cx, cy)})
	}
	for _, rt := range l.Routes {
		bur := BurrowScene()
		bx, by := BurrowPos(l, rt.Spawn)
		for _, p := range bur.Parts {
			p.pos = p.pos.Add(V3{bx, by, 0})
			p.bc = p.bc.Add(V3{bx, by, 0})
			for _, q := range p.sub {
				q.pos = q.pos.Add(V3{bx, by, 0})
				q.bc = q.bc.Add(V3{bx, by, 0})
			}
		}
		scenes = append(scenes, struct {
			y float64
			s *Scene
		}{by, bur})
	}

	// Height field: the worn road trench (rising into the base gate) and craters.
	bx0 := float64(l.BaseMin.X * CellPx)
	bx1 := math.Inf(1) // the east wall, when a road comes in there
	if l.EastGate() {
		bx1 = float64((l.BaseMin.X + l.BaseSize) * CellPx)
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			h := 0.0
			if d := dist[y*W+x]; d < hw+3 {
				fade := smoothstep(0, 30, math.Max(bx0-fx, fx-bx1))
				h = -RoadDepth * fade * (1 - smoothstep(hw-1.2, hw+2.2, d))
			}
			for _, cr := range craters {
				d := math.Hypot(fx-cr.x, fy-cr.y)
				if d > cr.r+5 {
					continue
				}
				if d < cr.r {
					k := d / cr.r
					h -= 1.8 * (1 - k*k)
				}
				h += 1.1 * math.Exp(-sq((d-cr.r)/1.7))
			}
			if ti.lake != nil {
				// A lake fills the trench with a flat sheet of ice; its
				// holes sit in a ring of broken ice.
				if lv := ti.lakeAt(fx, fy); lv > -3 {
					h += (lakeLevel - h) * smoothstep(-3, 2.5, lv)
				}
				if dh := ti.holeAt(fx, fy); dh < HoleR+5 {
					h += 1.1*math.Exp(-sq((dh-HoleR-0.5)/1.6)) - 0.6*(1-smoothstep(HoleR-2, HoleR, dh))
				}
			}
			ti.hg[y*W+x] = float32(h)
		}
	}

	// Bake static objects (in parallel) and their shadows.
	objs = make([]decor, len(scenes))
	Parallel(len(scenes), func(i int) {
		objs[i] = decor{scenes[i].y, scenes[i].s.Bake(Opts{Inner: 2.5, Shadows: true}, true, 0, 0)}
	})
	shade := make([]bool, CW*CH)
	mark := func(sh *Canvas, x0, y0 int) { markShadow(shade, CW, CH, sh, x0, y0) }
	for _, o := range objs {
		mark(o.sp.Sh, -o.sp.SX, topPad-o.sp.SY)
	}
	base := BaseScene(l.EastGate())
	bc := l.BaseCenter()
	for _, p := range base.Parts {
		p.pos = p.pos.Add(V3{bc.X * CellPx, bc.Y * CellPx, 0})
		p.bc = p.bc.Add(V3{bc.X * CellPx, bc.Y * CellPx, 0})
		for _, q := range p.sub {
			q.pos = q.pos.Add(V3{bc.X * CellPx, bc.Y * CellPx, 0})
			q.bc = q.bc.Add(V3{bc.X * CellPx, bc.Y * CellPx, 0})
		}
	}
	x0, y0, x1, y1 := base.shadowBounds()
	bv := View{W: int(x1-x0) + 4, H: int(y1-y0) + 4, OX: -math.Floor(x0) + 2, OY: -math.Floor(y0) + 2}
	mark(base.ShadowMask(bv), -int(bv.OX), topPad-int(bv.OY))

	paintGround(c, ti, bio, dist, shade)
	stampDecor(c, objs)
	return c, ti
}

// paintGround fills the canvas with the ground seen from the camera: the
// plateau with its roads (dist: distance to the road, see groundColor),
// trench walls and the front cliff. shade marks the baked cast shadows.
func paintGround(c *Canvas, ti *TerrainInfo, bio *biome, dist []float64, shade []bool) {
	CW, CH := c.W, c.H
	const zmax, zmin = 1.6, -RoadDepth - 0.3
	fH := float64(ti.H)
	cliffD := func(x float64) float64 {
		return 19 + 8*noise(x/26, 0.5, 31) + 3*noise(x/7, 0.5, 32)
	}
	Parallel(CH, func(j int) {
		sy := float64(j) + 0.5 - topPad
		for i := 0; i < CW; i++ {
			x := float64(i) + 0.5
			yA := (sy + zmax*HZ) / GY
			yB := (sy + zmin*HZ) / GY
			k := j*CW + i
			if yA > fH {
				zf := (fH*GY - sy) / HZ
				if zf < ti.Height(x, fH-0.01) {
					if u := -zf; u <= cliffD(x) {
						c.Set(i, j, cliffColor(bio, i, j, x, u, cliffD(x)))
					}
					continue
				}
				yA = fH
			}
			if yA < 0 {
				continue
			}
			const step = 0.25
			prevF := (yA*GY-sy)/HZ - ti.Height(x, yA)
			hitY, hit := 0.0, false
			for y := yA - step; y >= math.Max(yB, 0)-step; y -= step {
				f := (y*GY-sy)/HZ - ti.Height(x, y)
				if f <= 0 {
					hitY = y + step*(-f)/(prevF-f)
					hit = true
					break
				}
				prevF = f
			}
			if !hit || hitY < 0 {
				continue
			}
			c.Set(i, j, groundColor(bio, ti, dist, i, j, x, hitY, shade[k]))
		}
	})
}

// stampDecor draws the baked decorations back to front.
func stampDecor(c *Canvas, objs []decor) {
	sort.SliceStable(objs, func(a, b int) bool { return objs[a].y < objs[b].y })
	for _, o := range objs {
		c.Stamp(o.sp.C, -o.sp.OX, topPad-o.sp.OY)
	}
}

// markShadow sets shade wherever the shadow mask sh (placed at x0, y0) is opaque.
func markShadow(shade []bool, CW, CH int, sh *Canvas, x0, y0 int) {
	for y := 0; y < sh.H; y++ {
		for x := 0; x < sh.W; x++ {
			if sh.Get(x, y).A > 0 {
				px, py := x0+x, y0+y
				if px >= 0 && py >= 0 && px < CW && py < CH {
					shade[py*CW+px] = true
				}
			}
		}
	}
}

func sq(x float64) float64 { return x * x }

func smoothstep(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

func groundColor(bio *biome, ti *TerrainInfo, dist []float64, i, j int, x, y float64, shaded bool) Color {
	W := ti.W
	ix, iy := max(0, min(int(x), W-1)), max(0, min(int(y), ti.H-1))
	d := dist[iy*W+ix]
	h := ti.Height(x, y)
	n := ti.normal(x, y)
	// Cast shadows inside the trench and craters.
	if !shaded && h < -0.2 {
		p := V3{x, y, h}
		for s := 0.5; s < 9; s += 0.5 {
			q := p.Add(Light.Mul(s))
			if q.Z > 1.6 {
				break
			}
			if ti.Height(q.X, q.Y) > q.Z+0.05 {
				shaded = true
				break
			}
		}
	}
	ramp, lv := bio.ground, 3
	road := d < PathHalfWidth-0.4
	if c, ok := lakeColor(ti, i, j, x, y, n, shaded); ok {
		return c
	}
	if road {
		ramp = bio.road
		nn := noise(x/11, y/11, 5) + (dither(i, j)-0.5)*0.12
		lv = 3
		if nn > 0.68 {
			lv = 4
		} else if nn < 0.28 {
			lv = 2
		}
		if d > PathHalfWidth-1.5 {
			lv = 2
		}
		if hv := hash(i, j, 7); hv < 0.012 {
			lv = 5
		} else if hv > 0.985 {
			lv = 2
		}
	} else {
		nn := noise(x/34, y/34, 1)*0.62 + noise(x/9, y/9, 2)*0.38
		nn += (dither(i, j) - 0.5) * 0.09
		switch {
		case nn < 0.30:
			lv = 2
		case nn > 0.72:
			lv = 4
		}
		if d < PathHalfWidth+3 {
			lv = 3 // the trench wall
		}
		if hv := hash(i, j, 3); hv < 0.014 {
			lv = 1
		} else if hv > 0.99 {
			lv = 5
		}
	}
	lit := n.Dot(Light) - Light.Z
	lv += int(math.Floor(lit*4.5 + dither(i, j)*0.999))
	lv = max(0, min(lv, len(ramp)-1))
	if shaded {
		return Shadowed(ramp[lv])
	}
	return ramp[lv]
}

// lakeColor paints the frozen lakes: dark clear ice with white cracks and
// glints, a snowy shore and the ice holes.
func lakeColor(ti *TerrainInfo, i, j int, x, y float64, n V3, shaded bool) (Color, bool) {
	if ti.lake == nil {
		return Color{}, false
	}
	lk := ti.lakeAt(x, y)
	dh := ti.holeAt(x, y)
	if lk < -1.5 && dh > HoleR+3 {
		return Color{}, false
	}
	ramp, lv := lakeRamp, 3
	switch {
	case dh < HoleR-1.5:
		// The hole: black water under a skin of new ice.
		lv = 1
		if hash(i, j, 71) < 0.05 {
			lv = 2
		}
		if dh > HoleR-3.5 {
			lv = 2
		}
	case dh < HoleR+3:
		// A ring of broken ice round the hole.
		lv = 5
		if hash(int(x/2.5), int(y/2.5), 72) < 0.35 {
			lv = 6
		}
		if hash(i, j, 73) < 0.12 {
			lv = 4
		}
	case lk < 1.2:
		// A rim of snow along the shore.
		ramp, lv = iceBiome.ground, 4
		if lk < 0 {
			lv = 5
		}
	default:
		lv = 2
		nn := noise(x/13, y/13, 74)*0.7 + noise(x/5, y/5, 75)*0.3
		if nn > 0.62 {
			lv = 3
		}
		// Thin white cracks and a few glints in the clear ice.
		c1 := math.Abs(noise(x/17, y/17, 76) - 0.5)
		c2 := math.Abs(noise(x/9+40, y/9, 77) - 0.5)
		switch {
		case c1 < 0.018:
			lv = 5
		case c2 < 0.014:
			lv = 4
		}
		if math.Mod(x*0.7+y+noise(x/30, y/30, 78)*14, 23) < 1.2 && hash(i, j, 79) < 0.6 {
			lv = 5
		}
		if lk < 3 {
			lv = 4 // the ice thins out to white at the shore
		}
	}
	lit := n.Dot(Light) - Light.Z
	lv += int(math.Floor(lit*3.5 + dither(i, j)*0.999))
	lv = max(0, min(lv, len(ramp)-1))
	if shaded {
		return Shadowed(ramp[lv]), true
	}
	return ramp[lv], true
}

// ShadowAlpha is the strength of every cast shadow, baked or drawn live.
const ShadowAlpha = 0.42

// Shadowed darkens a colour the same way the live shadow layer does.
func Shadowed(c Color) Color {
	k := 1 - ShadowAlpha
	return Color{R: uint8(float64(c.R)*k + 0.5), G: uint8(float64(c.G)*k + 0.5), B: uint8(float64(c.B)*k + 0.5), A: c.A}
}

func cliffColor(bio *biome, i, j int, x, u, depth float64) Color {
	switch {
	case u < 1.2:
		return bio.rim[0]
	case u < 2.4:
		return bio.rim[1]
	case u > depth-1.1:
		return Ink
	}
	band := (u + 2.4*noise(x/11, 0.5, 21)) / 4.4
	lv := []int{2, 3, 2, 1}[int(math.Floor(band))%4]
	if u > depth-4.5 && lv > 1 {
		lv--
	}
	if math.Mod(band, 1) < 0.2 && dither(i, j) < 0.5 {
		lv = max(1, lv-1)
	}
	if hash(int(x/3), 0, 23) < 0.1 && hash(i, j, 24) < 0.75 {
		lv = max(1, lv-1)
	}
	if hash(i, j, 25) < 0.03 {
		lv = min(4, lv+1)
	}
	return bio.cliff[lv]
}

// BurrowPos is where the burrow mound of an entrance stands (world pixels):
// at the spawn cell, pulled inside the map edge.
func BurrowPos(l *game.Level, sp game.Cell) (float64, float64) {
	c := sp.Center()
	W, H := float64(l.W*CellPx), float64(l.H*CellPx)
	return math.Max(24, math.Min(c.X*CellPx, W-24)), math.Max(24, math.Min(c.Y*CellPx, H-24))
}
