package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
	"outpost/internal/game"
)

// ---- Colonies: the crystals of the victories build up the freed worlds ----

const (
	colonistAngles = 8
	colonistFrames = 4
	buildTime      = 2.2  // seconds a colony stage takes to rise
	maxWalkers     = 12   // colonists on the streets at most
	walkStride     = 13.0 // world px per walk cycle
	laneY          = 0.16 // cells off the middle of the avenue
	landTime       = 3.2  // the shuttle's descent (seconds)
	leaveTime      = 3.0
)

// colonyTab is one tab of the colony screen.
type colonyTab struct {
	label string
	loc   *game.Location // nil: the bridge of the ship
	soon  bool
	tip   string
}

func colonyTabs() []colonyTab {
	tabs := []colonyTab{{label: "Vanguard · Bridge"}}
	for _, l := range game.Colonies {
		t := colonyTab{label: l.Name + " · " + l.Boss, loc: l, soon: l.Soon}
		if l.Soon {
			t.tip = fmt.Sprintf("Defeat the %s in %s to free %s. Its colony comes in a later version.", l.Boss, l.Level.ID, l.Name)
		}
		tabs = append(tabs, t)
	}
	return tabs
}

// buildAnim is a stage of a building rising on its lot.
type buildAnim struct {
	b     *game.Building
	stage int // the stage being built (1 = the first)
	t     float64
}

// stop is a place the colonists walk to.
type stop struct {
	b      *game.Building // nil: the command center or a spot on the avenue
	door   game.Vec       // cells
	on     game.Vec       // where its street meets the avenue (cells)
	inside bool           // colonists go in (out of sight) while they wait
}

// walker is a colonist going about its day.
type walker struct {
	path    []game.Vec // world px
	seg     int
	pos     game.Vec // world px
	heading float64
	dist    float64 // walked so far (animation)
	speed   float64
	wait    float64 // seconds left at a stop
	at      stop
	hidden  bool
	variant int
	lane    float64 // sideways offset on the side streets (cells)
}

type shipPhase int

const (
	shipAway shipPhase = iota
	shipLanding
	shipParked
	shipLeaving
)

// shuttle is the supply shuttle of the landing pad.
type shuttle struct {
	phase shipPhase
	t     float64 // seconds in the phase
	wait  float64 // how long it stays away or parked
	dustT float64
}

// colonyState is the colony screen.
type colonyState struct {
	tab     int
	sel     *game.Building // selected building (nil: the overview)
	hover   *game.Building // lot under the cursor (or its row in the panel)
	anims   []buildAnim
	walkers []walker
	rng     *rand.Rand
	rank    string // the rank the player has seen
	t       float64
	smokeT  float64
	weldT   float64
	ship    shuttle
	flash   float64 // "need more crystals" lights up
	spent   int     // crystals just spent (a short popup at the counter)
	spentT  float64
	opened  bool // the screen has been open before (keep its tab)
}

// colonyLoc is the location of the open tab.
func (a *App) colonyLoc() *game.Location {
	tabs := colonyTabs()
	if a.col.tab < 0 || a.col.tab >= len(tabs) || tabs[a.col.tab].loc == nil || tabs[a.col.tab].soon {
		return game.Mars
	}
	return tabs[a.col.tab].loc
}

// colonyLocked reports whether a location is not free yet (its boss stands).
func (a *App) colonyLocked(l *game.Location) bool {
	return !a.dbg.unlock && a.best(l.Level) == 0
}

// colonyStage is the number of stages built of a building.
func (a *App) colonyStage(b *game.Building) int { return min(a.save.Colony[b.ID], len(b.Stages)) }

// colonyDone is the number of stages of a building that stand (the one
// being built does not count yet).
func (a *App) colonyDone(b *game.Building) int {
	st := a.colonyStage(b)
	if a.colonyBusy(b) {
		st--
	}
	return st
}

// colonyBuilt counts the stages that stand in a location.
func (a *App) colonyBuilt(l *game.Location) int {
	n := 0
	for _, b := range l.Buildings {
		n += a.colonyDone(b)
	}
	return n
}

// colonyRank is the rank of a location.
func (a *App) colonyRank(l *game.Location) string {
	r, _ := game.Rank(a.colonyBuilt(l), l.Stages())
	return r
}

// colonists is the number of settlers living in a location.
func (a *App) colonists(l *game.Location) int {
	if h := l.Building(l.ID + ".habitat"); h != nil {
		return 3 * a.colonyDone(h)
	}
	return 0
}

// colonyAnim is the running construction of a building, or nil.
func (a *App) colonyAnim(b *game.Building) *buildAnim {
	for i := range a.col.anims {
		if a.col.anims[i].b == b {
			return &a.col.anims[i]
		}
	}
	return nil
}

func (a *App) colonyBusy(b *game.Building) bool { return a.colonyAnim(b) != nil }

// visitOnly buildings are looked at from outside instead of entered.
func visitOnly(b *game.Building) bool {
	return strings.HasSuffix(b.ID, ".monument") || strings.HasSuffix(b.ID, ".pad")
}

// openColony shows the colony screen.
func (a *App) openColony() {
	c := &a.col
	if c.rng == nil {
		c.rng = rand.New(rand.NewSource(1 + int64(a.now*1000)))
	}
	a.scr = scrColony
	a.popup = panelNone
	a.paused = false
	a.fadeIn = 1
	a.fx.Reset()
	c.sel, c.hover, c.anims = nil, nil, c.anims[:0]
	c.flash, c.spentT = 0, 0
	a.br.sel, a.br.hover = nil, nil
	// The first time: Mars once it is free, else the bridge. Later the
	// screen opens on the tab it was left on.
	if t := colonyTabs(); !c.opened || c.tab < 0 || c.tab >= len(t) || t[c.tab].soon {
		c.tab = 0
		if !a.colonyLocked(game.Mars) {
			c.tab = 1
		}
	}
	c.opened = true
	a.showColony()
}

// openColonyAt opens the colony screen on the tab of a location (nil: the
// bridge).
func (a *App) openColonyAt(l *game.Location) {
	for i, t := range colonyTabs() {
		if t.loc == l && !t.soon {
			a.col.tab, a.col.opened = i, true
		}
	}
	a.openColony()
}

// showColony frames the location of the open tab and fills its streets.
func (a *App) showColony() {
	c := &a.col
	if a.onBridge() {
		a.as.Terrain, a.as.TI = a.as.Bridge.ter.img, a.as.Bridge.ter.ti
		tb := a.as.Terrain.Bounds()
		a.cam = NewCamera(float64(tb.Dx()), float64(tb.Dy()))
		a.applyView()
		c.walkers = c.walkers[:0]
		a.br.showT = 0
		return
	}
	lk := a.as.Colonies[a.colonyLoc()]
	a.as.Terrain, a.as.TI = lk.ter.img, lk.ter.ti
	tb := a.as.Terrain.Bounds()
	a.cam = NewCamera(float64(tb.Dx()), float64(tb.Dy()))
	a.applyView()
	a.spawnWalkers()
	c.ship = shuttle{wait: 1 + c.rng.Float64()*3}
	c.t = 0
	c.rank = a.colonyRank(a.colonyLoc())
}

// closeColony goes back to the title screen.
func (a *App) closeColony() {
	if won := a.save.claimTrophies(); len(won) > 0 {
		a.trophyToast(won)
	}
	a.fx.Reset()
	a.scr = scrMenu // the demo camera must use the full-screen menu view
	a.startDemo()
	a.fadeIn = 1
}

func (a *App) colonyKeys(enter bool) {
	jp := inpututil.IsKeyJustPressed
	c := &a.col
	if a.onBridge() {
		if a.bridgeKeys() {
			return
		}
	} else if jp(ebiten.KeyEscape) {
		if c.sel != nil {
			c.sel = nil
		} else {
			a.closeColony()
		}
		return
	} else if enter || jp(ebiten.KeyU) || jp(ebiten.KeyB) {
		a.buyStage()
	}
	if jp(ebiten.KeyHome) {
		a.cam.Home()
	}
	if jp(ebiten.KeyTab) && !a.onBridge() {
		bs := a.colonyLoc().Buildings
		i := 0
		for j, b := range bs {
			if b == c.sel {
				i = j + 1
			}
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			i -= 2
		}
		c.sel = bs[(i%len(bs)+len(bs))%len(bs)]
	}
	pr := ebiten.IsKeyPressed
	kx, ky := 0.0, 0.0
	if pr(ebiten.KeyA) || pr(ebiten.KeyArrowLeft) {
		kx--
	}
	if pr(ebiten.KeyD) || pr(ebiten.KeyArrowRight) {
		kx++
	}
	if pr(ebiten.KeyW) || pr(ebiten.KeyArrowUp) {
		ky--
	}
	if pr(ebiten.KeyS) || pr(ebiten.KeyArrowDown) {
		ky++
	}
	if kx != 0 && ky != 0 {
		kx, ky = kx*math.Sqrt2/2, ky*math.Sqrt2/2
	}
	a.cam.SetKeyInput(kx, ky)
}

// buyStage spends crystals on the next stage of the selected building.
func (a *App) buyStage() {
	c := &a.col
	b := c.sel
	if b == nil || a.colonyLocked(a.colonyLoc()) || a.colonyBusy(b) {
		return
	}
	st := a.colonyStage(b)
	if st >= len(b.Stages) {
		return
	}
	cost := b.Stages[st].Cost
	if a.save.Crystals < cost {
		c.flash = 1
		return
	}
	a.save.Crystals -= cost
	a.save.Colony[b.ID] = st + 1
	a.save.store()
	c.anims = append(c.anims, buildAnim{b: b, stage: st + 1})
	c.spent, c.spentT = cost, 1
	P := float64(art.CellPx)
	cc := b.Center()
	a.fx.Build(cc.X*P, cc.Y*P, art.SlabH)
}

// ---- Layout ----

type colonyLayout struct {
	back   Button
	titleX int
	tabs   []Button
	view   Rect // the map
	panel  Rect
	rows   []Button // overview: one per building
	all    Button   // detail: back to the overview
	build  Button   // detail: the next stage
}

func (a *App) colonyLayout() colonyLayout {
	u := a.u
	m := 10 * u
	c := &a.col
	var cl colonyLayout
	cl.back = Button{ID: "back", Label: "Back", Key: "Esc", Enabled: true}
	cl.back.R = Rect{m, m, a.buttonWidth(cl.back), a.btnHeight()}
	cl.titleX = m + cl.back.R.W + 12*u
	ty := m + 14*u + 5*u + 11*u + 8*u
	x := m
	for i, t := range colonyTabs() {
		b := Button{ID: fmt.Sprintf("loc:%d", i), Label: t.label, Enabled: !t.soon, Primary: i == c.tab}
		if t.soon {
			b.Key = "Soon"
		}
		b.R = Rect{x, ty, a.buttonWidth(b), a.btnHeight()}
		x += b.R.W + 4*u
		cl.tabs = append(cl.tabs, b)
	}
	top := ty + a.btnHeight() + 7*u
	pw := int(clampF(float64(a.sw)*0.28, float64(170*u), float64(220*u)))
	cl.panel = Rect{a.sw - m - pw, top, pw, a.sh - m - top}
	cl.view = Rect{m, top, cl.panel.X - 8*u - m, a.sh - m - top}
	loc := a.colonyLoc()
	px, iw := cl.panel.X+8*u, pw-16*u
	if a.onBridge() {
		a.layoutBridge(&cl, px, iw)
		return cl
	}
	if c.sel == nil {
		y := cl.panel.Y + 8*u + a.colonyHeadH(loc, iw)
		for i := range loc.Buildings {
			cl.rows = append(cl.rows, Button{ID: fmt.Sprintf("csel:%d", i), R: Rect{px - 3*u, y, iw + 6*u, 15 * u}, Enabled: true})
			y += 16 * u
		}
		return cl
	}
	cl.all = Button{ID: "call", Label: "All buildings", Key: "Esc", Enabled: true}
	cl.all.R = Rect{px, cl.panel.Y + 8*u, a.buttonWidth(cl.all), a.btnHeight()}
	b := c.sel
	st := a.colonyStage(b)
	if st < len(b.Stages) {
		next := b.Stages[st]
		bt := Button{ID: "cbuild", Label: "Build", Key: "Enter", Primary: true, Cost: next.Cost, CostIcon: "crystal"}
		if st > 0 {
			bt.Label = "Upgrade"
		}
		switch {
		case a.colonyLocked(loc):
			bt.Label, bt.Key, bt.Cost = "Locked", "", 0
		case a.colonyBusy(b):
			bt.Label, bt.Key, bt.Cost = "Building...", "", 0
		default:
			bt.Enabled = a.save.Crystals >= next.Cost
		}
		bt.R = Rect{px, cl.panel.Y + cl.panel.H - a.btnHeight() - 22*u, iw, a.btnHeight()}
		cl.build = bt
	}
	return cl
}

// colonyHeadH is the height of the overview above the list of buildings.
func (a *App) colonyHeadH(l *game.Location, w int) int {
	u := a.u
	return 18*u + 17*u + 11*u + 10*u + 15*u + 17*u + 11*u*len(a.wrap(l.Desc, w, u)) + 6*u + 13*u
}

func (a *App) layoutColony() {
	cl := a.colonyLayout()
	a.addButton(cl.back)
	for _, b := range cl.tabs {
		a.addButton(b)
	}
	for _, b := range cl.rows {
		a.addButton(b)
	}
	if cl.all.ID != "" {
		a.addButton(cl.all)
	}
	if cl.build.ID != "" {
		a.addButton(cl.build)
	}
	a.blockers = append(a.blockers, cl.panel)
}

// colonyView is the screen rectangle of the map.
func (a *App) colonyView() Rect { return a.colonyLayout().view }

// pressColony handles the buttons of the colony screen.
func (a *App) pressColony(id string) bool {
	c := &a.col
	if a.onBridge() && a.pressBridge(id) {
		return true
	}
	switch id {
	case "back":
		a.closeColony()
	case "call":
		c.sel = nil
	case "cbuild":
		a.buyStage()
	default:
		if n, ok := cutIndex(id, "csel:"); ok {
			if bs := a.colonyLoc().Buildings; n < len(bs) {
				c.sel = bs[n]
			}
		} else if n, ok := cutIndex(id, "loc:"); ok {
			if t := colonyTabs(); n < len(t) && !t[n].soon && n != c.tab {
				// Purchases are already saved. Finish their visual animation before
				// changing maps; old building pointers belong to the old location.
				c.anims = c.anims[:0]
				a.fx.Reset()
				if won := a.save.claimTrophies(); len(won) > 0 {
					a.trophyToast(won)
				}
				c.tab = n
				c.sel = nil
				a.showColony()
			}
		} else {
			return false
		}
	}
	return true
}

// ---- The map ----

// lotRect is the screen-space box of a lot with the model standing on it
// (terrain image pixels, before the camera).
func (a *App) lotRect(b *game.Building) (x0, y0, x1, y1 float64) {
	P := float64(art.CellPx)
	oy := float64(a.as.TI.OY)
	x0, x1 = float64(b.Lot.X)*P, float64(b.Lot.X+b.W)*P
	y0 = float64(b.Lot.Y)*P*art.GY + oy
	y1 = float64(b.Lot.Y+b.H)*P*art.GY + oy
	lk := a.as.Colonies[a.colonyLoc()]
	st := max(1, a.colonyStage(b))
	sp := lk.stages[b][st-1]
	cy := b.Center().Y*P*art.GY + oy
	y0 = math.Min(y0, cy-sp.oy+4)
	return
}

// lotAt finds the lot under a screen point, the front one first.
func (a *App) lotAt(mx, my int) *game.Building {
	loc := a.colonyLoc()
	if a.onBridge() || a.colonyLocked(loc) || !a.colonyView().Has(mx, my) {
		return nil
	}
	wx, wy := a.cam.ToWorld(float64(mx), float64(my))
	var best *game.Building
	bestY := math.Inf(-1)
	for _, b := range loc.Buildings {
		x0, y0, x1, y1 := a.lotRect(b)
		if wx >= x0 && wx < x1 && wy >= y0 && wy < y1 && y1 > bestY {
			best, bestY = b, y1
		}
	}
	return best
}

// colonyClick selects the building under the cursor (or none).
func (a *App) colonyClick(mx, my int) {
	if !a.colonyView().Has(mx, my) {
		return
	}
	a.col.sel = a.lotAt(mx, my)
}

// ---- Colonists ----

// colonyWalkers is how many colonists walk the streets.
func (a *App) colonyWalkers() int {
	l := a.colonyLoc()
	if a.colonyLocked(l) {
		return 0
	}
	return min(maxWalkers, 2+a.colonists(l))
}

// colonyStops lists the places the colonists go to: the command center, the
// buildings and a few spots along the avenue.
func (a *App) colonyStops() []stop {
	l := a.colonyLoc()
	hq := game.Vec{X: l.HQCenter().X, Y: l.Avenue}
	out := []stop{{door: hq, on: hq, inside: true}}
	for _, b := range l.Buildings {
		if a.colonyDone(b) > 0 {
			out = append(out, stop{b: b, door: l.Door(b), on: l.Junction(b), inside: !visitOnly(b)})
		}
	}
	av := l.Streets()[0]
	for _, f := range []float64{0.2, 0.62, 0.85} {
		p := game.Vec{X: av[0].X + (av[1].X-av[0].X)*f, Y: l.Avenue}
		out = append(out, stop{door: p, on: p})
	}
	return out
}

// standPoint is where a colonist waits at a stop (world px).
func (a *App) standPoint(s stop, w *walker) game.Vec {
	P := float64(art.CellPx)
	c := &a.col
	p := s.door
	switch {
	case s.inside && s.door != s.on:
		p.X += w.lane
	case !s.inside:
		p.Y += (c.rng.Float64()*2 - 1) * 0.3
		if s.b != nil {
			// In front of the monument or the pad, a step away from it.
			d := 0.15 + c.rng.Float64()*0.35
			if s.b.Center().X > p.X {
				d = -d
			}
			p.X += d
		}
	}
	return game.Vec{X: p.X * P, Y: p.Y * P}
}

// walkTo sends a colonist from its stop to another one.
func (a *App) walkTo(w *walker, stops []stop) {
	c := &a.col
	next := stops[c.rng.Intn(len(stops))]
	for i := 0; i < 8 && next.door == w.at.door; i++ {
		next = stops[c.rng.Intn(len(stops))]
	}
	P := float64(art.CellPx)
	ly := laneY
	if next.on.X < w.at.on.X {
		ly = -laneY
	}
	pts := []game.Vec{w.pos}
	add := func(x, y float64) {
		p := game.Vec{X: x * P, Y: y * P}
		if l := pts[len(pts)-1]; math.Hypot(l.X-p.X, l.Y-p.Y) > 0.5 {
			pts = append(pts, p)
		}
	}
	lane := func(s stop) float64 {
		if s.door != s.on {
			return w.lane
		}
		return 0
	}
	add(w.at.on.X+lane(w.at), w.at.on.Y+ly)
	add(next.on.X+lane(next), next.on.Y+ly)
	end := a.standPoint(next, w)
	if next.door != next.on {
		add(next.door.X+w.lane, next.door.Y)
	}
	if l := pts[len(pts)-1]; math.Hypot(l.X-end.X, l.Y-end.Y) > 0.5 {
		pts = append(pts, end)
	}
	w.path, w.seg, w.at, w.hidden = pts, 0, next, false
}

// advance walks a colonist d world px along its path.
func (w *walker) advance(d float64) {
	for d > 0 && w.seg < len(w.path)-1 {
		p := w.path[w.seg+1]
		dx, dy := p.X-w.pos.X, p.Y-w.pos.Y
		l := math.Hypot(dx, dy)
		if l < 1e-9 {
			w.seg++
			continue
		}
		w.heading = math.Atan2(dy, dx)
		if l <= d {
			w.pos = p
			w.seg++
			d -= l
			w.dist += l
			continue
		}
		w.pos.X += dx / l * d
		w.pos.Y += dy / l * d
		w.dist += d
		d = 0
	}
}

func pathLen(ps []game.Vec) float64 {
	n := 0.0
	for i := 1; i < len(ps); i++ {
		n += math.Hypot(ps[i].X-ps[i-1].X, ps[i].Y-ps[i-1].Y)
	}
	return n
}

// newWalker makes a colonist waiting at a stop.
func (a *App) newWalker(i int, at stop) walker {
	c := &a.col
	w := walker{variant: i % 2, speed: 18 + c.rng.Float64()*6, lane: (c.rng.Float64()*2 - 1) * 0.1, at: at}
	w.pos = a.standPoint(at, &w)
	w.heading = c.rng.Float64() * 2 * math.Pi
	return w
}

// spawnWalkers fills the streets, some colonists on their way already.
func (a *App) spawnWalkers() {
	c := &a.col
	c.walkers = c.walkers[:0]
	stops := a.colonyStops()
	for i := 0; i < a.colonyWalkers(); i++ {
		w := a.newWalker(i, stops[c.rng.Intn(len(stops))])
		if c.rng.Intn(5) < 2 {
			w.wait = c.rng.Float64() * 4
			w.hidden = w.at.inside
		} else {
			a.walkTo(&w, stops)
			w.advance(c.rng.Float64() * pathLen(w.path))
		}
		c.walkers = append(c.walkers, w)
	}
}

// inHQ reports whether a point is inside the walls of the command center.
func (a *App) inHQ(p game.Vec) bool {
	l := a.colonyLoc()
	P := float64(art.CellPx)
	return math.Abs(p.X-l.HQCenter().X*P) < 1.3*P && math.Abs(p.Y-l.Avenue*P) < 0.7*P
}

func (a *App) updateWalkers(dt float64) {
	c := &a.col
	stops := a.colonyStops()
	for i := range c.walkers {
		w := &c.walkers[i]
		if w.wait > 0 {
			w.wait -= dt
			if w.wait <= 0 {
				a.walkTo(w, stops)
			}
			continue
		}
		w.advance(w.speed * dt)
		if w.seg >= len(w.path)-1 {
			w.wait = 2 + c.rng.Float64()*4
			w.hidden = w.at.inside
			if b := w.at.b; b != nil && !w.at.inside {
				cc := b.Center()
				w.heading = math.Atan2(cc.Y*art.CellPx-w.pos.Y, cc.X*art.CellPx-w.pos.X)
			}
		}
	}
	// New homes bring new colonists out of the habitat one after another.
	if n := a.colonyWalkers(); len(c.walkers) < n {
		l := a.colonyLoc()
		at := stops[0]
		if h := l.Building(l.ID + ".habitat"); h != nil && a.colonyDone(h) > 0 {
			at = stop{b: h, door: l.Door(h), on: l.Junction(h), inside: true}
		}
		for i := len(c.walkers); i < n; i++ {
			w := a.newWalker(i, at)
			w.wait = 0.4 + float64(i-len(c.walkers))*1.1
			w.hidden = true
			c.walkers = append(c.walkers, w)
		}
	}
}

// ---- The shuttle ----

// shuttleAt is where the shuttle is relative to its landing spot, how
// solid it looks and how hard its engines push.
func (s *shuttle) at() (d art.V3, alpha, thrust float64, ok bool) {
	switch s.phase {
	case shipLanding:
		k := clampF(s.t/landTime, 0, 1)
		e := 1 - math.Pow(1-k, 3)
		return art.V3{X: 90 * (1 - e), Y: -40 * (1 - e), Z: 120 * (1 - e)}, smooth(0, 0.12, k), 0.5 + 0.5*k, true
	case shipParked:
		return art.V3{}, 1, 0, true
	case shipLeaving:
		k := clampF(s.t/leaveTime, 0, 1)
		e := k * k * k
		return art.V3{X: -100 * e, Y: -30 * e, Z: 130 * e}, 1 - smooth(0.85, 1, k), 1, true
	}
	return art.V3{}, 0, 0, false
}

// padOf is the landing pad of the location when shuttles fly to it.
func (a *App) padOf() (*game.Building, *art.V3) {
	l := a.colonyLoc()
	pad := l.Building(l.ID + ".pad")
	if pad == nil || a.colonyLocked(l) || a.colonyBusy(pad) {
		return nil, nil
	}
	st := a.colonyStage(pad)
	if st == 0 {
		return nil, nil
	}
	land := a.as.Colonies[l].fx[pad][st-1].Land
	if land == nil {
		return nil, nil
	}
	return pad, land
}

func (a *App) updateShuttle(dt float64) {
	c := &a.col
	s := &c.ship
	pad, land := a.padOf()
	if pad == nil {
		*s = shuttle{wait: 2}
		return
	}
	s.t += dt
	switch s.phase {
	case shipAway:
		if s.t >= s.wait {
			s.phase, s.t = shipLanding, 0
		}
	case shipLanding:
		if s.t >= landTime {
			s.phase, s.t, s.wait = shipParked, 0, 5+c.rng.Float64()*3
			P := float64(art.CellPx)
			cc := pad.Center()
			x, y := cc.X*P+land.X, cc.Y*P+land.Y
			a.fx.Rings = append(a.fx.Rings, Ring{X: x, Y: y, Z: land.Z, R0: 8, R1: 34, Life: 0.6, Width: 2, Col: art.Tan})
			for i := 0; i < 14; i++ {
				a.fx.Dust(x, y, land.Z, 1)
			}
		}
	case shipParked:
		if s.t >= s.wait {
			s.phase, s.t = shipLeaving, 0
		}
	case shipLeaving:
		if s.t >= leaveTime {
			s.phase, s.t, s.wait = shipAway, 0, 6+c.rng.Float64()*5
		}
	}
	// Engine dust while the shuttle is close to the pad.
	if d, _, th, ok := s.at(); ok && th > 0 && d.Z < 22 {
		s.dustT -= dt
		for s.dustT < 0 {
			s.dustT += 0.035
			P := float64(art.CellPx)
			cc := pad.Center()
			a.fx.Dust(cc.X*P+land.X+d.X, cc.Y*P+land.Y+d.Y, land.Z, 1-d.Z/30)
		}
	}
}

// ---- Update ----

func (a *App) updateColony(dt float64) {
	c := &a.col
	c.t += dt
	c.flash = math.Max(0, c.flash-dt*1.5)
	c.spentT = math.Max(0, c.spentT-dt)
	a.fadeIn = math.Max(0, a.fadeIn-dt*2.5)
	if a.onBridge() {
		a.updateBridge(dt)
		return
	}
	l := a.colonyLoc()
	lk := a.as.Colonies[l]
	P := float64(art.CellPx)
	for i := 0; i < len(c.anims); {
		an := &c.anims[i]
		an.t += dt
		cc := an.b.Center()
		x, y := cc.X*P, cc.Y*P
		sp := lk.stages[an.b][an.stage-1]
		if an.t >= buildTime {
			a.fx.Upgrade(x, y, art.SlabH, an.stage-1)
			text := an.b.Name + " built"
			if an.stage > 1 {
				text = fmt.Sprintf("%s · stage %d", an.b.Name, an.stage)
			}
			a.fx.Float(x, y, sp.oy/art.HZ+4, text, art.Lime, 2.2)
			c.anims = append(c.anims[:i], c.anims[i+1:]...)
			if r := a.colonyRank(l); r != c.rank {
				c.rank = r
				a.toast(fmt.Sprintf("%s is now a %s!", l.Name, r))
			}
			if won := a.save.claimTrophies(); len(won) > 0 {
				a.trophyToast(won)
			}
			continue
		}
		// Sparks crackle along the rising build line.
		rel := a.riseLine(an, lk)
		hw := float64(sp.img.Bounds().Dx()) * 0.35
		for k := 0; k < 2; k++ {
			if rel >= 0 {
				a.fx.RiseSparks(x, y+rel/art.GY, art.SlabH, hw)
			} else {
				a.fx.RiseSparks(x, y, -rel/art.HZ, hw)
			}
		}
		i++
	}
	// Chimney smoke and welding sparks of the finished buildings.
	c.smokeT -= dt
	c.weldT -= dt
	smoke, weld := c.smokeT < 0, c.weldT < 0
	if smoke {
		c.smokeT += 0.32
	}
	if weld {
		c.weldT += 0.5 + c.rng.Float64()*1.4
	}
	if !a.colonyLocked(l) {
		for _, b := range l.Buildings {
			st := a.colonyStage(b)
			if st == 0 || a.colonyBusy(b) {
				continue
			}
			cc := b.Center()
			fx := lk.fx[b][st-1]
			for _, s := range fx.Smoke {
				if smoke {
					a.fx.Chimney(cc.X*P+s.X, cc.Y*P+s.Y, s.Z)
				}
			}
			for _, s := range fx.Sparks {
				if weld {
					a.fx.Weld(cc.X*P+s.X, cc.Y*P+s.Y, s.Z)
				}
			}
		}
	}
	a.updateWalkers(dt)
	a.updateShuttle(dt)
}

// riseLine is the build line of a construction: image rows below the anchor
// of the model (negative: above it).
func (a *App) riseLine(an *buildAnim, lk *colonyLook) float64 {
	nw := lk.stages[an.b][an.stage-1]
	bot, top := float64(nw.img.Bounds().Dy())-nw.oy, nw.oy
	if an.stage > 1 {
		old := lk.stages[an.b][an.stage-2]
		bot = math.Max(bot, float64(old.img.Bounds().Dy())-old.oy)
		top = math.Max(top, old.oy)
	}
	k := easeInOut(clampF(an.t/buildTime, 0, 1))
	return bot - k*(bot+top)
}

// ---- Drawing ----

const (
	ciBuilding = iota
	ciHQ
	ciWalker
	ciShuttle
)

func (a *App) drawColony(dst *ebiten.Image) {
	u := a.u
	cl := a.colonyLayout()
	l := a.colonyLoc()
	v := cl.view
	view := dst.SubImage(image.Rect(v.X, v.Y, v.X+v.W, v.Y+v.H)).(*ebiten.Image)
	fillRect(dst, v.X-u, v.Y-u, v.W+2*u, v.H+2*u, pm(art.Navy, 1))
	fillRect(dst, v.X, v.Y, v.W, v.H, bgColor)
	bridge := a.onBridge()
	locked := a.colonyLocked(l) && !bridge
	switch {
	case bridge:
		a.drawBridgeMap(view)
	case locked:
		a.drawColonyMap(view, l)
		fillRect(dst, v.X, v.Y, v.W, v.H, pm(art.Ink, 0.55))
		a.drawColonyLock(dst, v, l)
	default:
		a.drawColonyMap(view, l)
		switch l.Biome {
		case "ice":
			a.drawColonySnow(view, v)
		case "titan":
			a.drawColonyDrizzle(view, v)
		}
		a.drawPriceTags(view, l)
	}

	// Header.
	m := 10 * u
	a.drawButton(dst, cl.back, 1)
	a.drawText(dst, "Colonies", cl.titleX, m, 2*u, art.White, 1)
	sub := "Crystals from your victories build up the worlds you have freed."
	if bridge {
		sub = "Trophies of your campaign. Each one pays crystals once."
	}
	a.drawText(dst, sub, cl.titleX, m+14*u+6*u, u, art.Silver, 1)
	cr := fmt.Sprint(a.save.Crystals)
	cw := a.textWidth(cr, 2*u)
	x := a.sw - m - cw
	a.drawText(dst, cr, x, m, 2*u, art.White, 1)
	a.drawIcon(dst, "crystal", x-26*u, m-4*u, 2*u, 1)
	if a.col.spentT > 0 {
		k := 1 - a.col.spentT
		s := fmt.Sprintf("-%d", a.col.spent)
		a.drawText(dst, s, a.sw-m-a.textWidth(s, u), m+16*u+int(8*float64(u)*k), u, art.Pink, 1-smooth(0.5, 1, k))
	}
	for _, b := range cl.tabs {
		a.drawButton(dst, b, 1)
	}

	// Panel.
	a.panel(dst, cl.panel, 1)
	switch {
	case bridge && a.br.sel == nil:
		a.drawBridgeOverview(dst, cl)
	case bridge:
		a.drawBridgeDetail(dst, cl)
	case a.col.sel == nil:
		a.drawColonyOverview(dst, cl, l)
	default:
		a.drawColonyDetail(dst, cl, l)
	}

	// Tooltips.
	for i, t := range colonyTabs() {
		if t.tip != "" && a.hoverBtn == fmt.Sprintf("loc:%d", i) {
			r := cl.tabs[i].R
			a.drawTipBox(dst, a.wrap(t.tip, 190*u, u), r.X, r.Y+r.H+4*u, art.Silver)
		}
	}
	if bridge && a.hoverBtn == "" && (a.br.table || a.br.hover != nil && a.br.hover != a.br.sel) {
		a.drawBridgeTip(dst, v)
	} else if h := a.col.hover; h != nil && h != a.col.sel && !locked && a.hoverBtn == "" && !bridge {
		a.drawLotTip(dst, h, v)
	}
}

// drawTipBox draws lines of text in a small panel with its top-left at (x, y).
func (a *App) drawTipBox(dst *ebiten.Image, lines []string, x, y int, col color.RGBA) {
	u := a.u
	w := 0
	for _, l := range lines {
		w = max(w, a.textWidth(l, u))
	}
	r := Rect{max(2*u, min(x, a.sw-w-10*u)), y, w + 8*u, len(lines)*11*u + 5*u}
	a.panel(dst, r, 1)
	for i, l := range lines {
		a.drawText(dst, l, r.X+4*u, r.Y+4*u+i*11*u, u, col, 1)
	}
}

// drawLotTip names the building under the cursor and its next stage.
func (a *App) drawLotTip(dst *ebiten.Image, b *game.Building, v Rect) {
	u := a.u
	st := a.colonyStage(b)
	lines := [][]Seg{{{b.Name, art.White}}}
	switch {
	case a.colonyBusy(b):
		lines = append(lines, []Seg{{"Under construction", art.Cyan}})
	case st >= len(b.Stages):
		lines = append(lines, []Seg{{"Fully built", art.Lime}})
	default:
		what := "Build"
		if st > 0 {
			what = fmt.Sprintf("Stage %d of %d · upgrade", st, len(b.Stages))
		}
		cost := b.Stages[st].Cost
		col := art.Pink
		if a.save.Crystals < cost {
			col = art.Gray
		}
		lines = append(lines, []Seg{{what + " ", art.Silver}, {"◆", col}, {fmt.Sprint(cost), col}})
	}
	pw := 0
	for _, l := range lines {
		pw = max(pw, a.segsWidthIcons(l, u))
	}
	pw += 8 * u
	ph := len(lines)*11*u + 5*u
	x0, y0, x1, _ := a.lotRect(b)
	sx, sy := a.cam.ToScreen((x0+x1)/2, y0)
	x := max(v.X+2*u, min(int(sx)-pw/2, v.X+v.W-pw-2*u))
	y := max(v.Y+2*u, int(sy)-ph-4*u)
	a.panel(dst, Rect{x, y, pw, ph}, 1)
	for i, l := range lines {
		a.drawSegsIcons(dst, l, x+4*u, y+4*u+i*11*u, u, 1)
	}
}

// drawColonyLock covers a location that is not free yet.
func (a *App) drawColonyLock(dst *ebiten.Image, v Rect, l *game.Location) {
	u := a.u
	t1 := fmt.Sprintf("Defeat the %s in %s to free %s.", l.Boss, l.Level.ID, l.Name)
	t2 := "Then spend your crystals here to build its colony."
	w := max(a.textWidth(t1, u), a.textWidth(t2, u)) + 20*u
	h := 26*u + 22*u
	r := Rect{v.X + v.W/2 - w/2, v.Y + v.H/2 - h/2, w, h}
	a.panel(dst, r, 1)
	a.drawIcon(dst, "lock", r.X+r.W/2-11*u, r.Y+6*u, 2*u, 1)
	a.drawText(dst, t1, r.X+r.W/2-a.textWidth(t1, u)/2, r.Y+30*u, u, art.Amber, 1)
	a.drawText(dst, t2, r.X+r.W/2-a.textWidth(t2, u)/2, r.Y+41*u, u, art.Silver, 1)
}

type colItem struct {
	key  float64
	kind int
	i    int
}

// drawColonySnow adds a gentle, pixel-sized snowfall over Europa's terrain.
func (a *App) drawColonySnow(dst *ebiten.Image, v Rect) {
	x0, y0 := a.cam.ToScreen(0, 0)
	x1, y1 := a.cam.ToScreen(a.cam.MapW, a.cam.MapH)
	left, top := math.Max(x0, float64(v.X)), math.Max(y0, float64(v.Y))
	right, bottom := math.Min(x1, float64(v.X+v.W)), math.Min(y1, float64(v.Y+v.H))
	w, h := right-left, bottom-top
	if w <= 1 || h <= 1 {
		return
	}
	for i := 0; i < 48; i++ {
		f := float64(i)
		x := left + math.Mod(f*83.7+a.col.t*(2+float64(i%3)), w)
		y := top + math.Mod(f*127.9+a.col.t*(6+float64(i%4)), h)
		size := max(1, a.u-1)
		fillRect(dst, int(x), int(y), size, size, pm(art.Silver, 0.5))
	}
}

// drawColonyDrizzle lets a slow methane drizzle fall over Titan's map: long,
// faint amber drops that sink far slower than rain on Earth.
func (a *App) drawColonyDrizzle(dst *ebiten.Image, v Rect) {
	x0, y0 := a.cam.ToScreen(0, 0)
	x1, y1 := a.cam.ToScreen(a.cam.MapW, a.cam.MapH)
	left, top := math.Max(x0, float64(v.X)), math.Max(y0, float64(v.Y))
	right, bottom := math.Min(x1, float64(v.X+v.W)), math.Min(y1, float64(v.Y+v.H))
	w, h := right-left, bottom-top
	if w <= 1 || h <= 1 {
		return
	}
	size := max(1, a.u-1)
	for i := 0; i < 40; i++ {
		f := float64(i)
		x := left + math.Mod(f*91.3+a.col.t*(1.5+float64(i%3)), w)
		y := top + math.Mod(f*131.7+a.col.t*(12+3*float64(i%4)), h)
		fillRect(dst, int(x), int(y), size, 3*size, pm(art.Sand, 0.3))
	}
}

// drawColonyMap draws the ground, the buildings, the colonists and the
// effects of a location into the map view.
func (a *App) drawColonyMap(dst *ebiten.Image, l *game.Location) {
	as, c := a.as, &a.col
	lk := as.Colonies[l]
	P := float64(art.CellPx)
	t := c.t
	locked := a.colonyLocked(l)
	a.drawGround(dst)
	hq := l.HQCenter()
	if locked {
		a.drawColonyHQ(dst, hq.X*P, hq.Y*P, t)
		return
	}

	// Lot outlines under the buildings.
	for _, b := range l.Buildings {
		al := 0.0
		col := art.Cyan
		switch {
		case b == c.sel:
			al = 0.75 + 0.25*math.Sin(a.now*5)
		case b == c.hover:
			al, col = 0.6, art.Silver
		case a.colonyStage(b) == 0:
			al = 0.22
		}
		if al > 0 {
			a.lotOutline(dst, b, col, al)
		}
	}

	// Shadows of everything that moves or grows.
	sl := a.shadowLayer()
	for _, b := range l.Buildings {
		st := a.colonyStage(b)
		cc := b.Center()
		x, y := cc.X*P, cc.Y*P
		if an := a.colonyAnim(b); an != nil {
			k := clampF(an.t/buildTime, 0, 1)
			nw := lk.stages[b][an.stage-1]
			a.blit(sl, nw.sh, x, y, 0, nw.sx, nw.sy, 1, csAlpha(k), false)
			if an.stage > 1 {
				old := lk.stages[b][an.stage-2]
				a.blit(sl, old.sh, x, y, 0, old.sx, old.sy, 1, csAlpha(1-k), false)
			}
			continue
		}
		if st > 0 {
			sp := lk.stages[b][st-1]
			a.blit(sl, sp.sh, x, y, 0, sp.sx, sp.sy, 1, ebiten.ColorScale{}, false)
		}
	}
	for i := range c.walkers {
		w := &c.walkers[i]
		if w.hidden || a.inHQ(w.pos) {
			continue
		}
		sp := a.walkerSprite(w)
		a.blit(sl, sp.sh, w.pos.X, w.pos.Y, a.ground(w.pos.X, w.pos.Y), sp.sx, sp.sy, 1, ebiten.ColorScale{}, false)
	}
	pad, land := a.padOf()
	if pad != nil {
		if d, al, _, ok := c.ship.at(); ok {
			cc := pad.Center()
			x, y := cc.X*P+land.X+d.X, cc.Y*P+land.Y+d.Y
			sp := as.Shuttle
			a.blit(sl, sp.sh, x+d.Z*shadowDX, y+d.Z*shadowDY, land.Z, sp.sx, sp.sy, 1, csAlpha(al*(1-d.Z/160)), false)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(art.ShadowAlpha)
	dst.DrawImage(sl, op)
	a.drawRingFX(dst)

	// Objects back to front.
	items := []colItem{{hq.Y*P + 0.5, ciHQ, 0}}
	for i, b := range l.Buildings {
		items = append(items, colItem{b.Center().Y * P, ciBuilding, i})
	}
	for i := range c.walkers {
		if w := &c.walkers[i]; !w.hidden && !a.inHQ(w.pos) {
			items = append(items, colItem{w.pos.Y, ciWalker, i})
		}
	}
	if pad != nil {
		if d, _, _, ok := c.ship.at(); ok {
			k := pad.Center().Y*P + float64(pad.H)*P/2
			if d.Z > 30 {
				k = math.Inf(1)
			}
			items = append(items, colItem{k, ciShuttle, 0})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].key < items[j].key })
	for _, it := range items {
		switch it.kind {
		case ciHQ:
			a.drawColonyHQ(dst, hq.X*P, hq.Y*P, t)
		case ciBuilding:
			a.drawColonyBuilding(dst, lk, l.Buildings[it.i], it.i)
		case ciWalker:
			w := &c.walkers[it.i]
			sp := a.walkerSprite(w)
			a.blit(dst, sp.img, w.pos.X, w.pos.Y, a.ground(w.pos.X, w.pos.Y), sp.ox, sp.oy, 1, ebiten.ColorScale{}, false)
		case ciShuttle:
			a.drawShuttle(dst, pad, land)
		}
	}
	a.drawParticles(dst)
	a.drawFloaters(dst)
}

func (a *App) walkerSprite(w *walker) sprite {
	f := 0
	if w.wait <= 0 {
		f = int(w.dist/walkStride*colonistFrames) % colonistFrames
	}
	return a.as.Colonists[w.variant][angleIndex(w.heading, colonistAngles)][f]
}

// lotOutline draws the edge of a lot on its foundation.
func (a *App) lotOutline(dst *ebiten.Image, b *game.Building, col color.RGBA, al float64) {
	P := float64(art.CellPx)
	x0, y0 := float64(b.Lot.X)*P+3, float64(b.Lot.Y)*P+3
	x1, y1 := float64(b.Lot.X+b.W)*P-3, float64(b.Lot.Y+b.H)*P-3
	pts := [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	lw := float32(math.Max(1, a.cam.Z*1.2))
	for i := range pts {
		p, q := pts[i], pts[(i+1)%4]
		ax, ay := a.screenAt(p[0], p[1], art.SlabH)
		bx, by := a.screenAt(q[0], q[1], art.SlabH)
		vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), lw, pm(col, al), true)
	}
}

// drawColonyHQ draws the command center with its radar and lights.
func (a *App) drawColonyHQ(dst *ebiten.Image, cx, cy, t float64) {
	as := a.as
	b := as.BaseEast
	a.blit(dst, b.img, cx, cy, 0, b.ox, b.oy, 1, ebiten.ColorScale{}, false)
	r := as.Radar[angleIndex(t*1.3, radarAngles)]
	rp := art.RadarPos
	a.blit(dst, r.img, cx+rp.X, cy+rp.Y, rp.Z, r.ox, r.oy, 1, ebiten.ColorScale{}, false)
	for i, l := range art.BaseLights {
		on := smooth(0.35, 0.8, 0.5+0.5*math.Sin(t*2.6+float64(i)*2.1))
		a.put(dst, as.White, cx+l.X, cy+l.Y, l.Z, 2.4, 1.6, csCol(art.Lime, 0.25+0.75*on), false)
		a.put(dst, as.Glow["green"], cx+l.X, cy+l.Y, l.Z, 0.35, 0.3, csAlpha(0.5*on), true)
	}
	d := art.DomeCenter
	a.put(dst, as.Glow["cyan"], cx+d.X, cy+d.Y, d.Z, 1.4, 1.2, csAlpha(0.12+0.06*math.Sin(t*1.7)), true)
}

// drawColonyBuilding draws a building at its stage: a hologram on an empty
// lot, the model rising during construction, its lights when it stands.
func (a *App) drawColonyBuilding(dst *ebiten.Image, lk *colonyLook, b *game.Building, i int) {
	P := float64(art.CellPx)
	c := &a.col
	cc := b.Center()
	x, y := cc.X*P, cc.Y*P
	if an := a.colonyAnim(b); an != nil {
		a.drawRising(dst, lk, an, x, y)
		return
	}
	st := a.colonyStage(b)
	if st == 0 {
		sp := lk.stages[b][0]
		al := 0.26 + 0.06*math.Sin(c.t*3+float64(i)*1.7)
		if b == c.hover || b == c.sel {
			al += 0.22
		}
		a.blit(dst, lk.holo[b], x, y, 0, sp.ox, sp.oy, 1, csAlpha(al), true)
		return
	}
	sp := lk.stages[b][st-1]
	a.blit(dst, sp.img, x, y, 0, sp.ox, sp.oy, 1, ebiten.ColorScale{}, false)
	if b == c.sel {
		k := 0.05 + 0.04*math.Sin(a.now*5)
		a.blit(dst, sp.sil, x, y, 0, sp.ox, sp.oy, 1, csCol(art.Cyan, k), true)
	}
	a.drawBuildingLights(dst, lk.fx[b][st-1], x, y, c.t+float64(i)*0.7)
}

// drawBuildingLights blinks the lamps of a building.
func (a *App) drawBuildingLights(dst *ebiten.Image, fx art.BuildingFX, x, y, t float64) {
	as := a.as
	lamp := func(p art.V3, col color.RGBA, glow string, on float64) {
		a.put(dst, as.White, x+p.X, y+p.Y, p.Z, 1.8, 1.4, csCol(col, 0.3+0.7*on), false)
		a.put(dst, as.Glow[glow], x+p.X, y+p.Y, p.Z, 0.3, 0.26, csAlpha(0.55*on), true)
	}
	for i, p := range fx.Red {
		lamp(p, art.Red, "red", smooth(0.55, 0.9, 0.5+0.5*math.Sin(t*2.2+float64(i)*1.3)))
	}
	for i, p := range fx.Green {
		lamp(p, art.Lime, "green", 0.55+0.45*math.Sin(t*1.4+float64(i)*2))
	}
	for i, p := range fx.Amber {
		ph := math.Mod(t*1.2+float64(i)*0.25, 1)
		lamp(p, art.Amber, "amber", 1-smooth(0.05, 0.35, ph))
	}
	for i, p := range fx.Glow {
		k := 0.16 + 0.05*math.Sin(t*1.1+float64(i))
		a.put(dst, as.Glow["green"], x+p.X, y+p.Y, p.Z, 0.9, 0.6, csAlpha(k), true)
	}
}

// blitRows draws the image rows r0..r1 of a sprite at its anchor.
func (a *App) blitRows(dst, img *ebiten.Image, sp sprite, x, y float64, r0, r1 int, cs ebiten.ColorScale, add bool) {
	b := img.Bounds()
	r0, r1 = max(0, r0), min(b.Dy(), r1)
	if r1 <= r0 {
		return
	}
	a.blit(dst, img.SubImage(image.Rect(0, r0, b.Dx(), r1)).(*ebiten.Image), x, y, 0, sp.ox, sp.oy-float64(r0), 1, cs, add)
}

// drawRising shows a stage being built from the bottom up: the new model
// below the build line, the old one (or a hologram) above it.
func (a *App) drawRising(dst *ebiten.Image, lk *colonyLook, an *buildAnim, x, y float64) {
	nw := lk.stages[an.b][an.stage-1]
	rel := a.riseLine(an, lk)
	row := func(sp sprite) int { return int(math.Round(sp.oy + rel)) }
	flick := 0.85 + 0.15*math.Sin(a.now*19)
	holo := art.Cyan
	if an.stage > 1 {
		old := lk.stages[an.b][an.stage-2]
		a.blitRows(dst, old.img, old, x, y, 0, row(old), ebiten.ColorScale{}, false)
		a.blitRows(dst, old.sil, old, x, y, 0, row(old), csCol(holo, 0.12*flick), true)
	} else {
		a.blitRows(dst, lk.holo[an.b], nw, x, y, 0, row(nw), csAlpha(0.45*flick), true)
	}
	big := 1 << 20
	a.blitRows(dst, nw.img, nw, x, y, row(nw), big, ebiten.ColorScale{}, false)
	a.blitRows(dst, nw.sil, nw, x, y, row(nw), big, csCol(holo, 0.18), true)
	// The build line.
	w := float64(nw.img.Bounds().Dx()) * 0.42 * a.cam.Z
	sx, sy := a.screenAt(x, y, 0)
	sy += rel * a.cam.Z
	lw := float32(math.Max(1, a.cam.Z))
	vector.StrokeLine(dst, float32(sx-w), float32(sy), float32(sx+w), float32(sy), lw*2.5, pm(holo, 0.25*flick), true)
	vector.StrokeLine(dst, float32(sx-w*0.8), float32(sy), float32(sx+w*0.8), float32(sy), lw, pm(art.White, 0.8*flick), true)
}

// drawShuttle draws the supply shuttle and the glow of its engines.
func (a *App) drawShuttle(dst *ebiten.Image, pad *game.Building, land *art.V3) {
	d, al, th, ok := a.col.ship.at()
	if !ok || al <= 0 {
		return
	}
	P := float64(art.CellPx)
	cc := pad.Center()
	x, y, z := cc.X*P+land.X+d.X, cc.Y*P+land.Y+d.Y, land.Z+d.Z
	sp := a.as.Shuttle
	a.blit(dst, sp.img, x, y, z, sp.ox, sp.oy, 1, csAlpha(al), false)
	if th > 0 {
		k := th * (0.8 + 0.2*math.Sin(a.now*40))
		for _, s := range []float64{-1, 1} {
			a.put(dst, a.as.Glow["orange"], x+17.5, y+s*3.4, z+5.5, 0.32*k, 0.32*k, csAlpha(al*k), true)
		}
		a.put(dst, a.as.Glow["amber"], x, y, z+1, 0.9*k, 0.6*k, csAlpha(0.5*al*k), true)
	}
}

// drawPriceTags shows the price of the next stage over every lot.
func (a *App) drawPriceTags(dst *ebiten.Image, l *game.Location) {
	u := a.u
	for _, b := range l.Buildings {
		st := a.colonyStage(b)
		if st >= len(b.Stages) || a.colonyBusy(b) || b == a.col.sel {
			continue
		}
		cost := b.Stages[st].Cost
		s := fmt.Sprint(cost)
		col := art.Pink
		al := 1.0
		if a.save.Crystals < cost {
			col, al = art.Silver, 0.75
		}
		w := 12*u + a.textWidth(s, u) + 5*u
		if st > 0 {
			w += 9 * u
		}
		x0, y0, x1, _ := a.lotRect(b)
		sx, sy := a.cam.ToScreen((x0+x1)/2, y0)
		x, y := int(sx)-w/2, int(sy)-13*u
		fillRect(dst, x, y, w, 12*u, pm(art.Ink, 0.8*al))
		fillRect(dst, x, y+12*u, w, u, pm(art.Navy, al))
		ix := x + 2*u
		if st > 0 {
			a.drawIcon(dst, "up", ix, y+u, u, al)
			ix += 9 * u
		}
		a.drawIcon(dst, "crystal", ix, y+u, u, al)
		a.drawText(dst, s, ix+12*u, y+3*u, u, col, al)
	}
}

// drawColonyOverview is the panel without a selected building: the rank of
// the colony and the list of its buildings.
func (a *App) drawColonyOverview(dst *ebiten.Image, cl colonyLayout, l *game.Location) {
	u := a.u
	r := cl.panel
	x, w := r.X+8*u, r.W-16*u
	y := r.Y + 8*u
	locked := a.colonyLocked(l)
	a.drawText(dst, l.Name, x, y, 2*u, art.White, 1)
	y += 18 * u
	if locked {
		a.drawIcon(dst, "lock", x, y-2*u, u, 1)
		a.drawText(dst, "Not free yet", x+14*u, y, u, art.Amber, 1)
	} else {
		a.drawText(dst, fmt.Sprintf("Freed in %s · %s", l.Level.ID, l.Boss), x, y, u, art.Gray, 1)
	}
	y += 17 * u
	built, total := a.colonyBuilt(l), l.Stages()
	rank, next := game.Rank(built, total)
	a.drawSegs(dst, []Seg{{"Rank ", art.Gray}, {rank, art.Cyan}}, x, y, u, 1)
	cnt := fmt.Sprintf("%d/%d stages", built, total)
	a.drawText(dst, cnt, x+w-a.textWidth(cnt, u), y, u, art.Silver, 1)
	y += 11 * u
	fillRect(dst, x, y, w, 5*u, pm(art.Navy, 1))
	if fw := w * built / max(1, total); fw > 0 {
		fillRect(dst, x, y, fw, 5*u, pm(art.Blue, 1))
		fillRect(dst, x, y, fw, u, pm(art.Cyan, 1))
	}
	for _, rk := range game.Ranks[1 : len(game.Ranks)-1] {
		fillRect(dst, x+w*rk.Pct/100, y-u, u, 7*u, pm(art.Silver, 1))
	}
	y += 10 * u
	if next >= 0 {
		name := ""
		for _, rk := range game.Ranks {
			if rk.Pct == next {
				name = rk.Name
			}
		}
		a.drawText(dst, fmt.Sprintf("Next rank: %s at %d%%", name, next), x, y, u, art.Gray, 1)
	} else {
		a.drawText(dst, "The colony is complete!", x, y, u, art.Lime, 1)
	}
	y += 15 * u
	segs := []Seg{{"Colonists ", art.Gray}, {fmt.Sprint(a.colonists(l)), art.White}}
	if a.colonists(l) == 0 && !locked {
		segs = append(segs, Seg{"  · build a Habitat", art.Gray})
	}
	a.drawSegs(dst, segs, x, y, u, 1)
	y += 17 * u
	for _, line := range a.wrap(l.Desc, w, u) {
		a.drawText(dst, line, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 6 * u
	a.drawText(dst, "Buildings", x, y, u, art.Gray, 1)
	for i, btn := range cl.rows {
		a.drawColonyRow(dst, btn, l.Buildings[i], locked)
	}
	// How to earn more, if there is room under the list.
	if n := len(cl.rows); n > 0 {
		y = cl.rows[n-1].R.Y + cl.rows[n-1].R.H + 8*u
		lines := a.wrap("Win stars in the campaign to earn crystals. Replays pay a little too.", w, u)
		if y+len(lines)*11*u <= r.Y+r.H-6*u {
			for _, line := range lines {
				a.drawText(dst, line, x, y, u, art.Gray, 1)
				y += 11 * u
			}
		}
	}
}

// drawColonyRow is one building in the overview list.
func (a *App) drawColonyRow(dst *ebiten.Image, btn Button, b *game.Building, locked bool) {
	u := a.u
	r := btn.R
	bg := pm(art.Navy, 0.5)
	if a.hoverBtn == btn.ID {
		bg = pm(art.Slate, 0.7)
	}
	fillRect(dst, r.X, r.Y, r.W, r.H, bg)
	st, n := a.colonyStage(b), len(b.Stages)
	ty := r.Y + (r.H-7*u)/2
	iy := r.Y + (r.H-11*u)/2
	nameCol := art.White
	if st == 0 || locked {
		nameCol = art.Silver
	}
	a.drawText(dst, b.Name, r.X+4*u, ty, u, nameCol, 1)
	rx := r.X + r.W - 4*u
	switch {
	case a.colonyBusy(b):
		s := "Building"
		rx -= a.textWidth(s, u)
		a.drawText(dst, s, rx, ty, u, art.Cyan, 0.6+0.4*math.Sin(a.now*6))
	case st >= n:
		rx -= 11 * u
		a.drawIcon(dst, "check", rx, iy, u, 1)
	default:
		cost := b.Stages[st].Cost
		s := fmt.Sprint(cost)
		col, al := art.Pink, 1.0
		if locked || a.save.Crystals < cost {
			col, al = art.Gray, 0.7
		}
		rx -= a.textWidth(s, u)
		a.drawText(dst, s, rx, ty, u, col, 1)
		rx -= 13 * u
		a.drawIcon(dst, "crystal", rx, iy, u, al)
	}
	px := rx - 5*u - n*5*u
	for i := 0; i < n; i++ {
		col := art.Slate
		if i < st {
			col = art.Lime
		}
		fillRect(dst, px+i*5*u, r.Y+r.H/2-2*u, 4*u, 4*u, pm(col, 1))
	}
}

// drawColonyDetail is the panel of the selected building: its stages and
// the button that builds the next one.
func (a *App) drawColonyDetail(dst *ebiten.Image, cl colonyLayout, l *game.Location) {
	u := a.u
	r := cl.panel
	b := a.col.sel
	x, w := r.X+8*u, r.W-16*u
	a.drawButton(dst, cl.all, 1)
	y := cl.all.R.Y + cl.all.R.H + 9*u
	a.drawText(dst, b.Name, x, y, 2*u, art.White, 1)
	y += 18 * u
	st, n := a.colonyStage(b), len(b.Stages)
	switch {
	case a.colonyBusy(b):
		a.drawText(dst, "Under construction", x, y, u, art.Cyan, 1)
	case st == 0:
		a.drawText(dst, "Not built yet", x, y, u, art.Gray, 1)
	case st < n:
		a.drawText(dst, fmt.Sprintf("Stage %d of %d", st, n), x, y, u, art.Cyan, 1)
	default:
		a.drawText(dst, "Fully built", x, y, u, art.Lime, 1)
	}
	y += 17 * u
	for _, line := range a.wrap(b.Desc, w, u) {
		a.drawText(dst, line, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 8 * u
	label := "Stages"
	if n == 1 {
		label = "Stage"
	}
	a.drawText(dst, label, x, y, u, art.Gray, 1)
	y += 14 * u
	tw := w - 14*u - 28*u
	for i, s := range b.Stages {
		col := art.Gray
		if i == st {
			col = art.Silver
		}
		if i < st {
			a.drawIcon(dst, "check", x, y-2*u, u, 1)
			col = art.White
		} else {
			num := fmt.Sprint(i + 1)
			a.drawText(dst, num, x+(11*u-a.textWidth(num, u))/2, y, u, col, 1)
			cost := fmt.Sprint(s.Cost)
			cc, al := art.Gray, 0.5
			if i == st {
				al = 1
				if a.save.Crystals >= s.Cost {
					cc = art.Pink
				}
			}
			cw := a.textWidth(cost, u)
			a.drawText(dst, cost, x+w-cw, y, u, cc, 1)
			a.drawIcon(dst, "crystal", x+w-cw-12*u, y-2*u, u, al)
		}
		for _, line := range a.wrap(s.Text, tw, u) {
			a.drawText(dst, line, x+14*u, y, u, col, 1)
			y += 11 * u
		}
		y += 4 * u
	}
	bt := cl.build
	if bt.ID == "" {
		msg := "Every stage is built."
		if n == 1 {
			msg = "It stands proud over the colony."
		}
		a.drawText(dst, msg, x, r.Y+r.H-8*u-18*u, u, art.Lime, 1)
		return
	}
	a.drawButton(dst, bt, 1)
	hy := bt.R.Y + bt.R.H + 6*u
	switch {
	case a.colonyLocked(l):
		a.drawText(dst, fmt.Sprintf("Defeat the %s in %s first.", l.Boss, l.Level.ID), x, hy, u, art.Amber, 1)
	case a.colonyBusy(b):
	case a.save.Crystals < bt.Cost:
		col := lerpCol(art.Amber, art.Red, a.col.flash)
		a.drawText(dst, fmt.Sprintf("Need %d more crystals", bt.Cost-a.save.Crystals), x, hy, u, col, 1)
	default:
		a.drawText(dst, fmt.Sprintf("You have %d crystals", a.save.Crystals), x, hy, u, art.Gray, 1)
	}
}
