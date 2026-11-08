package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"outpost/internal/art"
	"outpost/internal/game"
)

// A column is a centred panel of stacked rows: the pause menu, the result
// screen and the settings.
type rowKind int

const (
	rowText rowKind = iota
	rowSegs
	rowStars
	rowButton
	rowPair // two buttons side by side
	rowGap
)

type row struct {
	kind rowKind
	text string
	segs []Seg
	sc   int
	col  color.RGBA
	btn  Button
	btn2 Button // the right button of a pair
	h, y int
}

type column struct {
	panel Rect
	rows  []row
}

func (a *App) textRow(s string, sc int, col color.RGBA) row {
	return row{kind: rowText, text: s, sc: sc, col: col, h: 7*sc + 4*a.u}
}

func (a *App) segRow(segs []Seg) row { return row{kind: rowSegs, segs: segs, h: 11 * a.u} }
func (a *App) gapRow(units int) row  { return row{kind: rowGap, h: units * a.u} }
func (a *App) starsRow() row         { return row{kind: rowStars, h: 42 * a.u} }

func (a *App) btnRow(b Button) row {
	b.Enabled = true
	return row{kind: rowButton, btn: b, h: 16 * a.u}
}

// pairRows puts two buttons side by side when their labels fit in the
// width of one, else one under the other.
func (a *App) pairRows(b1, b2 Button, btnW int) []row {
	gap := 4 * a.u
	if half := (btnW - gap) / 2; a.buttonWidth(b1) > half || a.buttonWidth(b2) > half {
		return []row{a.btnRow(b1), a.btnRow(b2)}
	}
	b1.Enabled, b2.Enabled = true, true
	return []row{{kind: rowPair, btn: b1, btn2: b2, h: 16 * a.u}}
}

func (a *App) rowWidth(r row) int {
	switch r.kind {
	case rowText:
		return a.textWidth(r.text, r.sc)
	case rowSegs:
		return a.segsWidthIcons(r.segs, a.u)
	case rowStars:
		return 3*33*a.u + 2*10*a.u
	}
	return 0
}

// layoutColumn stacks the rows in a panel centred in the view.
func (a *App) layoutColumn(rows []row, minW, btnW int) column {
	u := a.u
	pw, h := max(minW, btnW+24*u), 10*u
	for _, r := range rows {
		pw = max(pw, a.rowWidth(r)+20*u)
		h += r.h
	}
	h += 7 * u
	top := 0
	if a.scr == scrGame {
		top = a.barH
	}
	y := max(top+2*u, top+(a.sh-top)/2-h/2)
	p := Rect{a.sw/2 - pw/2, y, pw, h}
	y = p.Y + 10*u
	for i := range rows {
		rows[i].y = y
		switch rows[i].kind {
		case rowButton:
			rows[i].btn.R = Rect{a.sw/2 - btnW/2, y, btnW, a.btnHeight()}
		case rowPair:
			gap := 4 * u
			half := (btnW - gap) / 2
			rows[i].btn.R = Rect{a.sw/2 - btnW/2, y, half, a.btnHeight()}
			rows[i].btn2.R = Rect{a.sw/2 - btnW/2 + half + gap, y, btnW - half - gap, a.btnHeight()}
		}
		y += rows[i].h
	}
	return column{p, rows}
}

func (a *App) columnButtons(c column) {
	for _, r := range c.rows {
		switch r.kind {
		case rowButton:
			a.addButton(r.btn)
		case rowPair:
			a.addButton(r.btn)
			a.addButton(r.btn2)
		}
	}
}

func (a *App) drawColumn(dst *ebiten.Image, c column, al float64) {
	u := a.u
	a.panel(dst, c.panel, al)
	for _, r := range c.rows {
		switch r.kind {
		case rowText:
			a.drawText(dst, r.text, a.sw/2-a.textWidth(r.text, r.sc)/2, r.y, r.sc, r.col, al)
		case rowSegs:
			a.drawSegsIcons(dst, r.segs, a.sw/2-a.segsWidthIcons(r.segs, u)/2, r.y, u, al)
		case rowButton:
			a.drawButton(dst, r.btn, al)
		case rowPair:
			a.drawButton(dst, r.btn, al)
			a.drawButton(dst, r.btn2, al)
		case rowStars:
			a.drawResultStars(dst, r.y, al)
		}
	}
}

// levelTitle is "1-2 · Dry Canyon".
func levelTitle(s *game.LevelSpec) string { return levelLabel(s) + " · " + s.Name }

// starRules explains the three stars.
func starRules() []Seg {
	g := art.Gray
	return []Seg{{"★", g}, {" Win  ·  ", g}, {"★", g}, {"★", g}, {" Base 50%+  ·  ", g}, {"★", g}, {"★", g}, {"★", g}, {" Base 90%+", g}}
}

// payNote tells what the crystals of a win were paid for.
func payNote(p game.Payout) string {
	stars := func(n int) string {
		if n == 1 {
			return "1 new star"
		}
		return fmt.Sprintf("%d new stars", n)
	}
	switch {
	case p.New > 0 && p.Again > 0:
		return stars(p.New) + " and a replay bonus"
	case p.New > 0:
		return stars(p.New)
	}
	return "replay bonus"
}

// trophyRows lists the trophies the battle has won (two at most, then how
// many more).
func (a *App) trophyRows() []row {
	ts := a.res.trophies
	if len(ts) == 0 {
		return nil
	}
	rows := []row{a.gapRow(3)}
	show := ts
	if len(ts) > 2 {
		show = ts[:1]
	}
	for _, t := range show {
		kind := "Trophy: "
		if t.Kind == game.TrSpecimen {
			kind = "Specimen: "
		}
		rows = append(rows, a.segRow([]Seg{{"{trophy}", art.Amber}, {" " + kind + t.Name + "  ", art.Amber}, {"◆", art.Pink}, {fmt.Sprintf(" +%d", t.Reward), art.Pink}}))
	}
	if len(ts) > 2 {
		pay := 0
		for _, t := range ts[1:] {
			pay += t.Reward
		}
		rows = append(rows, a.segRow([]Seg{{"{trophy}", art.Amber}, {fmt.Sprintf(" and %d more trophies  ", len(ts)-1), art.Amber}, {"◆", art.Pink}, {fmt.Sprintf(" +%d", pay), art.Pink}}))
	}
	return rows
}

var defeatTips = []string{
	"Upgrade the turrets that see the most of the road before you build new ones.",
	"Cryo slows enemies down, so every turret around it gets more shots.",
	"Railguns ignore armor: they are the answer to Walkers and the Hive Queen.",
	"Repair Drones fly: only Skyguard turrets can shoot them down.",
	"Flamers burn whole crowds of Swarmlings at once.",
}

// overlayColumn is the pause menu or the result screen, if one is up.
func (a *App) overlayColumn() (column, bool) {
	u := a.u
	w := a.world
	big := 2 * u
	s := a.spec
	var rows []row
	switch {
	case a.paused && w.Active():
		rows = append(rows,
			a.textRow("Paused", big, art.White),
			a.textRow(levelTitle(s), u, art.Gray),
			a.gapRow(6),
			a.btnRow(Button{ID: "resume", Label: "Resume", Key: "Esc", Primary: true}),
			a.btnRow(Button{ID: "restart", Label: "Restart", Key: "R"}),
			a.btnRow(Button{ID: "guide", Label: "Guide", Key: "I"}),
			a.btnRow(Button{ID: "settings", Label: "Settings"}),
			a.btnRow(Button{ID: "levels", Label: "Level select", Key: "M"}),
			a.btnRow(Button{ID: "quit", Label: "Quit game", Key: "Q"}),
		)
	case w.Phase == game.PhaseWon:
		rows = append(rows,
			a.textRow("Victory!", big, art.Lime),
			a.textRow(levelTitle(s), u, art.Silver),
			a.gapRow(2),
			a.starsRow(),
		)
		c, idx := chapterOf(s)
		switch {
		case a.res.unlocked != nil:
			if nc, ni := chapterOf(a.res.unlocked); nc != nil && nc != c && ni == 0 {
				rows = append(rows,
					a.textRow("Chapter complete!", u, art.Lime),
					a.textRow(fmt.Sprintf("New chapter unlocked: %d · %s", nc.Num, nc.Name), u, art.Cyan))
				break
			}
			rows = append(rows, a.textRow("New level unlocked: "+levelTitle(a.res.unlocked), u, art.Cyan))
		case a.res.prev == 0 && idx >= 0 && nextLevel(s) == nil:
			msg := "Chapter complete!"
			switch {
			case c.Soon > 0:
				msg = "More levels of this chapter are coming soon."
			case c.Num == len(game.Chapters):
				msg += " More chapters are coming soon."
			}
			rows = append(rows, a.textRow(msg, u, art.Cyan))
		}
		if b := a.res.bonus; b != nil {
			rows = append(rows, a.textRow("Skirmish unlocked: "+b.Name+", every turret at once", u, art.Pink))
		}
		if a.res.prev > 0 && a.res.stars > a.res.prev {
			rows = append(rows, a.textRow("New best!", u, art.Amber))
		}
		if isExpedition(s) {
			rows = append(rows, a.textRow(fmt.Sprintf("Expeditions won: %d", a.save.Exp.Won), u, art.Cyan))
		}
		if p := a.res.pay; p.Total > 0 {
			note := payNote(p)
			if isExpedition(s) {
				note = expPayNote(s)
			}
			rows = append(rows, a.gapRow(2), a.segRow([]Seg{{"◆", art.Pink}, {fmt.Sprintf(" +%d crystals", p.Total), art.Pink}, {"  · " + note, art.Gray}}))
		}
		rows = append(rows, a.trophyRows()...)
		colony := false
		if l := a.res.colonized; l != nil {
			if l.Soon {
				rows = append(rows, a.textRow(l.Name+" is free! Its colony comes in a later version.", u, art.Cyan))
			} else {
				rows = append(rows, a.textRow(l.Name+" is free! Build its colony with your crystals.", u, art.Lime))
				colony = true
			}
		}
		rows = append(rows,
			a.gapRow(3),
			a.textRow(fmt.Sprintf("Enemies destroyed: %d", w.Kills), u, art.Silver),
			a.textRow("Base integrity: "+pctText(w.BaseHP, w.BaseMaxHP), u, art.Silver),
		)
		if a.res.stars < 3 {
			rows = append(rows, a.gapRow(3), a.segRow(starRules()))
		}
		rows = append(rows, a.gapRow(6))
		if isExpedition(s) {
			rows = append(rows, a.btnRow(Button{ID: "expnext", Label: "Next expedition", Key: "Enter", Primary: true}))
		} else if nx := a.nextPlayable(); nx != nil {
			label := "Next level"
			if nx == game.Skirmish {
				label = "Play Skirmish"
			}
			rows = append(rows,
				a.btnRow(Button{ID: "next", Label: label, Key: "Enter", Primary: true}),
				a.btnRow(Button{ID: "restart", Label: "Replay", Key: "R"}),
			)
		} else {
			rows = append(rows, a.btnRow(Button{ID: "restart", Label: "Play again", Key: "R", Primary: true}))
		}
		colBtn := Button{ID: "colony", Label: "Colonies", Key: "C"}
		bridgeBtn := Button{ID: "bridge", Label: "Bridge", Key: "B"}
		switch {
		case colony && len(a.res.trophies) > 0:
			rows = append(rows, a.pairRows(colBtn, bridgeBtn, 120*u)...)
		case colony:
			rows = append(rows, a.btnRow(colBtn))
		case len(a.res.trophies) > 0:
			rows = append(rows, a.btnRow(bridgeBtn))
		}
		rows = append(rows, a.btnRow(Button{ID: "levels", Label: "Level select", Key: "Esc"}))
	case w.Phase == game.PhaseLost:
		rows = append(rows,
			a.textRow("Base destroyed", big, art.Red),
			a.textRow(levelTitle(s), u, art.Silver),
			a.gapRow(4),
			a.textRow(fmt.Sprintf("The swarm broke through on wave %d of %d", w.Wave, w.TotalWaves()), u, art.White),
		)
		rows = append(rows, a.trophyRows()...)
		rows = append(rows, a.gapRow(6))
		tip := defeatTips[a.runs%len(defeatTips)]
		switch {
		case s.Tip != "":
			tip = s.Tip
		case s.Lesson != nil:
			tip = s.Lesson.Solution
		}
		for i, l := range a.wrap("Tip: "+tip, 220*u, u) {
			if i == 0 {
				rows = append(rows, a.segRow([]Seg{{"Tip:", art.Amber}, {l[4:], art.Silver}}))
				continue
			}
			rows = append(rows, a.textRow(l, u, art.Silver))
		}
		rows = append(rows,
			a.gapRow(8),
			a.btnRow(Button{ID: "restart", Label: "Retry", Key: "Enter", Primary: true}),
		)
		if isExpedition(s) {
			rows = append(rows, a.btnRow(Button{ID: "expnew", Label: "New expedition", Key: "N"}))
		}
		if len(a.res.trophies) > 0 {
			rows = append(rows, a.btnRow(Button{ID: "bridge", Label: "Bridge", Key: "B"}))
		}
		rows = append(rows, a.btnRow(Button{ID: "levels", Label: "Level select", Key: "Esc"}))
	default:
		return column{}, false
	}
	return a.layoutColumn(rows, 170*u, 120*u), true
}

func (a *App) layoutOverlay() {
	if !a.world.Active() && a.endAlpha() < 0.6 {
		return
	}
	c, ok := a.overlayColumn()
	if !ok {
		return
	}
	a.columnButtons(c)
	a.blockers = append(a.blockers, Rect{0, 0, a.sw, a.sh})
}

func (a *App) drawOverlay(dst *ebiten.Image) {
	if a.popup != panelNone {
		return
	}
	c, ok := a.overlayColumn()
	if !ok {
		return
	}
	al := 1.0
	if !a.world.Active() {
		al = a.endAlpha()
		if al <= 0 {
			return
		}
		a.dim(dst, al*0.8)
	} else {
		a.dim(dst, 1)
	}
	a.drawColumn(dst, c, al)
}

// Stars pop in one after another on the victory screen.
const (
	starsFrom = 1.3
	starsStep = 0.35
)

func (a *App) drawResultStars(dst *ebiten.Image, y int, al float64) {
	u := a.u
	sz, gap := 33*u, 10*u
	x := a.sw/2 - (3*sz+2*gap)/2
	for i := 0; i < 3; i++ {
		cx, cy := float64(x+i*(sz+gap)+sz/2), float64(y+sz/2+3*u)
		t := a.endT - (starsFrom + float64(i)*starsStep)
		earned := i < a.res.stars && t >= 0
		op := &ebiten.DrawImageOptions{}
		s := float64(3 * u)
		img := a.as.Icons["star0"]
		if earned {
			img = a.as.Icons["star"]
			s *= easeOutBack(t / 0.3)
		}
		op.GeoM.Translate(-5.5, -5.5)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(cx, cy)
		op.ColorScale = csAlpha(al)
		dst.DrawImage(img, op)
		if earned && t < 0.7 {
			gop := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter, Filter: ebiten.FilterLinear}
			k := float64(sz) * (1.5 + 1.5*t) / 64
			gop.GeoM.Translate(-32, -32)
			gop.GeoM.Scale(k, k)
			gop.GeoM.Translate(cx, cy)
			gop.ColorScale = csAlpha(al * 0.9 * (1 - t/0.7))
			dst.DrawImage(a.as.SoftGlow, gop)
		}
	}
}

// ---- Windows over the screens: guide and settings ----

func (a *App) openPanel(p panelKind) {
	a.popup = p
	a.resetT = 0
}

func (a *App) closePanel() { a.popup = panelNone }

func (a *App) panelKeys() {
	jp := inpututil.IsKeyJustPressed
	if jp(ebiten.KeyEscape) || (a.popup == panelGuide && jp(ebiten.KeyI)) {
		a.closePanel()
		return
	}
	if a.popup != panelGuide {
		return
	}
	nt := len(guideTabs)
	if jp(ebiten.KeyArrowLeft) {
		a.tab = (a.tab + nt - 1) % nt
	}
	if jp(ebiten.KeyArrowRight) || jp(ebiten.KeyTab) {
		a.tab = (a.tab + 1) % nt
	}
	if n := len(a.guideEntries()); n > 0 {
		sel := min(a.tabSel[a.tab], n-1)
		if jp(ebiten.KeyArrowUp) {
			sel = (sel + n - 1) % n
		}
		if jp(ebiten.KeyArrowDown) {
			sel = (sel + 1) % n
		}
		a.tabSel[a.tab] = sel
	}
}

func (a *App) layoutPanel() {
	switch a.popup {
	case panelGuide:
		gl := a.guideLayout()
		a.buttons = append(a.buttons, gl.tabs...)
		a.buttons = append(a.buttons, gl.entries...)
		a.addButton(gl.close)
	case panelSettings:
		a.columnButtons(a.settingsColumn())
	}
}

func (a *App) drawPanel(dst *ebiten.Image) {
	a.dim(dst, 1)
	switch a.popup {
	case panelGuide:
		a.drawGuide(dst)
	case panelSettings:
		a.drawColumn(dst, a.settingsColumn(), 1)
	}
}

func onOff(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}

const resetConfirmTime = 3.0

func (a *App) settingsColumn() column {
	u := a.u
	reset := Button{ID: "reset", Label: "Reset progress"}
	switch {
	case a.resetT > 0 && a.now-a.resetT < resetConfirmTime:
		reset.Label, reset.Danger = "Click again to erase all stars", true
	case a.resetT < 0 && a.now+a.resetT < 2:
		reset.Label = "Progress erased"
	}
	rows := []row{
		a.textRow("Settings", 2*u, art.White),
		a.gapRow(6),
		a.btnRow(Button{ID: "fullscreen", Label: "Fullscreen: " + onOff(ebiten.IsFullscreen()), Key: "F11"}),
		a.btnRow(Button{ID: "autopause", Label: "Pause when the window is inactive: " + onOff(!a.save.NoAutoPause)}),
		a.btnRow(reset),
		a.gapRow(2),
		a.textRow("Progress is saved automatically.", u, art.Gray),
		a.gapRow(6),
		a.btnRow(Button{ID: "close", Label: "Back", Key: "Esc", Primary: true}),
	}
	bw := 0
	for _, r := range rows {
		if r.kind == rowButton {
			bw = max(bw, a.buttonWidth(r.btn))
		}
	}
	return a.layoutColumn(rows, 0, max(bw, 150*u))
}

// resetProgress erases the stars and met enemies after a second click.
func (a *App) resetProgress() {
	if a.resetT <= 0 || a.now-a.resetT > resetConfirmTime {
		a.resetT = a.now
		return
	}
	a.save.reset()
	clear(a.seen)
	a.pick = 0
	a.resetT = -a.now // shows "Progress erased" for a moment
}

// ---- Guide ----

var guideTabs = []string{"Controls", "Turrets", "Enemies"}

type guideEntry struct {
	tower *game.TowerDef
	enemy *game.EnemyDef
	known bool
}

// guideEntries lists the turrets of this level or every enemy type.
func (a *App) guideEntries() []guideEntry {
	var out []guideEntry
	switch a.tab {
	case 1:
		for _, d := range a.spec.Arsenal() {
			out = append(out, guideEntry{tower: d, known: true})
		}
	case 2:
		for _, d := range game.EnemyOrder {
			out = append(out, guideEntry{enemy: d, known: a.seen[d]})
		}
	}
	return out
}

type guideLayout struct {
	panel   Rect
	tabs    []Button
	close   Button
	entries []Button
	body    Rect
}

func (a *App) guideLayout() guideLayout {
	u := a.u
	pw := min(a.sw-12*u, 470*u)
	ph := min(a.sh-12*u, 300*u)
	p := Rect{a.sw/2 - pw/2, a.sh/2 - ph/2, pw, ph}
	gl := guideLayout{panel: p}
	y := p.Y + 7*u
	x := p.X + 10*u + a.textWidth("Guide", 2*u) + 14*u
	for i, t := range guideTabs {
		b := Button{ID: fmt.Sprintf("tab:%d", i), Label: t, Enabled: true, Primary: i == a.tab}
		b.R = Rect{x, y, a.buttonWidth(b), a.btnHeight()}
		x += b.R.W + 3*u
		gl.tabs = append(gl.tabs, b)
	}
	cb := Button{ID: "close", Label: "Close", Key: "Esc", Enabled: true}
	cw := a.buttonWidth(cb)
	cb.R = Rect{p.X + pw - cw - 8*u, y, cw, a.btnHeight()}
	gl.close = cb
	top := y + a.btnHeight() + 9*u
	body := Rect{p.X + 10*u, top, pw - 20*u, p.Y + ph - 8*u - top}
	if a.tab == 0 {
		gl.body = body
		return gl
	}
	lw := 100 * u
	for i := range a.guideEntries() {
		gl.entries = append(gl.entries, Button{ID: fmt.Sprintf("entry:%d", i), R: Rect{body.X, body.Y + i*26*u, lw, 24 * u}, Enabled: true})
	}
	gl.body = Rect{body.X + lw + 12*u, body.Y, body.W - lw - 12*u, body.H}
	return gl
}

func (a *App) drawGuide(dst *ebiten.Image) {
	u := a.u
	gl := a.guideLayout()
	p := gl.panel
	fillRect(dst, p.X, p.Y, p.W, p.H, art.Ink) // the guide is text-heavy: an opaque backing keeps it readable
	a.panel(dst, p, 1)
	a.drawText(dst, "Guide", p.X+10*u, p.Y+7*u+(a.btnHeight()-14*u)/2, 2*u, art.White, 1)
	for _, b := range gl.tabs {
		a.drawButton(dst, b, 1)
	}
	a.drawButton(dst, gl.close, 1)
	fillRect(dst, p.X+6*u, gl.body.Y-5*u, p.W-12*u, u, pm(art.Navy, 1))
	if a.tab == 0 {
		a.drawControls(dst, gl.body)
		return
	}
	entries := a.guideEntries()
	if len(entries) == 0 {
		return
	}
	sel := min(a.tabSel[a.tab], len(entries)-1)
	for i, b := range gl.entries {
		a.drawGuideEntry(dst, b, entries[i], i == sel)
	}
	e := entries[sel]
	switch {
	case e.tower != nil:
		a.drawTowerInfo(dst, e.tower, gl.body)
	case e.known:
		a.drawEnemyInfo(dst, e.enemy, gl.body)
	default:
		a.drawUnknownEnemy(dst, e.enemy, gl.body)
	}
	if a.tab == 1 && len(entries) < len(game.TowerOrder) {
		a.drawText(dst, "More turrets unlock", gl.body.X-112*u, gl.body.Y+gl.body.H-20*u, u, art.Gray, 1)
		a.drawText(dst, "as you progress.", gl.body.X-112*u, gl.body.Y+gl.body.H-9*u, u, art.Gray, 1)
	}
}

func (a *App) drawControls(dst *ebiten.Image, r Rect) {
	u := a.u
	list := a.controls()
	kw, dw := 0, 0
	for _, c := range list {
		kw = max(kw, a.textWidth(c[0], u))
		dw = max(dw, a.textWidth(c[1], u))
	}
	x := r.X + (r.W-(kw+12*u+dw))/2
	y := r.Y + max(0, (r.H-len(list)*12*u)/2)
	for _, c := range list {
		a.drawText(dst, c[0], x, y, u, art.Cyan, 1)
		a.drawText(dst, c[1], x+kw+12*u, y, u, art.Silver, 1)
		y += 12 * u
	}
}

// drawGuideEntry draws one line of the list on the left of the guide.
func (a *App) drawGuideEntry(dst *ebiten.Image, b Button, e guideEntry, sel bool) {
	u := a.u
	r := b.R
	hover := a.hoverBtn == b.ID
	switch {
	case sel:
		fillRect(dst, r.X, r.Y, r.W, r.H, pm(art.Slate, 0.9))
		fillRect(dst, r.X, r.Y, 2*u, r.H, art.Cyan)
	case hover:
		fillRect(dst, r.X, r.Y, r.W, r.H, pm(art.Navy, 0.9))
	}
	name := "???"
	col := art.Gray
	var img *ebiten.Image
	cs := ebiten.ColorScale{}
	if e.tower != nil {
		img = a.as.Towers[e.tower].icon.img
		name, col = e.tower.Name, art.White
	} else {
		lk := a.as.Looks[e.enemy]
		img = lk.icon.img
		if e.known {
			name, col = e.enemy.Name, art.White
		} else {
			img = lk.icon.sil
			cs = csCol(art.Steel, 1)
		}
	}
	ib := img.Bounds()
	op := &ebiten.DrawImageOptions{ColorScale: cs}
	op.GeoM.Scale(float64(u), float64(u))
	op.GeoM.Translate(float64(r.X+5*u+(26*u-ib.Dx()*u)/2), float64(r.Y+(r.H-ib.Dy()*u)/2))
	dst.DrawImage(img, op)
	a.drawText(dst, name, r.X+35*u, r.Y+(r.H-7*u)/2, u, col, 1)
}

// infoHead draws a portrait with a name and up to two tag lines beside it and
// returns the y below it.
func (a *App) infoHead(dst *ebiten.Image, r Rect, img *ebiten.Image, cs ebiten.ColorScale, name string, nameCol color.RGBA, tags ...Seg) int {
	u := a.u
	pr := Rect{r.X, r.Y, 52 * u, 42 * u}
	a.drawPortrait(dst, pr, img, cs, 1)
	tx := pr.X + pr.W + 9*u
	a.drawText(dst, name, tx, r.Y+4*u, 2*u, nameCol, 1)
	y := r.Y + 4*u + 14*u + 6*u
	for _, t := range tags {
		if t.S == "" {
			continue
		}
		a.drawSegsIcons(dst, splitIcons(t), tx, y, u, 1)
		y += 11 * u
	}
	return pr.Y + pr.H + 7*u
}

// splitIcons turns a tag with a leading "@" into an icon and text.
func splitIcons(t Seg) []Seg {
	if len(t.S) > 1 && t.S[0] == '@' {
		return []Seg{{"@", t.Col}, {t.S[1:], t.Col}}
	}
	return []Seg{t}
}

func (a *App) drawTowerInfo(dst *ebiten.Image, d *game.TowerDef, r Rect) {
	u := a.u
	lk := a.as.Towers[d]
	y := a.infoHead(dst, r, lk.card.img, ebiten.ColorScale{}, d.Name, art.White,
		Seg{d.Role, lk.accent}, Seg{fmt.Sprintf("@%d", d.Cost()), art.Yellow})
	x := r.X
	for _, l := range a.wrap(d.Desc, r.W, u) {
		a.drawText(dst, l, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 5 * u
	var table [][][]Seg
	for i, lv := range d.Levels {
		cells := [][]Seg{{{fmt.Sprintf("Lv %d", i+1), art.Cyan}}}
		cells = append(cells, statCells(towerStatSegs(d, lv, art.Gray, art.White))...)
		cells = append(cells, []Seg{{"@", art.Yellow}, {fmt.Sprint(lv.Cost), art.Yellow}})
		table = append(table, cells)
	}
	y = a.drawTable(dst, table, x, y, 8*u)
	if a.spec.NoSuper {
		a.drawText(dst, "◆ Super upgrade: unlocks later in the campaign", x, y, u, art.Gray, 1)
		y += 11 * u
	} else {
		segs := []Seg{{"◆ " + d.Super.Name + "  ", lk.accent}, {"@", art.Yellow}, {fmt.Sprint(d.Super.Cost), art.Yellow}}
		a.drawSegsIcons(dst, segs, x, y, u, 1)
		y += 11 * u
		for _, l := range a.wrap(d.Super.Desc, r.W, u) {
			a.drawText(dst, l, x, y, u, art.Silver, 1)
			y += 11 * u
		}
	}
	y += 5 * u
	a.drawSegsIcons(dst, towerTraits[d.ID], x, y, u, 1)
	y += 11 * u
	if d.Aims() && a.world.CanTarget() {
		a.drawSegsIcons(dst, []Seg{{"Target priority (T): ", art.Gray}, {"First · Last · Strong · Close", art.White}}, x, y, u, 1)
	}
}

// statCells splits label/value segment pairs into separate table cells.
func statCells(segs []Seg) [][]Seg {
	var cells [][]Seg
	for i := 0; i+1 < len(segs); i += 2 {
		l := segs[i]
		l.S = strings.TrimLeft(l.S, " ")
		cells = append(cells, []Seg{l, segs[i+1]})
	}
	return cells
}

// drawTable draws rows of cells in aligned columns and returns the y below.
func (a *App) drawTable(dst *ebiten.Image, rows [][][]Seg, x, y, gap int) int {
	u := a.u
	var widths []int
	for _, r := range rows {
		for i, c := range r {
			w := a.segsWidthIcons(c, u)
			if i == len(widths) {
				widths = append(widths, w)
			}
			widths[i] = max(widths[i], w)
		}
	}
	for _, r := range rows {
		cx := x
		for i, c := range r {
			a.drawSegsIcons(dst, c, cx, y, u, 1)
			cx += widths[i] + gap
		}
		y += 11 * u
	}
	return y
}

func (a *App) drawEnemyInfo(dst *ebiten.Image, d *game.EnemyDef, r Rect) {
	u := a.u
	lk := a.as.Looks[d]
	col := art.White
	if d.Boss {
		col = art.Pink
	}
	y := a.infoHead(dst, r, lk.card.img, ebiten.ColorScale{}, d.Name, col,
		Seg{enemyRoles[d.ID], art.Gray}, Seg{fmt.Sprintf("@%d per kill", d.Reward), art.Yellow})
	for _, l := range a.wrap(d.Desc, r.W, u) {
		a.drawText(dst, l, r.X, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 5 * u
	for _, l := range a.enemyLines(d, 1) {
		a.drawSegsIcons(dst, l, r.X, y, u, 1)
		y += 11 * u
	}
	y += 5 * u
	a.drawText(dst, "Health grows with every wave.", r.X, y, u, art.Gray, 1)
}

func (a *App) drawUnknownEnemy(dst *ebiten.Image, d *game.EnemyDef, r Rect) {
	u := a.u
	lk := a.as.Looks[d]
	y := a.infoHead(dst, r, lk.card.sil, csCol(art.Slate, 1), "???", art.Gray, Seg{"Unknown enemy", art.Gray})
	a.drawText(dst, "Not encountered yet.", r.X, y, u, art.Silver, 1)
	a.drawText(dst, "Its details appear here when it first attacks.", r.X, y+11*u, u, art.Gray, 1)
}

var _ = math.Pi
