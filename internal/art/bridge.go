package art

import (
	"math"
	"math/rand"
	"sort"
	"sync"
)

func newRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// ---- The bridge of the Vanguard: trophies, specimen jars, the holo-table ----

// The bridge is a deck BridgeW × BridgeD world pixels with its back wall
// standing along y = 0.
const (
	BridgeW     = 22 * CellPx
	BridgeD     = 12 * CellPx
	BridgeWallH = 112.0
	BridgeOY    = 100 // canvas row of the back edge of the deck
	deckFront   = 14.0

	TableR    = 50.0 // the holo-table
	TableH    = 9.0
	PedestalR = 11.0
	PedestalH = 15.0

	// The specimen cabinet in the wall: two shelves of four jars.
	cabX0, cabX1 = 22.0, 180.0 // opening (world x)
	cabZ0, cabZ1 = 10.0, 97.0  // opening (height on the wall)
	cabDepth     = 6.0         // how deep the cabinet goes into the wall
	shelfLip     = 3.0
	JarY         = -3.0 // jars stand in the middle of the shelves
	JarRX        = 10.5 // jar radius
	JarBody      = 23.0 // glass height
	JarCap       = 3.0
	JarBase      = 2.5

	// The window and the status display.
	winX0, winX1 = 208.0, 496.0
	winZ0, winZ1 = 24.0, 94.0
	winFrame     = 6.0
	dispX0       = 526.0
	dispX1       = 684.0
	dispZ0       = 28.0
	dispZ1       = 92.0
)

// shelfZ are the tops of the two shelves, the upper one first.
var shelfZ = [2]float64{56, cabZ0}

// BridgeTable is the centre of the holo-table on the deck.
var BridgeTable = V3{11 * CellPx, 6.6 * CellPx, 0}

// PedestalPos is the floor centre of trophy pedestal i (0..11): pairs mirror
// each other left and right of the holo-table, from the back to the front.
func PedestalPos(i int) V3 {
	row := float64(i / 2)
	a := (235 - 22*row) * math.Pi / 180
	x := BridgeTable.X + 7.6*CellPx*math.Cos(a)
	y := BridgeTable.Y + 4.1*CellPx*math.Sin(a)
	if i%2 == 1 {
		x = BridgeW - x
	}
	return V3{x, y, 0}
}

// JarPos is the bottom centre of specimen jar i (0..7) on its shelf.
func JarPos(i int) V3 {
	slot := (cabX1 - cabX0) / 4
	return V3{cabX0 + slot*(float64(i%4)+0.5), JarY, shelfZ[i/4]}
}

// dispTop is the canvas row of the top of the status display.
var dispTop = BridgeOY - int(math.Round(dispZ1*HZ))

// DisplayRect is the screen of the status display in the room image.
func DisplayRect() (x, y, w, h int) {
	return int(dispX0), dispTop, int(dispX1 - dispX0), int(math.Round((dispZ1 - dispZ0) * HZ))
}

// WindowStars are stars that twinkle in the window: their pixel in the room
// image and how fast they twinkle.
var WindowStars = func() [][3]float64 {
	r := newRand(17)
	var out [][3]float64
	for len(out) < 18 {
		x := winX0 + 4 + r.Float64()*(winX1-winX0-8)
		z := winZ0 + 4 + r.Float64()*(winZ1-winZ0-8)
		y := float64(BridgeOY) - z*HZ
		if math.Hypot(x-marsC[0], y-marsC[1]) < marsR+4 || math.Abs(x-304) < 5 || math.Abs(x-400) < 5 {
			continue
		}
		out = append(out, [3]float64{math.Floor(x), math.Floor(y), 0.8 + r.Float64()*2})
	}
	return out
}()

// DisplaySlot is lamp i of the status display (12 trophies, then 8
// specimens): its rectangle in the room image.
func DisplaySlot(i int) (x, y, w, h int) {
	x0 := int(dispX0) + 10
	if i < 12 {
		return x0 + (i%6)*23, dispTop + 13 + (i/6)*8, 18, 6
	}
	return x0 + (i-12)*17 + 1, dispTop + 40, 14, 5
}

// CrewPos are where two officers stand on the bridge (world px) and the
// way they face.
var (
	CrewPos     = [2]V3{{60, 262, 0}, {644, 262, 0}}
	CrewHeading = [2]float64{-0.35, math.Pi + 0.35}
)

var (
	mSeat     = mat(Ramp{Ink, Plum, Maroon, Red, Pink}, -0.04, 0.3)
	mDeck     = mat(Ramp{Ink, Navy, Slate, Steel, Gray, Silver}, -0.02, 0.25)
	mDeckDark = mat(Ramp{Ink, Navy, Slate, Steel}, -0.04, 0.2)
	mScreen   = glow(Ramp{DeepBlue, Blue, Cyan, White}, -0.25)
	mScreenG  = glow(Ramp{Forest, Green, Lime, White}, -0.25)
	mScreenA  = glow(Ramp{Ochre, Amber, Yellow, White}, -0.25)
	mProj     = mat(Ramp{Ink, Ink, Navy, Navy, DeepBlue}, -0.1, 0.5)
	mGridG    = glow(Ramp{Navy, DeepBlue, Blue}, -0.2)
	mGoldT    = mat(Ramp{Plum, Brown, Ochre, Amber, Yellow, White}, -0.02, 0.7)
	mGoldTD   = mat(Ramp{Ink, Plum, Brown, Ochre, Amber}, -0.02, 0.5)
	mMarsG    = mat(Ramp{Ink, Plum, Maroon, Rust, Ochre, Tan}, 0, 0.4)
	mIceT     = mat(Ramp{Navy, DeepBlue, Blue, Cyan, White}, 0.04, 0.8)
	mBoneT    = mat(Ramp{Ink, Plum, Brown, Copper, Tan, Sand, White}, 0.02, 0.35)
	mFlag     = mat(Ramp{Ink, Maroon, Red, Pink}, 0, 0.2)
)

// ---- The deck furniture (baked into the room image) ----

// consoleScene is the helm console under the window.
func consoleScene() *Scene {
	s := &Scene{}
	cx := BridgeTable.X
	s.Add(Block(V3{cx, 11, 5.5}, V3{128, 10, 5.5}, Ident, 3, 0.6, mPanel))
	R := Roll(-0.42)
	top := xf{R, V3{cx, 11.5, 13}}
	s.Add(Block(top.T, V3{126, 11, 1.8}, R, 3, 0.8, mDeck))
	r := newRand(7)
	for _, px := range []float64{-96, -48, 0, 48, 96} {
		w := 15.0
		if px == 0 {
			w = 20
		}
		m := mScreen
		switch {
		case px == -96:
			m = mScreenG
		case px == 96:
			m = mScreenA
		}
		s.Add(Block(top.p(V3{px, -2.5, 1.9}), V3{w, 6, 0.3}, R, 0, 0, m).WithPaint(screenPaint(r.Int63())))
		// Rows of buttons under every screen.
		for k := 0; k < 5; k++ {
			bm := []*Mat{mCyanG, mYellowG, mRedG, mScreenG, mPanel}[r.Intn(5)]
			s.Add(Block(top.p(V3{px - 10 + float64(k)*5, 7, 1.9}), V3{1.4, 1.2, 0.5}, R, 0, 0, bm))
		}
	}
	return s
}

var mScreenDk = glow(Ramp{Ink, Navy, DeepBlue}, 0)

// screenPaint draws scan lines and a few bars of data on a screen.
func screenPaint(seed int64) func(p, n V3) *Mat {
	r := newRand(seed)
	bars := make([]float64, 6)
	for i := range bars {
		bars[i] = r.Float64()
	}
	return func(p, n V3) *Mat {
		if n.Z < 0.5 {
			return nil
		}
		if math.Mod(p.Y+20, 2) < 0.7 {
			return mScreenDk
		}
		// Bars along the lower half.
		if p.Y > 0.5 {
			k := int((p.X + 30) / 4)
			if k >= 0 && k < len(bars) && (p.Y-0.5)/5 > bars[k] && math.Mod(p.X+30, 4) < 2.6 {
				return mScreenDk
			}
		}
		return nil
	}
}

// chairScene is a pilot seat facing the window.
func chairScene(x, y float64) *Scene {
	s := &Scene{}
	s.Add(Lathe(V3{x, y, 0}, 5.5, 4.5, 1.2, 16, Ident, mPanel))
	s.Add(Cylinder(V3{x, y, 1}, V3{x, y, 7}, 1.4, mPanel))
	s.Add(Block(V3{x, y, 8.4}, V3{6.5, 6.5, 1.5}, Ident, 2, 0.8, mSeat))
	s.Add(Block(V3{x, y + 6, 16}, V3{6.5, 1.7, 7.5}, Roll(0.15), 2, 0.8, mSeat))
	s.Add(Block(V3{x, y + 7.2, 25}, V3{4, 1.5, 2.2}, Roll(0.15), 1.2, 0.6, mSeat))
	for _, side := range []float64{-1, 1} {
		s.Add(Block(V3{x + side*7.3, y + 0.5, 12}, V3{1.1, 5.5, 0.9}, Ident, 0.4, 0.3, mPanel))
		s.Add(Cylinder(V3{x + side*7.3, y + 4, 9}, V3{x + side*7.3, y + 4, 11.5}, 0.7, mPanel))
	}
	return s
}

// tableScene is the holo-table in the middle of the bridge.
func tableScene() *Scene {
	c := BridgeTable
	s := &Scene{}
	s.Add(Lathe(c, 36, 32, 5, 48, Ident, mPanel))
	rim := Lathe(c.Add(V3{0, 0, 4}), TableR, TableR-1, TableH-4, 56, Ident, mHull)
	rim.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.7 && p.Z > 1.4 && p.Z < 2.6 {
			return mCyanG
		}
		return nil
	})
	s.Add(rim)
	top := Lathe(c.Add(V3{0, 0, TableH - 0.6}), TableR-6, TableR-7, 1, 56, Ident, mProj)
	top.WithPaint(func(p, n V3) *Mat {
		if n.Z < 0.9 {
			return nil
		}
		r := math.Hypot(p.X, p.Y)
		if math.Abs(r-12) < 0.7 || math.Abs(r-26) < 0.6 || (r > 12 && math.Mod(math.Atan2(p.Y, p.X)+math.Pi, math.Pi/6) < 0.04) {
			return mGridG
		}
		return nil
	})
	s.Add(top)
	s.Add(Lathe(c.Add(V3{0, 0, TableH - 0.6}), 6, 5, 1.6, 24, Ident, mCyanG))
	for k := 0; k < 8; k++ {
		a := (float64(k) + 0.5) / 8 * 2 * math.Pi
		p := c.Add(V3{math.Cos(a) * (TableR - 3), math.Sin(a) * (TableR - 3) * 1, TableH + 0.4})
		s.Add(Block(p, V3{2.2, 2.2, 0.9}, Yaw(a), 0.6, 0.4, mDeck))
		s.Add(Sphere(p.Add(V3{0, 0, 1}), 1.2, mCyanG))
	}
	return s
}

// pedestalScene is a trophy pedestal; its emitter is lit in the game.
func pedestalScene(c V3) *Scene {
	s := &Scene{}
	s.Add(Lathe(c, PedestalR+2, PedestalR+1, 2, 28, Ident, mPanel))
	body := Prism(c.Add(V3{0, 0, 2}), PedestalR, PedestalR-2.5, PedestalH-4, 8, Yaw(math.Pi/8), mDeck)
	s.Add(body)
	s.Add(Lathe(c.Add(V3{0, 0, PedestalH - 2}), PedestalR-1.5, PedestalR-1, 2, 28, Ident, mHull))
	s.Add(Lathe(c.Add(V3{0, 0, PedestalH - 0.3}), PedestalR-3.5, PedestalR-3.5, 0.5, 28, Ident, mProj))
	return s
}

// sideConsoleScene is a standing console at the side of the bridge.
func sideConsoleScene(x, y float64, flip bool) *Scene {
	s := &Scene{}
	a := 0.0
	if flip {
		a = math.Pi
	}
	R := Yaw(a)
	x0 := xf{R, V3{x, y, 0}}
	s.Add(Block(x0.p(V3{0, 0, 9}), V3{9, 11, 9}, R, 2, 0.5, mPanel))
	top := R.Mul(Pitch(-0.6))
	s.Add(Block(x0.p(V3{1.5, 0, 19.5}), V3{8, 11.5, 1.6}, top, 1.5, 0.5, mDeck))
	scr := xf{top, x0.p(V3{1.5, 0, 19.5})}
	s.Add(Block(scr.p(V3{0, -2, 1.7}), V3{5, 7, 0.3}, top, 0, 0, mScreen))
	s.Add(Block(scr.p(V3{0, 7.5, 1.7}), V3{4.5, 1.6, 0.4}, top, 0, 0, mScreenA))
	return s
}

// ---- The room image ----

// BridgeTerrain renders the bridge: the back wall with the window, the
// specimen cabinet and the status display, the deck and its furniture.
func BridgeTerrain() (*Canvas, *TerrainInfo) {
	W, H := BridgeW, BridgeD
	OY := BridgeOY
	ti := &TerrainInfo{OY: OY, W: W, H: H, hg: make([]float32, W*H)}
	CW := W
	CH := OY + int(math.Ceil(float64(H)*GY+deckFront*HZ)) + 2
	c := NewCanvas(CW, CH)

	var scenes []decorScene
	add := func(y float64, s *Scene) { scenes = append(scenes, decorScene{y, s}) }
	add(11, consoleScene())
	for _, x := range []float64{BridgeTable.X - 48, BridgeTable.X + 48} {
		add(52, chairScene(x, 52))
	}
	add(BridgeTable.Y, tableScene())
	for i := 0; i < 12; i++ {
		p := PedestalPos(i)
		add(p.Y, pedestalScene(p))
	}
	add(276, sideConsoleScene(24, 276, false))
	add(276, sideConsoleScene(BridgeW-24, 276, true))

	objs := make([]decor, len(scenes))
	Parallel(len(scenes), func(i int) {
		objs[i] = decor{scenes[i].y, scenes[i].s.Bake(Opts{Inner: 2.5, Shadows: true}, true, 0, 0)}
	})
	shade := make([]bool, CW*CH)
	for _, o := range objs {
		markShadow(shade, CW, CH, o.sp.Sh, -o.sp.SX, OY-o.sp.SY)
	}

	Parallel(CH, func(j int) {
		for i := 0; i < CW; i++ {
			switch {
			case j < OY:
				z := (float64(OY-j) - 0.5) / HZ
				if z <= BridgeWallH {
					c.Set(i, j, wallColor(float64(i)+0.5, z, i, j))
				}
			default:
				y := (float64(j-OY) + 0.5) / GY
				if y < float64(H) {
					c.Set(i, j, deckColor(float64(i)+0.5, y, i, j, shade[j*CW+i]))
				} else if u := (float64(j-OY) + 0.5 - float64(H)*GY) / HZ; u < deckFront {
					c.Set(i, j, deckFaceColor(float64(i)+0.5, u, i, j))
				}
			}
		}
	})
	paintDisplay(c)
	sort.SliceStable(objs, func(a, b int) bool { return objs[a].y < objs[b].y })
	for _, o := range objs {
		c.Stamp(o.sp.C, -o.sp.OX, OY-o.sp.OY)
	}
	return c, ti
}

type decorScene struct {
	y float64
	s *Scene
}

// pick dithers between two colours: t = 0 is a, t = 1 is b.
func pick(a, b Color, t float64, i, j int) Color {
	if t > dither(i, j) {
		return b
	}
	return a
}

func mix(a, b Color, t float64) Color {
	t = math.Max(0, math.Min(1, t))
	return Color{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t), A: 255}
}

// wallColor is the back wall at x and height z.
func wallColor(x, z float64, i, j int) Color {
	// The window onto space.
	if x >= winX0 && x < winX1 && z >= winZ0 && z < winZ1 {
		return windowColor(x, z, i, j)
	}
	if x >= winX0-winFrame && x < winX1+winFrame && z >= winZ0-winFrame && z < winZ1+winFrame {
		return frameColor(x, z, winX0, winX1, winZ0, winZ1, winFrame)
	}
	// The specimen cabinet.
	if x >= cabX0 && x < cabX1 && z >= cabZ0 && z < cabZ1 {
		return cabinetColor(x, z, i, j)
	}
	if x >= cabX0-5 && x < cabX1+5 && z >= cabZ0-5 && z < cabZ1+5 {
		return frameColor(x, z, cabX0, cabX1, cabZ0, cabZ1, 5)
	}
	// The status display (its content is painted afterwards).
	if x >= dispX0 && x < dispX1 && z >= dispZ0 && z < dispZ1 {
		return screenBack(x, z, i, j)
	}
	if x >= dispX0-5 && x < dispX1+5 && z >= dispZ0-5 && z < dispZ1+5 {
		return frameColor(x, z, dispX0, dispX1, dispZ0, dispZ1, 5)
	}
	// The ceiling beam with its lamps.
	if z > BridgeWallH-9 {
		u := BridgeWallH - z
		switch {
		case u < 1.2:
			return Ink
		case u > 7.6:
			return Navy
		}
		if m := math.Mod(x+24, 48); m > 22 && m < 26 && u > 3 && u < 6 {
			return Amber
		}
		if math.Mod(x, 16) < 1 {
			return Navy
		}
		return pick(Slate, Steel, 0.35-u*0.03, i, j)
	}
	// The skirting.
	if z < 6 {
		if z > 4.8 {
			return Steel
		}
		if math.Mod(x+8, 32) < 2 {
			return Ink
		}
		return Navy
	}
	// Plain panels with seams, a little darker towards the ceiling.
	px := math.Mod(x+6, 46)
	pz := math.Mod(z-6, 34)
	switch {
	case px < 1 || pz < 1:
		return Navy
	case px < 2 || pz > 33:
		return Steel
	}
	base := pick(Slate, Navy, smoothstep(40, 100, z)*0.55, i, j)
	if px > 41 && px < 43 && pz > 4 && pz < 9 {
		return Steel
	}
	return base
}

// frameColor is a bevelled metal frame around an opening.
func frameColor(x, z, x0, x1, z0, z1, w float64) Color {
	dl, dr := x0-x, x-x1
	db, dt := z0-z, z-z1
	d := math.Max(math.Max(dl, dr), math.Max(db, dt))
	switch {
	case d < 1.2:
		return Ink
	case d > w-1:
		return Navy
	}
	// Lit from the upper left: top and left faces bright.
	switch {
	case dt >= math.Max(dl, dr):
		return Gray
	case dl >= math.Max(db, dt):
		return Steel
	case db >= math.Max(dl, dr):
		return Slate
	}
	return Slate
}

// The planet seen through the window (canvas coordinates of its centre).
var (
	marsC = [2]float64{452, BridgeOY - 34}
	marsR = 52.0
	moonC = [2]float64{238, BridgeOY - 64}
)

// windowColor is space seen through the window: stars, Mars and a moon,
// two mullions and a faint glare on the glass.
func windowColor(x, z float64, i, j int) Color {
	// Mullions.
	for _, mx := range []float64{304, 400} {
		if d := x - mx; d > -2.5 && d < 2.5 {
			switch {
			case d < -1.5:
				return Ink
			case d < 0:
				return Steel
			case d < 1.5:
				return Slate
			}
			return Ink
		}
	}
	sx, sy := x, float64(j)+0.5
	var col Color
	if dx, dy := sx-marsC[0], sy-marsC[1]; dx*dx+dy*dy < marsR*marsR {
		col = marsColor(dx/marsR, dy/marsR, i, j)
	} else if mdx, mdy := (sx-moonC[0])/6.5, (sy-moonC[1])/5; mdx*mdx+mdy*mdy < 1 {
		l := -0.55*mdx - 0.45*mdy + 0.5*math.Sqrt(math.Max(0, 1-mdx*mdx-mdy*mdy))
		col = RockRamp.At(0.25 + 0.55*l + 0.15*noise(sx/3, sy/3, 9))
	} else {
		// Deep space with a faint nebula.
		n := noise(sx/60, sy/40, 3)*0.7 + noise(sx/17, sy/13, 4)*0.3
		col = Ink
		if n > 0.62 {
			col = pick(Ink, Purple, (n-0.62)*1.6, i, j)
		}
		if n > 0.5 && n <= 0.62 {
			col = pick(Ink, Navy, (n-0.5)*3, i, j)
		}
		// Atmosphere glow around Mars.
		if d := math.Hypot(sx-marsC[0], sy-marsC[1]) - marsR; d < 5 {
			col = pick(col, Maroon, (1-d/5)*0.6, i, j)
		}
		if h := hash(i, j, 21); h > 0.985 {
			switch {
			case h > 0.998:
				col = White
			case h > 0.993:
				col = Silver
			default:
				col = Steel
			}
		}
	}
	// Glare streaks across the glass.
	if g := math.Mod(x+(winZ1-z)*1.4, 120); g < 9 || (g > 13 && g < 15) {
		col = mix(col, Silver, 0.12)
	}
	// The glass edge is a little darker.
	if x < winX0+1.5 || x > winX1-1.5 || z > winZ1-1.5 || z < winZ0+1.5 {
		col = mix(col, Ink, 0.5)
	}
	return col
}

var marsRamp = Ramp{Ink, Plum, Maroon, Brown, Rust, Ochre, Clay, Tan, Peach}

// marsColor shades the planet at (u, v), its centre at 0 and limb at 1.
func marsColor(u, v float64, i, j int) Color {
	w := math.Sqrt(math.Max(0, 1-u*u-v*v))
	l := -0.62*u - 0.38*v + 0.68*w
	// Surface: dark lowlands and bright plains drifting over the sphere.
	su, sv := u/(0.4+w)*12, v/(0.4+w)*12
	n := noise(su+30, sv+11, 13)*0.65 + noise(su*2.7, sv*2.7, 14)*0.35
	t := 0.18 + 0.72*math.Max(0, l) + (n-0.5)*0.28
	col := marsRamp.At(t + (dither(i, j)-0.5)*0.08)
	if v < -0.8 && l > 0 {
		col = pick(col, Silver, (-v-0.8)*6, i, j)
	}
	if l <= 0.02 {
		col = pick(Ink, Plum, 0.4+l*4, i, j)
	}
	// A thin bright rim on the lit limb.
	if w < 0.22 && l > 0 {
		col = pick(col, Peach, 0.6, i, j)
	}
	return col
}

// cabinetColor is the inside of the specimen cabinet: green back light and
// two shelves.
func cabinetColor(x, z float64, i, j int) Color {
	// Looking into the cabinet the eye drops by this much at its back.
	drop := cabDepth * GY / HZ
	for k, sz := range shelfZ {
		if z >= sz && z < sz+drop {
			// The top of the shelf, its front edge brightest.
			d := (z - sz) / drop
			if d < 0.2 {
				return Silver
			}
			if k == 1 {
				return pick(Steel, Slate, d, i, j)
			}
			return pick(Gray, Steel, d, i, j)
		}
		if k == 0 && z < sz && z >= sz-shelfLip {
			if z < sz-shelfLip+1 {
				return Ink
			}
			// Name plates under the jars.
			slot := math.Mod(x-cabX0, (cabX1-cabX0)/4)
			if slot > 13 && slot < 27 && z > sz-shelfLip+1.2 && z < sz-0.6 {
				return Ochre
			}
			return Steel
		}
	}
	zb := z - drop
	// Back light behind every jar.
	slot := (cabX1 - cabX0) / 4
	sx := math.Mod(x-cabX0, slot) - slot/2
	k := 0
	if zb > shelfZ[0] {
		k = 0
	} else {
		k = 1
	}
	cz := shelfZ[k] + JarBody*0.55
	g := math.Exp(-(sx*sx)/(2*13*13) - (zb-cz)*(zb-cz)/(2*15*15))
	col := DeepTeal
	switch {
	case g > 0.55:
		col = Forest
	case g > 0.45:
		col = pick(DeepTeal, Forest, 0.5, i, j)
	case g < 0.12:
		col = pick(Ink, DeepTeal, 0.5, i, j)
	}
	// Horizontal slats.
	if math.Mod(zb, 6) < 1 {
		col = mix(col, Ink, 0.45)
	}
	// Vertical ribs of the back panel.
	if math.Mod(x-cabX0, slot) < 1 {
		col = Ink
	}
	// Shadow under the upper shelf.
	if k == 1 && zb > shelfZ[0]-shelfLip-6 {
		col = mix(col, Ink, 0.55)
	}
	if zb > cabZ1-drop-5 {
		col = mix(col, Ink, 0.5)
	}
	return col
}

// screenBack is the empty status display: a dark blue grid.
func screenBack(x, z float64, i, j int) Color {
	col := pick(Ink, Navy, 0.6, i, j)
	if math.Mod(x-dispX0, 10) < 1 || math.Mod(z-dispZ0, 8) < 1 {
		col = pick(Navy, DeepBlue, 0.25, i, j)
	}
	return col
}

// paintDisplay writes the labels and the empty lamps of the status display.
func paintDisplay(c *Canvas) {
	x0 := int(dispX0) + 10
	drawLabel(c, "TROPHIES", x0, dispTop+3, Cyan)
	drawLabel(c, "SPECIMENS", x0, dispTop+30, Lime)
	for i := 0; i < 20; i++ {
		x, y, w, h := DisplaySlot(i)
		col := DeepBlue
		if i >= 12 {
			col = DeepTeal
		}
		for py := y; py < y+h; py++ {
			for px := x; px < x+w; px++ {
				cc := col
				if px == x || py == y || px == x+w-1 || py == y+h-1 {
					cc = Navy
				}
				c.Set(px, py, cc)
			}
		}
	}
}

var glyphCache map[rune]Glyph

// drawLabel writes text into a canvas with the game font (top-left at x, y).
func drawLabel(c *Canvas, s string, x, y int, col Color) {
	if glyphCache == nil {
		glyphCache = Glyphs()
	}
	for _, r := range s {
		if r == ' ' {
			x += FontSpace + 1
			continue
		}
		g, ok := glyphCache[r]
		if !ok {
			continue
		}
		for j := 0; j < g.C.H; j++ {
			for i := 0; i < g.C.W; i++ {
				if g.C.Get(i, j).A > 0 {
					c.Set(x+i, y+j, col)
				}
			}
		}
		x += g.Advance
	}
}

// The ring of pedestals around the holo-table (world px radii).
const ringRX, ringRY = 7.6 * CellPx, 4.1 * CellPx

// deckColor is the floor: staggered riveted plates, a ring through the
// pedestals, walkways to the table and warning stripes at the front edge.
func deckColor(x, y float64, i, j int, shaded bool) Color {
	tx, ty := x-BridgeTable.X, y-BridgeTable.Y
	dt := math.Hypot(tx, ty)
	// Plates of 64 × 32, every other row shifted by half a plate.
	row := int(y / 32)
	ox := 0.0
	if row%2 == 1 {
		ox = 32
	}
	px, py := math.Mod(x+ox, 64), math.Mod(y, 32)
	col := Navy
	switch {
	case px < 1 || py < 1:
		col = Ink
	case px < 2 || py < 2:
		col = Slate
	default:
		v := hash(int((x+ox)/64), row, 41)
		col = pick(Navy, Slate, 0.1+v*0.22, i, j)
		for _, c := range [][2]float64{{4.5, 4.5}, {59.5, 4.5}, {4.5, 27.5}, {59.5, 27.5}} {
			if math.Abs(px-c[0]) < 1 && math.Abs(py-c[1]) < 1 {
				col = Steel
			}
		}
		// Now and then a vent grate.
		if v > 0.9 && px > 12 && px < 52 && py > 9 && py < 23 {
			col = Slate
			if int(py)%3 == 0 {
				col = Ink
			}
		}
	}
	// Walkways: from the front edge to the table and from the table to the helm.
	if math.Abs(tx) < 34 && (ty > TableR+12 || (ty < -TableR-12 && y > 66)) {
		ax := math.Abs(tx)
		switch {
		case ax > 32:
			col = Ink
		case ax > 29.5:
			col = Steel
			if math.Mod(y, 14) < 2 {
				col = Cyan
			}
		default:
			col = pick(Slate, Navy, 0.3, i, j)
			if math.Mod(y, 32) < 1 {
				col = Navy
			}
			// Chevrons pointing at the table.
			if ty > 0 {
				if m := math.Mod(y-ax*0.6, 32); m > 12 && m < 15 && ax < 18 {
					col = Steel
				}
			}
		}
	}
	// The ring through the pedestals.
	a := math.Atan2(ty, tx)
	rr := 1 / math.Hypot(math.Cos(a)/ringRX, math.Sin(a)/ringRY)
	if d := math.Abs(dt - rr); d < 13 {
		switch {
		case d < 9:
			col = pick(Slate, Navy, 0.45, i, j)
			if math.Mod(a+math.Pi, math.Pi/18) < 0.012 {
				col = Navy
			}
		case d < 10.5:
			col = DeepBlue
		case d < 12:
			col = Ink
		}
	}
	// The ring around the holo-table.
	switch {
	case dt < TableR+10:
		col = pick(Ink, Navy, 0.5, i, j)
		if math.Mod(a+math.Pi, math.Pi/8) < 0.03 {
			col = Ink
		}
	case dt < TableR+12:
		col = Blue
	case dt < TableR+13.5:
		col = Ink
	}
	// Warning stripes along the front edge.
	if e := BridgeD - y; e < 8 {
		col = Ink
		if e > 1.5 && e < 7 && math.Mod(x+y*0.75, 12) < 6 {
			col = Amber
		} else if e > 1.5 && e < 7 {
			col = Plum
		}
	}
	// Light: a cyan pool around the table, warm light from the window.
	if g := math.Exp(-dt * dt / (2 * 95 * 95)); g > 0.05 {
		col = mix(col, Cyan, g*0.16)
	}
	if x > winX0-20 && x < winX1+20 && y < 120 {
		col = mix(col, Rust, (1-y/120)*0.12)
	}
	// Darker towards the sides.
	if e := math.Min(x, BridgeW-x); e < 60 {
		col = mix(col, Ink, (1-e/60)*0.4)
	}
	if shaded {
		col = Shadowed(col)
	}
	return col
}

// deckFaceColor is the front face of the deck, u below its top edge.
func deckFaceColor(x, u float64, i, j int) Color {
	switch {
	case u < 1.2:
		return Steel
	case u > deckFront-1.5:
		return Ink
	}
	if math.Mod(x+20, 40) < 3 && u > 4 && u < 7 {
		return Cyan
	}
	if math.Mod(x, 80) < 1 {
		return Ink
	}
	return pick(Slate, Navy, u/deckFront, i, j)
}

// segDist is the distance from (x, y) to the segment a-b and where along it
// the closest point lies (0..1).
func segDist(x, y, ax, ay, bx, by float64) (float64, float64) {
	dx, dy := bx-ax, by-ay
	t := ((x-ax)*dx + (y-ay)*dy) / (dx*dx + dy*dy)
	tc := math.Max(0, math.Min(1, t))
	return math.Hypot(x-ax-dx*tc, y-ay-dy*tc), t
}

// ---- Trophy models ----

// TrophyScene is the model of a trophy (see game.Trophy.Model), centred on
// the origin and standing on z = 0.
func TrophyScene(model string) *Scene {
	var s *Scene
	switch model {
	case "queen":
		s = QueenScene(PortraitHeading, 0, 8)
	case "wyrm":
		s = WyrmPortraitScene()
	case "star-mars":
		s = starScene(mMarsG, false)
	case "star-europa":
		s = starScene(mIceT, true)
	case "kepler":
		s = BaseScene(true)
	case "shield":
		s = shieldScene()
	case "flag":
		s = flagScene()
	case "gem":
		s = gemScene()
	case "skull":
		s = skullScene()
	case "cup":
		s = cupScene()
	case "chevrons":
		s = chevronScene()
	case "city":
		s = habitatScene(3)
	case "ship":
		s = ShuttleScene(0)
	default:
		s = EnemyScene(model, PortraitHeading, 0, 8)
	}
	centre(s)
	return s
}

// centre moves a scene so that its footprint is centred on the origin.
func centre(s *Scene) {
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, p := range s.Parts {
		r := p.sh.bound() * 0.6
		x0, x1 = math.Min(x0, p.bc.X-r), math.Max(x1, p.bc.X+r)
		y0, y1 = math.Min(y0, p.bc.Y-r), math.Max(y1, p.bc.Y+r)
	}
	Shift(s, V3{-(x0 + x1) / 2, -(y0 + y1) / 2, 0})
}

// stand is the small round plinth most trophies stand on.
func stand(s *Scene, r float64) float64 {
	s.Add(Lathe(V3{}, r+1.5, r+0.5, 2, 24, Ident, mDark))
	s.Add(Lathe(V3{0, 0, 2}, r, r-1, 2.5, 24, Ident, mGoldTD))
	return 4.5
}

// starScene is a five-pointed star around a globe on a stand.
func starScene(globe *Mat, ice bool) *Scene {
	s := &Scene{}
	z := stand(s, 6)
	s.Add(Cylinder(V3{0, 0, z}, V3{0, 0, z + 6}, 1.2, mGoldT))
	c := V3{0, 0, z + 17}
	for k := 0; k < 5; k++ {
		a := float64(k)/5*2*math.Pi - math.Pi/2
		d := V3{math.Cos(a), 0, -math.Sin(a)}
		s.Add(Shard(c.Add(d.Mul(4)), 3, 3.5, 6, alongZ(d), mGoldT))
	}
	s.Add(Sphere(c, 5, globe))
	if ice {
		for k := 0; k < 4; k++ {
			a := float64(k)/4*2*math.Pi + 0.4
			s.Add(Shard(V3{math.Cos(a) * 6, math.Sin(a) * 6, 1}, 1.4, 2, 2.5, alongZ(V3{math.Cos(a) * 0.4, math.Sin(a) * 0.4, 1}), mIceT))
		}
	}
	return s
}

// shieldScene is a heater shield with a boss, standing upright.
func shieldScene() *Scene {
	s := &Scene{}
	z := stand(s, 7)
	body := Ellipsoid(V3{0, 0, z + 14}, V3{12, 3.2, 15}, Ident, mHull).Clip(V3{0, 0, 1}, 9)
	body.WithPaint(func(p, n V3) *Mat {
		if math.Abs(p.X) < 1.6 || math.Abs(p.Z-2) < 1.6 {
			return mCyanG
		}
		return nil
	})
	s.Add(body)
	s.Add(Block(V3{0, 0, z + 23.5}, V3{12.5, 3.6, 1.2}, Ident, 1, 0.4, mGoldT))
	s.Add(Sphere(V3{0, 2.6, z + 16}, 3.2, mGoldT))
	return s
}

// flagScene is a flag still standing on a heap of rubble.
func flagScene() *Scene {
	s := &Scene{}
	r := newRand(5)
	for k := 0; k < 9; k++ {
		a := float64(k) * 2.4
		d := 3 + r.Float64()*6
		s.Add(Block(V3{math.Cos(a) * d, math.Sin(a) * d, 1.5 + r.Float64()}, V3{2.5 + r.Float64()*2, 2 + r.Float64()*1.5, 1.8}, Yaw(r.Float64()*3), 0.8, 0.5, mStoneLo))
	}
	s.Add(Cylinder(V3{0, 0, 0}, V3{0, 0, 36}, 0.9, mHull))
	s.Add(Sphere(V3{0, 0, 36.5}, 1.4, mGoldT))
	for k := 0; k < 5; k++ {
		x := 1 + float64(k)*3.4
		w := math.Sin(float64(k)*1.1) * 1.8
		b := Block(V3{x, w, 30}, V3{1.75, 0.5, 5.5}, Yaw(math.Cos(float64(k)*1.1)*0.5), 0, 0, mFlag)
		if k == 2 {
			b = Block(V3{x, w, 30}, V3{1.75, 0.5, 5.5}, Yaw(math.Cos(float64(k)*1.1)*0.5), 0, 0, mWindow)
		}
		s.Add(b)
	}
	return s
}

// gemScene is a cut gem with the five supers orbiting it.
func gemScene() *Scene {
	s := &Scene{}
	z := stand(s, 6)
	c := V3{0, 0, z + 16}
	m := mat(Ramp{Ink, Purple, Violet, Lilac, White}, 0.02, 0.9)
	s.Add(Prism(c, 8, 0, 9, 4, Ident, m))
	s.Add(Prism(c, 8, 0, 9, 4, Roll(math.Pi), m))
	orb := []*Mat{mCyanG, mat(Ramp{Navy, DeepBlue, Blue, Cyan, White}, 0, 0.6), mRedG, mYellowG, mat(Ramp{Purple, Magenta, Pink, Peach}, 0, 0.5)}
	for k := 0; k < 5; k++ {
		a := float64(k) / 5 * 2 * math.Pi
		s.Add(Sphere(c.Add(V3{math.Cos(a) * 13, math.Sin(a) * 13, math.Sin(a*2) * 3}), 2.2, orb[k]))
	}
	s.Add(Cylinder(V3{0, 0, z}, V3{0, 0, z + 7}, 1, mGoldT))
	return s
}

// skullScene is the skull of a hive alien on a stand.
func skullScene() *Scene {
	s := &Scene{}
	z := stand(s, 7)
	c := V3{0, 0, z + 11}
	cut := mat(Ramp{Ink, Ink, Plum}, 0, 0)
	head := Ellipsoid(c.Add(V3{-3, 0, 2}), V3{11, 8, 8}, Pitch(-0.25), mBoneT)
	head = head.Sub(cut,
		Ellipsoid(c.Add(V3{6.5, 3.8, 2.5}), V3{3, 2.6, 2.4}, Ident, cut),
		Ellipsoid(c.Add(V3{6.5, -3.8, 2.5}), V3{3, 2.6, 2.4}, Ident, cut))
	s.Add(head)
	// Crest and mandibles.
	for k := 0; k < 3; k++ {
		s.Add(Shard(c.Add(V3{-6 - float64(k)*3, 0, 7 - float64(k)}), 1.4, 2, 4, alongZ(V3{-0.6, 0, 1}), mBoneT))
	}
	for _, side := range []float64{-1, 1} {
		s.Add(Shard(c.Add(V3{6, side * 4, -4}), 1.5, 3, 5, alongZ(V3{0.8, side * 0.35, -0.5}), mBoneT))
	}
	s.Add(Sphere(c.Add(V3{6.2, 3.8, 2.5}), 1, mRedG))
	s.Add(Sphere(c.Add(V3{6.2, -3.8, 2.5}), 1, mRedG))
	return s
}

// cupScene is a gold trophy cup.
func cupScene() *Scene {
	s := &Scene{}
	z := stand(s, 8)
	s.Add(Lathe(V3{0, 0, z}, 6, 2, 3, 24, Ident, mGoldT))
	s.Add(Cylinder(V3{0, 0, z + 3}, V3{0, 0, z + 9}, 1.6, mGoldT))
	s.Add(Lathe(V3{0, 0, z + 9}, 2.5, 9, 9, 28, Ident, mGoldT))
	s.Add(Lathe(V3{0, 0, z + 18}, 9, 11, 7, 28, Ident, mGoldT))
	s.Add(Lathe(V3{0, 0, z + 24.6}, 9.6, 9.6, 0.6, 28, Ident, mGoldTD))
	for _, side := range []float64{-1, 1} {
		a, b := V3{side * 9.5, 0, z + 21}, V3{side * 14, 0, z + 19}
		c := V3{side * 9, 0, z + 12}
		s.Add(Rod(a, b, 1.1, mGoldT), Rod(b, c, 1.1, mGoldT))
	}
	return s
}

// chevronScene is the stacked chevrons of a commander's rank.
func chevronScene() *Scene {
	s := &Scene{}
	z := stand(s, 7)
	s.Add(Block(V3{0, 0, z + 2.5}, V3{2, 2, 2.5}, Ident, 0.5, 0, mDark))
	for k := 0; k < 3; k++ {
		h := z + 9 + float64(k)*7
		m := mGoldT
		if k == 2 {
			m = mCyanG
		}
		for _, side := range []float64{-1, 1} {
			R := Pitch(side * 0.62)
			s.Add(Block(V3{side * 5.2, 0, h}, V3{6.5, 2, 1.6}, R, 0, 0.4, m))
		}
	}
	return s
}

// ---- Holograms ----

var (
	fitMu   sync.Mutex
	fitSize = map[string][2]float64{}
)

// TrophyFit is the scale that fits a trophy model into a w × h box from
// every side as it turns.
func TrophyFit(model string, w, h float64) float64 {
	fitMu.Lock()
	sz, ok := fitSize[model]
	fitMu.Unlock()
	if !ok {
		sz = [2]float64{1, 1}
		for k := 0; k < 4; k++ {
			s := TrophyScene(model)
			Turn(s, float64(k)*math.Pi/4)
			p := s.BakeScaled(Opts{}, 1)
			sz = [2]float64{math.Max(sz[0], float64(p.C.W)), math.Max(sz[1], float64(p.C.H))}
		}
		fitMu.Lock()
		fitSize[model] = sz
		fitMu.Unlock()
	}
	return math.Min(w/sz[0], h/sz[1])
}

// TrophyModel renders a trophy at scale k, turned by a; the anchor is the
// centre of its base.
func TrophyModel(model string, k, a float64) *Sprite {
	s := TrophyScene(model)
	Turn(s, a)
	return s.BakeScaled(Opts{Inner: 2.5 / k, Shadows: true}, k)
}

// Holo turns a rendered model into a projection: its own colours washed
// into cyan by mix, every other row faint, the edges bright.
func Holo(src *Canvas, mixK float64) *Canvas {
	out := NewCanvas(src.W, src.H)
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			p := src.Get(x, y)
			if p.A == 0 {
				continue
			}
			l := (0.3*float64(p.R) + 0.59*float64(p.G) + 0.11*float64(p.B)) / 255
			cy := EnergyRamp.At(0.1 + l*1.1)
			col := mix(p, cy, mixK)
			a := 0.92
			if y%2 == 1 {
				a = 0.5
			}
			edge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if q := src.Get(x+d[0], y+d[1]); q.A == 0 {
					edge = true
				}
			}
			if edge {
				col, a = mix(col, White, 0.35), 1
			}
			out.Set(x, y, premul(col, a))
		}
	}
	return out
}

// Ghost is the dim outline of a trophy not won yet.
func Ghost(src *Canvas) *Canvas {
	out := NewCanvas(src.W, src.H)
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			if src.Get(x, y).A == 0 {
				continue
			}
			col, a := Slate, 0.7
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if q := src.Get(x+d[0], y+d[1]); q.A == 0 {
					col, a = Gray, 0.8
				}
			}
			if y%2 == 1 {
				a *= 0.6
			}
			out.Set(x, y, premul(col, a))
		}
	}
	return out
}

func premul(c Color, a float64) Color {
	return Color{R: uint8(float64(c.R) * a), G: uint8(float64(c.G) * a), B: uint8(float64(c.B) * a), A: uint8(255 * a)}
}

// ---- Specimen jars ----

// Jar is a specimen jar in three layers drawn over each other: the liquid
// behind the specimen, and the glass, the cap and the base in front.
type Jar struct {
	Back, Front *Canvas
	OX, OY      int // the bottom centre of the jar in both canvases
}

// JarLayers paints a jar; an empty one (not won yet) is dark.
func JarLayers(full bool) Jar {
	rx := JarRX
	ry := rx * GY
	rl := rx * 0.8 // the lid
	zg := JarBase + JarBody
	zt := zg + JarCap
	w := int(2*rx) + 4
	h := int(math.Ceil(zt*HZ+rl*GY+ry)) + 4
	cx := float64(w) / 2
	cy := float64(h) - ry - 2
	back, front := NewCanvas(w, h), NewCanvas(w, h)
	level := JarBase + JarBody*0.84
	liq := Ramp{DeepTeal, Forest, Green, Lime}
	top := Ramp{Green, Lime, Lime, Yellow}
	if !full {
		liq = Ramp{Ink, Ink, Ink, Navy}
		top = Ramp{Ink, Navy, Navy, DeepTeal}
	}
	// zAt is the height where the ray through a pixel meets the front (s = 1)
	// or the back (s = -1) of a cylinder of radius r.
	zAt := func(px, py, r, s float64) (float64, bool) {
		u := (px - cx) / r
		if math.Abs(u) >= 1 {
			return 0, false
		}
		return (cy - py + s*math.Sqrt(1-u*u)*r*GY) / HZ, true
	}
	inEll := func(px, py, z, r float64) bool {
		dx, dy := (px-cx)/r, (py-(cy-z*HZ))/(r*GY)
		return dx*dx+dy*dy <= 1
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			px, py := float64(i)+0.5, float64(j)+0.5
			u := (px - cx) / rx
			lit := 0.5 - 0.4*u
			// The lid: its top and its rim.
			if inEll(px, py, zt, rl) {
				col := MetalRamp.At(0.55 + 0.3*lit)
				if inEll(px, py, zt, rl*0.5) {
					col = Steel
				}
				front.Set(i, j, col)
				continue
			}
			if zf, ok := zAt(px, py, rl, 1); ok && zf >= zg && zf <= zt {
				front.Set(i, j, MetalRamp.At(0.15+0.55*lit))
				continue
			}
			zf, ok := zAt(px, py, rx, 1)
			if !ok {
				continue
			}
			zb, _ := zAt(px, py, rx, -1)
			// The base ring.
			if zf >= 0 && zf < JarBase {
				front.Set(i, j, DarkMetal.At(0.2+0.6*lit))
				continue
			}
			if inEll(px, py, JarBase, rx) && zf < JarBase {
				front.Set(i, j, DarkMetal.At(0.5))
				continue
			}
			// Inside the glass: what the ray meets behind the front pane.
			inside := zf >= JarBase && zb <= zg
			if zf > zg && !inEll(px, py, zg, rx) {
				continue
			}
			if !inside {
				continue
			}
			switch {
			case zf <= level:
				back.Set(i, j, liq.At(0.2+0.75*(1-math.Abs(u))-0.15*(1-zf/level)))
			case zb <= level:
				back.Set(i, j, top.At(0.3+0.6*(1-math.Abs(u))))
			default:
				back.Set(i, j, pick(Ink, Navy, 0.4, i, j))
			}
			// The glass pane: tint, highlights, edges and the rim at the top.
			col, a := Cyan, 0.1
			if full && zf <= level {
				col, a = Lime, 0.16 // the specimen is seen through the liquid
			}
			switch {
			case zf > zg:
				col, a = Silver, 0.45
			case math.Abs(u) > 0.86:
				col, a = Gray, 0.55
			case u > -0.66 && u < -0.44:
				col, a = White, 0.42
			case u > 0.52 && u < 0.62:
				col, a = White, 0.2
			}
			front.Set(i, j, premul(col, a))
		}
	}
	// One outline around the whole jar.
	solid := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < w && y < h && (back.Get(x, y).A > 0 || front.Get(x, y).A > 0)
	}
	var edge [][2]int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !solid(x, y) && (solid(x-1, y) || solid(x+1, y) || solid(x, y-1) || solid(x, y+1)) {
				edge = append(edge, [2]int{x, y})
			}
		}
	}
	for _, e := range edge {
		front.Set(e[0], e[1], Ink)
	}
	return Jar{Back: back, Front: front, OX: int(cx), OY: int(cy)}
}
