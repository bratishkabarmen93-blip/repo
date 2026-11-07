package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
	"outpost/internal/game"
)

// ---- The bridge of the Vanguard: trophies on pedestals, specimens in jars ----

const (
	pedAngles   = 16   // turns of a pedestal hologram
	tableAngles = 24   // turns of the big hologram over the holo-table
	holoTurn    = 0.55 // radians per second
	idleShow    = 7.0  // seconds each won trophy is shown while none is picked
	revealTime  = 0.7  // seconds a new projection takes to build up
	tableZ      = art.TableH + 10
	pedZ        = art.PedestalH + 3
	tableBoxW   = 96 // the big hologram fits this box (world px)
	tableBoxH   = 88
)

// bridgeState is the bridge tab of the colony screen.
type bridgeState struct {
	sel, hover *game.Trophy
	table      bool // the cursor is over the holo-table
	t          float64
	show       string  // the model on the holo-table
	showT      float64 // seconds it has been projected
	idle       int     // which won trophy the idle table shows
	idleT      float64
	holos      map[string]*tableHolo
}

// tableHolo is the big projection of one model, baked in the background.
type tableHolo struct {
	ready         atomic.Bool
	holoC, ghostC []*art.Canvas
	anch          [][2]int
	holo, ghost   []sprite
}

// onBridge reports whether the colony screen shows the bridge.
func (a *App) onBridge() bool {
	t := colonyTabs()
	return a.col.tab >= 0 && a.col.tab < len(t) && t[a.col.tab].loc == nil
}

func (a *App) won(t *game.Trophy) bool { return a.save.Trophies[t.ID] }

func trophyIndex(t *game.Trophy) int {
	for i, x := range game.AllTrophies {
		if x == t {
			return i
		}
	}
	return -1
}

// trophyEarned is the crystals the trophies won so far have paid.
func (a *App) trophyEarned() int {
	n := 0
	for _, t := range game.AllTrophies {
		if a.won(t) {
			n += t.Reward
		}
	}
	return n
}

// tableModel is what the holo-table projects: the picked trophy (a dim
// outline if it is not won yet), else the won trophies in turn, else the
// Vanguard herself.
func (a *App) tableModel() (model string, ghost bool) {
	b := &a.br
	if b.sel != nil {
		return b.sel.Model, !a.won(b.sel)
	}
	if ws := a.wonTrophies(); len(ws) > 0 {
		return ws[b.idle%len(ws)].Model, false
	}
	return "ship", false
}

func (a *App) wonTrophies() []*game.Trophy {
	var ws []*game.Trophy
	for _, t := range game.Trophies {
		if a.won(t) {
			ws = append(ws, t)
		}
	}
	return ws
}

// tableName names what the holo-table shows.
func (a *App) tableName() string {
	m, _ := a.tableModel()
	for _, t := range game.AllTrophies {
		if t.Model == m {
			return t.Name
		}
	}
	return "Vanguard"
}

// holoFor returns the big projection of a model, starting its bake in the
// background the first time and uploading it once it is done.
func (a *App) holoFor(model string) *tableHolo {
	b := &a.br
	if b.holos == nil {
		b.holos = map[string]*tableHolo{}
	}
	h := b.holos[model]
	if h == nil {
		h = &tableHolo{}
		b.holos[model] = h
		go bakeTableHolo(model, h)
	}
	if h.holo == nil && h.ready.Load() {
		for i := range h.holoC {
			ox, oy := float64(h.anch[i][0]), float64(h.anch[i][1])
			h.holo = append(h.holo, sprite{img: toImage(h.holoC[i]), ox: ox, oy: oy})
			h.ghost = append(h.ghost, sprite{img: toImage(h.ghostC[i]), ox: ox, oy: oy})
		}
		h.holoC, h.ghostC = nil, nil
	}
	return h
}

func bakeTableHolo(model string, h *tableHolo) {
	k := art.TrophyFit(model, tableBoxW, tableBoxH)
	n := tableAngles
	holo, ghost := make([]*art.Canvas, n), make([]*art.Canvas, n)
	anch := make([][2]int, n)
	art.Parallel(n, func(i int) {
		sp := art.TrophyModel(model, k, angleOf(i, n)+0.6)
		holo[i], ghost[i] = art.Holo(sp.C, 0.55), art.Ghost(sp.C)
		anch[i] = [2]int{sp.OX, sp.OY}
	})
	h.holoC, h.ghostC, h.anch = holo, ghost, anch
	h.ready.Store(true)
}

// ---- Layout ----

// bridgeHeadH is the height of the overview above the list of trophies.
func (a *App) bridgeHeadH() int { return 18*a.u + 12*a.u + 12*a.u + 15*a.u }

// layoutBridge fills the panel buttons of the bridge into the layout.
func (a *App) layoutBridge(cl *colonyLayout, px, iw int) {
	u := a.u
	if a.br.sel == nil {
		y := cl.panel.Y + 8*u + a.bridgeHeadH()
		for i := range game.Trophies {
			cl.rows = append(cl.rows, Button{ID: fmt.Sprintf("tsel:%d", i), R: Rect{px - 3*u, y, iw + 6*u, 11 * u}, Enabled: true})
			y += 12 * u
		}
		y += 5*u + 11*u
		n := len(game.Specimens)
		gap := 2 * u
		bw := (iw + 6*u - gap*(n-1)) / n
		for i := range game.Specimens {
			cl.rows = append(cl.rows, Button{ID: fmt.Sprintf("tsel:%d", len(game.Trophies)+i),
				R: Rect{px - 3*u + i*(bw+gap), y, bw, 22 * u}, Enabled: true})
		}
		return
	}
	cl.all = Button{ID: "tall", Label: "All trophies", Key: "Esc", Enabled: true}
	cl.all.R = Rect{px, cl.panel.Y + 8*u, a.buttonWidth(cl.all), a.btnHeight()}
}

// pressBridge handles the panel buttons of the bridge.
func (a *App) pressBridge(id string) bool {
	if id == "tall" {
		a.br.sel = nil
		return true
	}
	if n, ok := cutIndex(id, "tsel:"); ok && n < len(game.AllTrophies) {
		a.br.sel = game.AllTrophies[n]
		return true
	}
	return false
}

// ---- The map ----

// pedestalBox is the box of a pedestal with its hologram in terrain image
// pixels.
func (a *App) pedestalBox(i int) (x0, y0, x1, y1 float64) {
	p := art.PedestalPos(i)
	oy := float64(a.as.TI.OY)
	cy := oy + p.Y*art.GY
	return p.X - 17, cy - (pedZ+36)*art.HZ, p.X + 17, cy + (art.PedestalR+2)*art.GY
}

// jarBox is the box of a specimen jar in terrain image pixels.
func (a *App) jarBox(i int) (x0, y0, x1, y1 float64) {
	p := art.JarPos(i)
	oy := float64(a.as.TI.OY)
	cy := oy + p.Y*art.GY - p.Z*art.HZ
	top := (art.JarBase + art.JarBody + art.JarCap) * art.HZ
	return p.X - art.JarRX - 2, cy - top - art.JarRX*art.GY, p.X + art.JarRX + 2, cy + art.JarRX*art.GY + 1
}

// tableBox is the holo-table with the space of its projection.
func (a *App) tableBox() (x0, y0, x1, y1 float64) {
	tb := art.BridgeTable
	cy := float64(a.as.TI.OY) + tb.Y*art.GY
	r := art.TableR + 4
	return tb.X - r, cy - (tableZ+tableBoxH)*art.HZ, tb.X + r, cy + r*art.GY
}

// trophyAt finds the trophy (pedestal or jar) under a screen point, or
// reports the holo-table.
func (a *App) trophyAt(mx, my int) (*game.Trophy, bool) {
	if !a.colonyView().Has(mx, my) {
		return nil, false
	}
	wx, wy := a.cam.ToWorld(float64(mx), float64(my))
	in := func(x0, y0, x1, y1 float64) bool { return wx >= x0 && wx < x1 && wy >= y0 && wy < y1 }
	var best *game.Trophy
	bestY := math.Inf(-1)
	for i, t := range game.Trophies {
		if p := art.PedestalPos(i); in(a.pedestalBox(i)) && p.Y > bestY {
			best, bestY = t, p.Y
		}
	}
	if best != nil {
		return best, false
	}
	for i, t := range game.Specimens {
		if in(a.jarBox(i)) {
			return t, false
		}
	}
	return nil, in(a.tableBox())
}

// bridgeClick picks the trophy under the cursor (or none).
func (a *App) bridgeClick(mx, my int) {
	if !a.colonyView().Has(mx, my) {
		return
	}
	t, table := a.trophyAt(mx, my)
	if table {
		return
	}
	a.br.sel = t
}

// bridgeKeys: Esc lets go of the trophy or leaves, Tab steps through them.
func (a *App) bridgeKeys() bool {
	jp := inpututil.IsKeyJustPressed
	b := &a.br
	if jp(ebiten.KeyEscape) {
		if b.sel != nil {
			b.sel = nil
		} else {
			a.closeColony()
		}
		return true
	}
	if jp(ebiten.KeyTab) {
		all := game.AllTrophies
		i := trophyIndex(b.sel) + 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			i -= 2
			if b.sel == nil {
				i = len(all) - 1
			}
		}
		b.sel = all[(i%len(all)+len(all))%len(all)]
	}
	return false
}

// ---- Update ----

func (a *App) updateBridge(dt float64) {
	b := &a.br
	b.t += dt
	if b.sel == nil {
		if b.idleT += dt; b.idleT > idleShow {
			b.idleT = 0
			b.idle++
		}
	} else {
		b.idleT = 0
	}
	model, _ := a.tableModel()
	if model != b.show {
		b.show, b.showT = model, 0
	}
	if h := a.holoFor(model); h.holo != nil {
		b.showT += dt
	}
	// Bake the next idle projection ahead of time.
	if ws := a.wonTrophies(); b.sel == nil && len(ws) > 1 {
		a.holoFor(ws[(b.idle+1)%len(ws)].Model)
	}
}

// ---- Drawing ----

// drawBridgeMap draws the room, the lamps of its display, the jars, the
// holograms and the crew into the map view.
func (a *App) drawBridgeMap(dst *ebiten.Image) {
	b := &a.br
	a.drawGround(dst)
	a.drawBridgeWall(dst)
	for i := range game.Specimens {
		a.drawJar(dst, i)
	}
	// Rings on the floor around the picked and the hovered pedestal.
	for i, t := range game.Trophies {
		if t != b.sel && t != b.hover {
			continue
		}
		p := art.PedestalPos(i)
		col, al := art.Silver, 0.7
		if t == b.sel {
			col, al = art.Cyan, 0.75+0.25*math.Sin(a.now*5)
		}
		rg := a.as.Ring(int(art.PedestalR)+6, col)
		a.blit(dst, rg.img, p.X, p.Y, 0, rg.ox, rg.oy, 1, csAlpha(al), false)
	}
	if b.table {
		tb := art.BridgeTable
		rg := a.as.Ring(int(art.TableR)+12, art.Silver)
		a.blit(dst, rg.img, tb.X, tb.Y, 0, rg.ox, rg.oy, 1, csAlpha(0.6), false)
	}
	// Everything on the deck, back to front.
	type item struct {
		y    float64
		kind int // 0 pedestal, 1 table, 2 crew
		i    int
	}
	var items []item
	for i := range game.Trophies {
		items = append(items, item{art.PedestalPos(i).Y, 0, i})
	}
	items = append(items, item{art.BridgeTable.Y, 1, 0})
	for i, p := range art.CrewPos {
		items = append(items, item{p.Y, 2, i})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].y < items[j].y })
	for _, it := range items {
		switch it.kind {
		case 0:
			a.drawPedestal(dst, it.i)
		case 1:
			a.drawHoloTable(dst)
		case 2:
			p := art.CrewPos[it.i]
			a.put(dst, a.as.Glow["white"], p.X+2, p.Y+1, 0, 0.55, 0.3, csCol(art.Ink, 0.55), false)
			sp := a.as.Bridge.crew[it.i]
			a.blit(dst, sp.img, p.X, p.Y, 0, sp.ox, sp.oy, 1, ebiten.ColorScale{}, false)
		}
	}
	a.drawParticles(dst)
	a.drawFloaters(dst)
}

// imgRect fills a rectangle given in terrain image pixels.
func (a *App) imgRect(dst *ebiten.Image, x, y, w, h float64, col ebiten.ColorScale, add bool) {
	x0, y0 := a.cam.ToScreen(x, y)
	x1, y1 := a.cam.ToScreen(x+w, y+h)
	op := &a.op
	*op = ebiten.DrawImageOptions{}
	op.GeoM.Scale(x1-x0, y1-y0)
	op.GeoM.Translate(x0, y0)
	op.ColorScale = col
	if add {
		op.Blend = ebiten.BlendLighter
	}
	dst.DrawImage(a.as.White, op)
}

// drawBridgeWall lights the lamps of the status display, sweeps its scan
// line and lets a few stars twinkle in the window.
func (a *App) drawBridgeWall(dst *ebiten.Image) {
	t := a.br.t
	for i, tr := range game.AllTrophies {
		x, y, w, h := art.DisplaySlot(i)
		if !a.won(tr) {
			continue
		}
		col := art.Cyan
		if i >= len(game.Trophies) {
			col = art.Lime
		}
		fl := 0.85 + 0.15*math.Sin(t*3+float64(i))
		a.imgRect(dst, float64(x+1), float64(y+1), float64(w-2), float64(h-2), csCol(col, fl), false)
		if tr == a.br.sel || tr == a.br.hover {
			a.imgRect(dst, float64(x), float64(y), float64(w), float64(h), csCol(art.White, 0.5), true)
		}
	}
	dx, dy, dw, dh := art.DisplayRect()
	sy := float64(dy) + math.Mod(t*14, float64(dh))
	a.imgRect(dst, float64(dx), sy, float64(dw), 1, csCol(art.Cyan, 0.12), true)
	for _, s := range art.WindowStars {
		k := 0.5 + 0.5*math.Sin(t*s[2]+s[0])
		if k < 0.6 {
			continue
		}
		al := (k - 0.6) / 0.4
		a.imgRect(dst, s[0], s[1], 1, 1, csCol(art.White, al), true)
		if s[2] > 2.2 {
			a.imgRect(dst, s[0]-1, s[1], 3, 1, csCol(art.Silver, al*0.35), true)
			a.imgRect(dst, s[0], s[1]-1, 1, 3, csCol(art.Silver, al*0.35), true)
		}
	}
}

// drawJar draws a specimen jar: the liquid, the specimen if won and its
// bubbles, then the glass.
func (a *App) drawJar(dst *ebiten.Image, i int) {
	bl := a.as.Bridge
	t := game.Specimens[i]
	p := art.JarPos(i)
	won := a.won(t)
	j := bl.jar[0]
	if won {
		j = bl.jar[1]
	}
	hot := t == a.br.sel || t == a.br.hover
	if hot {
		al := 0.35
		if t == a.br.sel {
			al = 0.45 + 0.2*math.Sin(a.now*5)
		}
		a.put(dst, a.as.Glow["green"], p.X, p.Y, p.Z+art.JarBody*0.5, 1.1, 1.3, csAlpha(al), true)
	}
	a.blit(dst, j.back, p.X, p.Y, p.Z, j.ox, j.oy, 1, ebiten.ColorScale{}, false)
	if won {
		sp := bl.specimens[i]
		var cs ebiten.ColorScale
		cs.Scale(0.9, 1.1, 0.9, 1)
		bob := 0.6 * math.Sin(a.br.t*1.3+float64(i))
		a.blit(dst, sp.img, p.X, p.Y, p.Z+art.JarBase+1.5+bob, sp.ox, sp.oy, 1, cs, false)
		level := art.JarBase + art.JarBody*0.84
		for k := 0; k < 3; k++ {
			ph := math.Mod(a.br.t*0.45+float64(k)/3+float64(i)*0.37, 1)
			z := art.JarBase + 1 + ph*(level-art.JarBase-2)
			x := p.X + math.Sin(float64(k)*2.1+float64(i))*art.JarRX*0.5 + 0.8*math.Sin(a.br.t*4+float64(k))
			a.put(dst, a.as.White, x, p.Y+3, p.Z+z, 1.4, 1.4, csCol(art.Lime, 0.7*(1-ph*0.5)), false)
		}
	}
	a.blit(dst, j.front, p.X, p.Y, p.Z, j.ox, j.oy, 1, ebiten.ColorScale{}, false)
}

// drawPedestal draws the hologram over a pedestal: the trophy turning in
// cyan light when it is won, its dim outline when it is not.
func (a *App) drawPedestal(dst *ebiten.Image, i int) {
	bl := a.as.Bridge
	b := &a.br
	tr := game.Trophies[i]
	p := art.PedestalPos(i)
	t := b.t
	hot := tr == b.sel || tr == b.hover
	if !a.won(tr) {
		sp := bl.ghost[i]
		al := 0.75
		if hot {
			al = 1
		}
		a.blit(dst, sp.img, p.X, p.Y, pedZ, sp.ox, sp.oy, 1, csAlpha(al), false)
		return
	}
	pulse := 0.5 + 0.5*math.Sin(t*2.2+float64(i))
	a.put(dst, a.as.Glow["cyan"], p.X, p.Y, art.PedestalH, 0.5, 0.36, csAlpha(0.5+0.25*pulse), true)
	a.put(dst, a.as.Glow["cyan"], p.X, p.Y, art.PedestalH+16, 0.55, 1.15, csAlpha(0.1+0.04*pulse), true)
	k := angleIndex(t*holoTurn+float64(i)*0.7, pedAngles)
	sp := bl.holo[i][k]
	z := pedZ + 1.2*math.Sin(t*1.7+float64(i))
	fl := 0.84 + 0.16*math.Sin(t*21+float64(i)*3)
	a.blit(dst, sp.img, p.X, p.Y, z, sp.ox, sp.oy, 1, csAlpha(fl), false)
	if hot {
		a.blit(dst, sp.img, p.X, p.Y, z, sp.ox, sp.oy, 1, csAlpha(0.3), true)
	}
}

// drawHoloTable draws the cone of light over the holo-table and the big
// hologram turning in it, building up row by row when it changes.
func (a *App) drawHoloTable(dst *ebiten.Image) {
	b := &a.br
	tb := art.BridgeTable
	t := b.t
	model, ghost := a.tableModel()
	h := b.holos[model]
	ready := h != nil && h.holo != nil
	// The cone of light.
	fl := 0.85 + 0.15*math.Sin(t*17)
	var path vector.Path
	const n = 20
	for k := 0; k <= n; k++ {
		ang := math.Pi * float64(k) / n
		x, y := a.screenAt(tb.X+math.Cos(ang)*8, tb.Y+math.Sin(ang)*8, art.TableH)
		if k == 0 {
			path.MoveTo(float32(x), float32(y))
		} else {
			path.LineTo(float32(x), float32(y))
		}
	}
	for k := n; k >= 0; k-- {
		ang := math.Pi * float64(k) / n
		x, y := a.screenAt(tb.X+math.Cos(ang)*44, tb.Y-math.Sin(ang)*44*0.3, tableZ+tableBoxH*0.95)
		path.LineTo(float32(x), float32(y))
	}
	path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true, Blend: ebiten.BlendLighter}
	cone := 0.1
	if ghost {
		cone = 0.05
	}
	op.ColorScale.ScaleWithColor(pm(art.Cyan, cone*fl))
	vector.FillPath(dst, &path, &vector.FillOptions{}, op)
	a.put(dst, a.as.Glow["cyan"], tb.X, tb.Y, art.TableH, 1.2, 0.85, csAlpha(0.45*fl), true)
	// The disc of light the projection stands on.
	z := tableZ + 1.5*math.Sin(t*1.3)
	disc := 0.55 * fl
	if ghost {
		disc *= 0.5
	}
	a.put(dst, a.as.Glow["cyan"], tb.X, tb.Y, z, 2.2, 1.6, csAlpha(disc*0.45), true)
	rg := a.as.Ring(int(art.TableR)-14, art.Cyan)
	a.blit(dst, rg.img, tb.X, tb.Y, z, rg.ox, rg.oy, 1, csAlpha(disc), true)
	if !ready {
		// Still baking: a flicker of light over the emitter.
		a.put(dst, a.as.Glow["cyan"], tb.X, tb.Y, tableZ+20, 1.4, 1.6, csAlpha(0.25+0.15*math.Sin(t*13)), true)
		return
	}
	k := angleIndex(t*holoTurn, tableAngles)
	sp := h.holo[k]
	al := 0.92 * fl
	if ghost {
		sp = h.ghost[k]
		al = 0.8
	}
	// Build up from the bottom when the projection changes.
	rk := clampF(b.showT/revealTime, 0, 1)
	hgt := float64(sp.img.Bounds().Dy())
	cut := int(math.Round(hgt * (1 - easeOut(rk))))
	lift := sprite{img: sp.img, ox: sp.ox, oy: sp.oy + z*art.HZ}
	// A ghost is added to the light, or it would vanish over the bright table.
	a.blitRows(dst, sp.img, lift, tb.X, tb.Y, cut, sp.img.Bounds().Dy(), csAlpha(al), ghost)
	if !ghost {
		a.blitRows(dst, sp.img, lift, tb.X, tb.Y, cut, sp.img.Bounds().Dy(), csAlpha(0.18*fl), true)
	}
	if rk < 1 {
		w := float64(sp.img.Bounds().Dx()) * 0.5 * a.cam.Z
		sx, sy := a.screenAt(tb.X, tb.Y, z)
		sy += (float64(cut) - lift.oy + z*art.HZ) * a.cam.Z
		lw := float32(math.Max(1, a.cam.Z))
		vector.StrokeLine(dst, float32(sx-w), float32(sy), float32(sx+w), float32(sy), lw*2, pm(art.White, 0.7*(1-rk)), true)
	}
}

// ---- The panel ----

// drawBridgeOverview is the panel without a picked trophy: the tally and
// the list of trophies and specimens.
func (a *App) drawBridgeOverview(dst *ebiten.Image, cl colonyLayout) {
	u := a.u
	r := cl.panel
	x, w := r.X+8*u, r.W-16*u
	y := r.Y + 8*u
	a.drawText(dst, "Vanguard", x, y, 2*u, art.White, 1)
	y += 18 * u
	a.drawText(dst, "The bridge of your flagship", x, y, u, art.Gray, 1)
	y += 12 * u
	nt, ns := a.save.trophyCount(game.Trophies), a.save.trophyCount(game.Specimens)
	a.drawIcon(dst, "trophy", x, y-2*u, u, 1)
	a.drawSegs(dst, []Seg{{fmt.Sprintf("Trophies %d/%d", nt, len(game.Trophies)), art.White},
		{fmt.Sprintf("  · Specimens %d/%d", ns, len(game.Specimens)), art.Silver}}, x+14*u, y, u, 1)
	y += 12 * u
	total := 0
	for _, t := range game.AllTrophies {
		total += t.Reward
	}
	a.drawSegsIcons(dst, []Seg{{"Crystals paid: ", art.Silver}, {"◆", art.Pink}, {fmt.Sprintf(" %d/%d", a.trophyEarned(), total), art.Pink}}, x, y, u, 1)
	nT := len(game.Trophies)
	for i, btn := range cl.rows[:nT] {
		a.drawTrophyRow(dst, btn, game.Trophies[i])
	}
	if len(cl.rows) > nT {
		jy := cl.rows[nT].R.Y
		a.drawText(dst, "Specimens", x, jy-11*u, u, art.Gray, 1)
		for i, btn := range cl.rows[nT:] {
			a.drawJarButton(dst, btn, game.Specimens[i])
		}
		hy := jy + 22*u + 8*u
		lines := a.wrap("Trophies pay their crystals once. Pick one to see it on the holo-table.", w, u)
		if hy+len(lines)*11*u <= r.Y+r.H-6*u {
			for _, l := range lines {
				a.drawText(dst, l, x, hy, u, art.Gray, 1)
				hy += 11 * u
			}
		}
	}
}

// drawTrophyRow is one trophy in the overview list.
func (a *App) drawTrophyRow(dst *ebiten.Image, btn Button, t *game.Trophy) {
	u := a.u
	r := btn.R
	bg := pm(art.Navy, 0.5)
	if a.hoverBtn == btn.ID || t == a.br.hover {
		bg = pm(art.Slate, 0.7)
	}
	fillRect(dst, r.X, r.Y, r.W, r.H, bg)
	ty := r.Y + (r.H-7*u)/2
	iy := r.Y + (r.H-11*u)/2
	won := a.won(t)
	col := art.Silver
	if won {
		col = art.White
	}
	a.drawText(dst, t.Name, r.X+4*u, ty, u, col, 1)
	rx := r.X + r.W - 4*u
	if won {
		rx -= 11 * u
		a.drawIcon(dst, "check", rx, iy, u, 1)
		return
	}
	s := fmt.Sprint(t.Reward)
	rx -= a.textWidth(s, u)
	a.drawText(dst, s, rx, ty, u, art.Pink, 1)
	rx -= 13 * u
	a.drawIcon(dst, "crystal", rx, iy, u, 1)
	if t.Counter() {
		have, goal := t.Progress(a.save.record())
		p := fmt.Sprintf("%d/%d", have, goal)
		rx -= 5*u + a.textWidth(p, u)
		a.drawText(dst, p, rx, ty, u, art.Gray, 1)
	}
}

// drawJarButton is one specimen in the overview: the enemy, lit once won,
// with a bar of how many have been destroyed.
func (a *App) drawJarButton(dst *ebiten.Image, btn Button, t *game.Trophy) {
	u := a.u
	r := btn.R
	won := a.won(t)
	bg := pm(art.Navy, 0.5)
	switch {
	case t == a.br.sel:
		bg = pm(art.Slate, 0.95)
	case a.hoverBtn == btn.ID || t == a.br.hover:
		bg = pm(art.Slate, 0.7)
	}
	fillRect(dst, r.X, r.Y, r.W, r.H, bg)
	if won {
		fillRect(dst, r.X, r.Y+r.H-u, r.W, u, pm(art.Lime, 1))
	}
	lk := a.as.Looks[t.Enemy]
	img := lk.icon.img
	cs := ebiten.ColorScale{}
	if !won {
		img = lk.icon.sil
		cs = csCol(art.Slate, 1)
	}
	ib := img.Bounds()
	op := &ebiten.DrawImageOptions{ColorScale: cs}
	op.GeoM.Scale(float64(u), float64(u))
	op.GeoM.Translate(float64(r.X+(r.W-ib.Dx()*u)/2), float64(r.Y+3*u+(14*u-ib.Dy()*u)/2))
	dst.DrawImage(img, op)
	if !won {
		have, goal := t.Progress(a.save.record())
		bw := r.W - 4*u
		fillRect(dst, r.X+2*u, r.Y+r.H-4*u, bw, 2*u, pm(art.Ink, 1))
		if fw := bw * have / max(1, goal); fw > 0 {
			fillRect(dst, r.X+2*u, r.Y+r.H-4*u, fw, 2*u, pm(art.Green, 1))
		}
	}
}

// drawBridgeDetail is the panel of the picked trophy.
func (a *App) drawBridgeDetail(dst *ebiten.Image, cl colonyLayout) {
	u := a.u
	r := cl.panel
	t := a.br.sel
	x, w := r.X+8*u, r.W-16*u
	a.drawButton(dst, cl.all, 1)
	y := cl.all.R.Y + cl.all.R.H + 9*u
	a.drawText(dst, t.Name, x, y, 2*u, art.White, 1)
	y += 18 * u
	won := a.won(t)
	kind := "Trophy"
	if t.Kind == game.TrSpecimen {
		kind = "Specimen"
	}
	if won {
		a.drawSegs(dst, []Seg{{kind + " · ", art.Gray}, {"Won", art.Lime}}, x, y, u, 1)
	} else {
		a.drawSegs(dst, []Seg{{kind + " · ", art.Gray}, {"Not won yet", art.Amber}}, x, y, u, 1)
	}
	y += 17 * u
	for _, l := range a.wrap(t.Desc, w, u) {
		a.drawText(dst, l, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	if t.Counter() {
		y += 6 * u
		have, goal := t.Progress(a.save.record())
		a.drawText(dst, "Progress", x, y, u, art.Gray, 1)
		p := fmt.Sprintf("%d/%d", have, goal)
		a.drawText(dst, p, x+w-a.textWidth(p, u), y, u, art.White, 1)
		y += 11 * u
		fillRect(dst, x, y, w, 5*u, pm(art.Navy, 1))
		if fw := w * have / max(1, goal); fw > 0 {
			col := art.Blue
			if won {
				col = art.Green
			}
			fillRect(dst, x, y, fw, 5*u, pm(col, 1))
			fillRect(dst, x, y, fw, u, pm(art.Cyan, 1))
		}
		y += 10 * u
	}
	y += 8 * u
	a.drawText(dst, "Reward", x, y, u, art.Gray, 1)
	y += 13 * u
	if won {
		a.drawSegsIcons(dst, []Seg{{"◆", art.Pink}, {fmt.Sprintf(" %d crystals", t.Reward), art.White}, {" · paid", art.Lime}}, x, y, u, 1)
	} else {
		a.drawSegsIcons(dst, []Seg{{"◆", art.Pink}, {fmt.Sprintf(" %d crystals, paid once", t.Reward), art.Pink}}, x, y, u, 1)
	}
	y += 20 * u
	msg := "On the holo-table now"
	if !won {
		msg = "Its outline is on the holo-table"
	}
	a.drawText(dst, msg, x, y, u, art.Cyan, 0.8+0.2*math.Sin(a.now*3))
}

// drawBridgeTip names the trophy (or the table) under the cursor.
func (a *App) drawBridgeTip(dst *ebiten.Image, v Rect) {
	u := a.u
	b := &a.br
	var lines [][]Seg
	var x0, y0, x1 float64
	switch {
	case b.hover != nil:
		t := b.hover
		lines = append(lines, []Seg{{t.Name, art.White}})
		switch {
		case a.won(t):
			lines = append(lines, []Seg{{"Won · ", art.Lime}, {"◆", art.Pink}, {fmt.Sprintf("%d paid", t.Reward), art.Gray}})
		case t.Counter():
			have, goal := t.Progress(a.save.record())
			lines = append(lines, []Seg{{fmt.Sprintf("%d/%d · ", have, goal), art.Silver}, {"◆", art.Pink}, {fmt.Sprint(t.Reward), art.Pink}})
		default:
			lines = append(lines, []Seg{{"Not won yet · ", art.Silver}, {"◆", art.Pink}, {fmt.Sprint(t.Reward), art.Pink}})
		}
		if i := trophyIndex(t); i < len(game.Trophies) {
			x0, y0, x1, _ = a.pedestalBox(i)
		} else {
			x0, y0, x1, _ = a.jarBox(i - len(game.Trophies))
		}
	case b.table:
		lines = append(lines, []Seg{{"Holo-table", art.White}}, []Seg{{"Showing: " + a.tableName(), art.Cyan}})
		if b.sel == nil {
			lines = append(lines, []Seg{{"Pick a trophy to see it here", art.Gray}})
		}
		x0, y0, x1, _ = a.tableBox()
	default:
		return
	}
	pw := 0
	for _, l := range lines {
		pw = max(pw, a.segsWidthIcons(l, u))
	}
	pw += 8 * u
	ph := len(lines)*11*u + 5*u
	sx, sy := a.cam.ToScreen((x0+x1)/2, y0)
	x := max(v.X+2*u, min(int(sx)-pw/2, v.X+v.W-pw-2*u))
	y := max(v.Y+2*u, int(sy)-ph-4*u)
	a.panel(dst, Rect{x, y, pw, ph}, 1)
	for i, l := range lines {
		a.drawSegsIcons(dst, l, x+4*u, y+4*u+i*11*u, u, 1)
	}
}

var _ = image.Rect
