package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"outpost/internal/art"
	"outpost/internal/game"
)

const version = "v0.15.0"

// demoSpec is the backdrop battle of the menus: the skirmish map with short
// breaks and a base that cannot fall.
var demoSpec = func() *game.LevelSpec {
	s := *game.Skirmish
	s.FirstPrep, s.Prep = 1, 3
	s.BaseHP = 1 << 30
	return &s
}()

// demoTowers is the defence of the backdrop battle: pad, turret type, level.
var demoTowers = []struct {
	slot  int
	id    string
	level int
}{
	{2, "blaster", 1}, {5, "cryo", 1}, {6, "flamer", 3}, {9, "blaster", 0}, {3, "railgun", 2},
	{0, "skyguard", 1}, {4, "blaster", 3}, {7, "cryo", 3}, {8, "railgun", 3},
}

func (a *App) startDemo() {
	a.bankRun()
	a.demo = true
	a.spec = demoSpec
	a.lvl = a.demoLvl
	a.setTerrain(demoSpec.ID)
	a.restart()
}

// setupDemo builds the backdrop defence and lets the battle run for a few
// seconds, so that the road is busy from the first frame.
func (a *App) setupDemo() {
	w := a.world
	for _, t := range demoTowers {
		w.PlaceTower(t.slot, game.TowerDefs[t.id], t.level)
	}
	w.DebugStartAt(2+a.runs%5, 0)
	w.Update(7)
	w.Events = w.Events[:0]
}

// cinematic drifts the backdrop camera slowly over the battle.
func (a *App) cinematic(dt float64) {
	c := a.cam
	a.demoT += dt
	c.Z = c.Fit * 1.3
	c.TZ = c.Z
	hw, hh := c.vw/2/c.Z, c.vh/2/c.Z
	ax, ay := math.Max(0, c.MapW/2-hw), math.Max(0, c.MapH/2-hh)
	c.X = c.MapW/2 + ax*math.Sin(a.demoT*0.05)
	c.Y = c.MapH/2 + ay*math.Sin(a.demoT*0.037+1.3)
	c.clamp()
}

func (a *App) drawFade(dst *ebiten.Image) {
	if a.fadeIn > 0 {
		fillRect(dst, 0, 0, a.sw, a.sh, pm(black, easeOut(a.fadeIn)))
	}
}

// ---- Title screen ----

type menuLayout struct {
	x, titleY, tagY int
	buttons         []Button
}

const titleScale = 5

func (a *App) menuLayout() menuLayout {
	u := a.u
	ml := menuLayout{x: 28 * u, titleY: max(20*u, a.sh*20/100)}
	big := titleScale * u
	ml.tagY = ml.titleY + 2*7*big + 8*u + 6*u
	y := ml.tagY + 26*u
	bw := 130 * u
	for _, b := range []Button{
		{ID: "play", Label: "Play", Key: "Enter", Primary: true},
		{ID: "colony", Label: "Colonies", Key: "C"},
		{ID: "settings", Label: "Settings"},
		{ID: "quit", Label: "Quit", Key: "Q"},
	} {
		b.Enabled = true
		b.R = Rect{ml.x, y, bw, a.btnHeight()}
		ml.buttons = append(ml.buttons, b)
		y += 18 * u
	}
	return ml
}

func (a *App) layoutMenu() {
	a.buttons = append(a.buttons, a.menuLayout().buttons...)
}

// shadeLeft darkens the left part of the screen behind the menu text.
func (a *App) shadeLeft(dst *ebiten.Image, frac, alpha float64) {
	w := int(float64(a.sw) * frac)
	const n = 48
	for i := 0; i < n; i++ {
		x0, x1 := w*i/n, w*(i+1)/n
		al := alpha * (1 - smooth(0, 1, (float64(i)+0.5)/n))
		fillRect(dst, x0, 0, x1-x0, a.sh, pm(art.Ink, al))
	}
}

func (a *App) drawMenu(dst *ebiten.Image) {
	u := a.u
	fillRect(dst, 0, 0, a.sw, a.sh, pm(art.Ink, 0.18))
	a.shadeLeft(dst, 0.62, 0.9)
	ml := a.menuLayout()
	big := titleScale * u
	in := smooth(0, 0.8, a.now)
	a.drawText(dst, "OUTPOST", ml.x, ml.titleY, big, art.White, in)
	a.drawText(dst, "DEFENSE", ml.x, ml.titleY+7*big+8*u, big, art.Amber, in)
	fillRect(dst, ml.x, ml.tagY-6*u, 46*u, 2*u, pm(art.Cyan, in))
	a.drawText(dst, "Hold the line against the hive.", ml.x, ml.tagY+2*u, u, art.Silver, in)
	for _, b := range ml.buttons {
		a.drawButton(dst, b, in)
	}
	// Campaign progress and the version at the bottom.
	c := a.currentChapter()
	got, all := a.totalStars()
	y := a.sh - 16*u
	segs := []Seg{{"★", art.Amber}, {fmt.Sprintf(" %d/%d   ", got, all), art.White},
		{"◆", art.Pink}, {fmt.Sprintf(" %d   ", a.save.Crystals), art.White},
		{"{trophy}", art.Amber}, {fmt.Sprintf(" %d/%d", a.save.trophyCount(game.Trophies), len(game.Trophies)), art.White},
		{fmt.Sprintf("   Chapter %d · %s", c.Num, c.Name), art.Gray}}
	a.drawSegsIcons(dst, segs, ml.x, y, u, in)
	a.drawText(dst, version, a.sw-a.textWidth(version, u)-10*u, y, u, art.Gray, in)
}

// ---- Level select ----

// levelSlots are the cards of the level select: the levels of the chapter,
// then the ones still to come.
func (a *App) levelSlots() []*game.LevelSpec {
	c := a.chapter
	out := append([]*game.LevelSpec{}, c.Levels...)
	for i := 0; i < c.Soon; i++ {
		out = append(out, nil)
	}
	return out
}

const levelCols = 4

type levelCard struct {
	spec  *game.LevelSpec // nil: coming soon
	num   int             // place in the chapter, from 1
	btn   Button
	thumb Rect
}

type levelsLayout struct {
	back   Button
	titleX int
	top    int
	cards  []levelCard
	detail Rect
	play   Button
}

func (a *App) levelsLayout() levelsLayout {
	u := a.u
	m := 10 * u
	var ll levelsLayout
	ll.back = Button{ID: "back", Label: "Back", Key: "Esc", Enabled: true}
	ll.back.R = Rect{m, m, a.buttonWidth(ll.back), a.btnHeight()}
	ll.titleX = m + ll.back.R.W + 12*u
	ll.top = m + 14*u + 5*u + 11*u + 12*u
	dw := int(clampF(float64(a.sw)*0.33, float64(185*u), float64(250*u)))
	gap := 6 * u
	gw := a.sw - 2*m - dw - 12*u
	cw := (gw - (levelCols-1)*gap) / levelCols
	tw := cw - 8*u
	th := int(float64(tw) * 0.44)
	ch := 4*u + th + 4*u + 10*u + 11*u + 14*u
	slots := a.levelSlots()
	rows := (len(slots) + levelCols - 1) / levelCols
	gh := rows*ch + (rows-1)*gap
	y0 := ll.top + max(0, (a.sh-m-ll.top-gh)/3)
	for i, s := range slots {
		x := m + (i%levelCols)*(cw+gap)
		y := y0 + (i/levelCols)*(ch+gap)
		lc := levelCard{spec: s, num: i + 1}
		lc.btn = Button{ID: fmt.Sprintf("lv:%d", i), R: Rect{x, y, cw, ch}, Enabled: s != nil}
		lc.thumb = Rect{x + 4*u, y + 4*u, tw, th}
		ll.cards = append(ll.cards, lc)
	}
	ll.detail = Rect{a.sw - m - dw, ll.top, dw, a.sh - m - ll.top}
	s := a.pickSpec()
	play := Button{ID: "start", Label: "Play", Key: "Enter", Primary: true, Enabled: s != nil && a.unlocked(s)}
	if s != nil && !a.unlocked(s) {
		play.Label, play.Key = "Locked", ""
	}
	pw := dw - 16*u
	play.R = Rect{ll.detail.X + 8*u, ll.detail.Y + ll.detail.H - a.btnHeight() - 8*u, pw, a.btnHeight()}
	ll.play = play
	return ll
}

func (a *App) pickSpec() *game.LevelSpec {
	slots := a.levelSlots()
	if a.pick < 0 || a.pick >= len(slots) {
		return nil
	}
	return slots[a.pick]
}

func (a *App) layoutLevels() {
	ll := a.levelsLayout()
	a.addButton(ll.back)
	for _, c := range ll.cards {
		if c.spec != nil {
			a.addButton(c.btn)
		}
	}
	a.addButton(ll.play)
}

// openLevels leaves a level for the level select of its chapter (the
// chapter select for the skirmish).
func (a *App) openLevels() {
	c, _ := chapterOf(a.spec)
	if a.demo {
		c = a.chapter
	}
	if c == nil {
		card := len(game.Chapters)
		if isExpedition(a.spec) && !a.demo {
			card = expCard()
		}
		a.openChapters()
		a.chap = card
		return
	}
	if !a.demo {
		a.startDemo()
		a.fadeIn = 1
	}
	a.showLevels(c)
}

// showLevels opens the level select of a chapter.
func (a *App) showLevels(c *game.Chapter) {
	a.chapter = c
	a.chap = c.Num - 1
	a.scr = scrLevels
	a.paused = false
	a.popup = panelNone
	a.pick = a.defaultPick()
}

// defaultPick selects the level of the chapter the player most likely wants
// next: the one played last if it is not won yet, else the first new
// unlocked level.
func (a *App) defaultPick() int {
	slots := a.levelSlots()
	last := -1
	for i, s := range slots {
		if s != nil && s.ID == a.save.Last {
			last = i
		}
	}
	if last >= 0 && a.best(slots[last]) == 0 {
		return last
	}
	for i, s := range slots {
		if s != nil && a.unlocked(s) && a.best(s) == 0 {
			return i
		}
	}
	return max(0, last)
}

func (a *App) pickCard(i int) {
	if i == a.pick {
		a.playPick()
		return
	}
	a.pick = i
}

func (a *App) playPick() {
	if s := a.pickSpec(); s != nil && a.unlocked(s) {
		a.startLevel(s)
	}
}

func (a *App) levelKeys(enter bool) {
	jp := inpututil.IsKeyJustPressed
	if jp(ebiten.KeyEscape) {
		a.scr = scrChapters
		return
	}
	if enter {
		a.playPick()
		return
	}
	slots := a.levelSlots()
	ok := func(i int) bool { return i >= 0 && i < len(slots) && slots[i] != nil }
	step := func(d int) {
		for i := a.pick + d; i >= 0 && i < len(slots); i += d {
			if ok(i) {
				a.pick = i
				return
			}
		}
	}
	vert := func(d int) {
		row := a.pick/levelCols + d
		if row < 0 || row*levelCols >= len(slots) {
			return
		}
		best, bd := -1, 1<<30
		for i := row * levelCols; i < min(len(slots), (row+1)*levelCols); i++ {
			if ok(i) {
				if dd := abs(i%levelCols - a.pick%levelCols); dd < bd {
					best, bd = i, dd
				}
			}
		}
		if best >= 0 {
			a.pick = best
		}
	}
	switch {
	case jp(ebiten.KeyArrowLeft) || jp(ebiten.KeyA):
		step(-1)
	case jp(ebiten.KeyArrowRight) || jp(ebiten.KeyD):
		step(1)
	case jp(ebiten.KeyArrowUp) || jp(ebiten.KeyW):
		vert(-1)
	case jp(ebiten.KeyArrowDown) || jp(ebiten.KeyS):
		vert(1)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawLevelPic draws a small picture of a level's map into r.
func (a *App) drawLevelPic(dst *ebiten.Image, s *game.LevelSpec, r Rect, cs ebiten.ColorScale) {
	t := a.as.Terrains[s.ID]
	l := a.levels[s.ID]
	b := t.img.Bounds()
	k := math.Min(float64(r.W)/float64(b.Dx()), float64(r.H)/float64(b.Dy()))
	ox := float64(r.X) + (float64(r.W)-float64(b.Dx())*k)/2
	oy := float64(r.Y) + (float64(r.H)-float64(b.Dy())*k)/2
	draw := func(img *ebiten.Image, x, y float64) {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, ColorScale: cs}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(ox+x*k, oy+y*k)
		dst.DrawImage(img, op)
	}
	draw(t.img, 0, 0)
	P := float64(art.CellPx)
	bc := l.BaseCenter()
	cx, cy := bc.X*P, bc.Y*P
	px, py := art.Project(cx, cy, 0)
	base := a.as.baseOf(l)
	draw(base.img, px-base.ox, py+float64(t.ti.OY)-base.oy)
	rp := art.RadarPos
	rd := a.as.Radar[angleIndex(0.7, radarAngles)]
	px, py = art.Project(cx+rp.X, cy+rp.Y, rp.Z)
	draw(rd.img, px-rd.ox, py+float64(t.ti.OY)-rd.oy)
}

func (a *App) drawLevels(dst *ebiten.Image) {
	u := a.u
	fillRect(dst, 0, 0, a.sw, a.sh, pm(art.Ink, 0.72))
	ll := a.levelsLayout()
	c := a.chapter
	m := 10 * u
	a.drawButton(dst, ll.back, 1)
	a.drawText(dst, fmt.Sprintf("Chapter %d · %s", c.Num, c.Name), ll.titleX, m, 2*u, art.White, 1)
	a.drawText(dst, c.Desc, ll.titleX, m+14*u+6*u, u, art.Silver, 1)
	// Stars of the chapter, top right.
	st := fmt.Sprintf("%d/%d", a.chapterStars(c), 3*len(c.Levels))
	sw := a.textWidth(st, 2*u)
	x := a.sw - m - sw
	a.drawText(dst, st, x, m, 2*u, art.White, 1)
	a.drawIcon(dst, "star", x-26*u, m-4*u, 2*u, 1)
	for i, lc := range ll.cards {
		a.drawLevelCard(dst, lc, i == a.pick)
	}
	a.drawLevelDetail(dst, ll)
}

func (a *App) drawLevelCard(dst *ebiten.Image, lc levelCard, sel bool) {
	u := a.u
	r := lc.btn.R
	hover := a.hoverBtn == lc.btn.ID
	a.panel(dst, r, 1)
	tr := lc.thumb
	fillRect(dst, tr.X, tr.Y, tr.W, tr.H, bgColor)
	s := lc.spec
	ty := tr.Y + tr.H + 4*u
	if s == nil {
		q := "?"
		a.drawText(dst, q, tr.X+tr.W/2-a.textWidth(q, 3*u)/2, tr.Y+tr.H/2-10*u, 3*u, art.Slate, 1)
		a.drawText(dst, fmt.Sprintf("%d-%d", a.chapter.Num, lc.num), tr.X, ty, u, art.Slate, 1)
		a.drawText(dst, "Coming soon", tr.X, ty+10*u, u, art.Gray, 1)
		return
	}
	open := a.unlocked(s)
	var cs ebiten.ColorScale
	if !open {
		cs.Scale(0.3, 0.3, 0.36, 1)
	}
	a.drawLevelPic(dst, s, tr, cs)
	if !open {
		a.drawIcon(dst, "lock", tr.X+tr.W/2-11*u, tr.Y+tr.H/2-11*u, 2*u, 1)
	}
	best := a.best(s)
	if open && best == 0 {
		al := 0.75 + 0.25*math.Sin(a.now*5)
		nw := a.textWidth("NEW", u) + 6*u
		fillRect(dst, tr.X+tr.W-nw-2*u, tr.Y+2*u, nw, 11*u, pm(art.Ink, 0.8))
		a.drawText(dst, "NEW", tr.X+tr.W-nw+u, tr.Y+4*u, u, art.Amber, al)
	}
	label, lcol := s.ID, art.Cyan
	if s == game.Skirmish {
		label, lcol = "Skirmish", art.Pink
	}
	nameCol := art.White
	if !open {
		lcol, nameCol = art.Gray, art.Gray
	}
	a.drawText(dst, label, tr.X, ty, u, lcol, 1)
	a.drawText(dst, s.Name, tr.X, ty+10*u, u, nameCol, 1)
	for i := 0; i < 3; i++ {
		ic := "star0"
		if i < best {
			ic = "star"
		}
		a.drawIcon(dst, ic, tr.X+i*12*u, ty+22*u, u, 1)
	}
	// Frame: cyan when selected, lighter on hover.
	var edge color.RGBA
	switch {
	case sel:
		edge = art.Cyan
	case hover:
		edge = art.Steel
	}
	if edge.A != 0 {
		fillRect(dst, r.X, r.Y, r.W, u, edge)
		fillRect(dst, r.X, r.Y+r.H-u, r.W, u, edge)
		fillRect(dst, r.X, r.Y, u, r.H, edge)
		fillRect(dst, r.X+r.W-u, r.Y, u, r.H, edge)
	}
}

func (a *App) drawLevelDetail(dst *ebiten.Image, ll levelsLayout) {
	a.drawSpecDetail(dst, ll.detail, a.pickSpec(), ll.play)
}

// drawSpecDetail describes a level in the panel r above its play button.
func (a *App) drawSpecDetail(dst *ebiten.Image, r Rect, s *game.LevelSpec, play Button) {
	u := a.u
	a.panel(dst, r, 1)
	a.drawButton(dst, play, 1)
	if s == nil {
		return
	}
	open := a.unlocked(s)
	x, w := r.X+8*u, r.W-16*u
	y := r.Y + 8*u
	label := "Level " + s.ID
	switch {
	case s == game.Skirmish && open:
		label = "Skirmish · every turret unlocked"
	case s == game.Skirmish:
		label = "Skirmish · bonus map"
	}
	a.drawText(dst, label, x, y, u, art.Cyan, 1)
	// How hard it is: it sets the crystals its stars pay.
	tc := map[game.Tier]color.RGBA{game.TierLow: art.Lime, game.TierMedium: art.Amber, game.TierHigh: art.Red}[s.Tier]
	tsegs := []Seg{{"Threat ", art.Gray}, {s.Tier.String(), tc}}
	a.drawSegs(dst, tsegs, x+w-a.segsWidth(tsegs, u), y, u, 1)
	y += 11 * u
	a.drawText(dst, s.Name, x, y, 2*u, art.White, 1)
	y += 14*u + 6*u
	for _, l := range a.wrap(s.Desc, w, u) {
		a.drawText(dst, l, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 5 * u
	// What the level brings.
	if l := s.Lesson; l != nil && l.Turret != "" {
		d := game.TowerDefs[l.Turret]
		lk := a.as.Towers[d]
		a.drawText(dst, "New turret", x, y+7*u, u, art.Gray, 1)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(x+56*u), float64(y))
		dst.DrawImage(lk.icon.img, op)
		a.drawText(dst, d.Name, x+84*u, y+7*u, u, lk.accent, 1)
		y += 24 * u
	} else if l != nil && l.Ability != "" {
		a.drawText(dst, "New ability", x, y+7*u, u, art.Gray, 1)
		a.drawIcon(dst, l.Icon, x+56*u+u, y+u, 2*u, 1)
		a.drawText(dst, l.Ability, x+84*u, y+7*u, u, art.Lime, 1)
		y += 24 * u
	} else if l != nil && l.Threat != "" {
		a.drawText(dst, "New threat", x, y+7*u, u, art.Gray, 1)
		if d := game.EnemyDefs[l.Icon]; d != nil {
			// A boss: its portrait.
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(u), float64(u))
			op.GeoM.Translate(float64(x+59*u), float64(y+4*u))
			dst.DrawImage(a.as.Looks[d].icon.img, op)
		} else {
			a.drawIcon(dst, l.Icon, x+56*u+u, y+u, 2*u, 1)
		}
		a.drawText(dst, l.Threat, x+84*u, y+7*u, u, art.Amber, 1)
		y += 24 * u
	} else if s.Turrets == nil {
		a.drawText(dst, "Turrets", x, y+7*u, u, art.Gray, 1)
		for i, d := range game.TowerOrder {
			img := a.as.Towers[d].icon.img
			var cs ebiten.ColorScale
			if !a.knownTurret(d) {
				img, cs = a.as.Towers[d].icon.sil, csCol(art.Steel, 1)
			}
			op := &ebiten.DrawImageOptions{ColorScale: cs}
			op.GeoM.Scale(float64(u), float64(u))
			op.GeoM.Translate(float64(x+56*u+i*25*u), float64(y))
			dst.DrawImage(img, op)
		}
		y += 24 * u
	}
	a.drawText(dst, "Enemies", x, y+3*u, u, art.Gray, 1)
	var foes []*game.EnemyDef
	for _, d := range s.Enemies() {
		if l := s.Lesson; l == nil || game.EnemyDefs[l.Icon] != d { // a boss threat is shown above
			foes = append(foes, d)
		}
	}
	gap := 20 * u
	if n := len(foes); n > 1 {
		gap = min(gap, (w-56*u-18*u)/(n-1))
	}
	for i, d := range foes {
		lk := a.as.Looks[d]
		img := lk.icon.img
		var cs ebiten.ColorScale
		if !a.seen[d] {
			img = lk.icon.sil
			cs = csCol(art.Steel, 1)
		}
		op := &ebiten.DrawImageOptions{ColorScale: cs}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(x+56*u+i*gap), float64(y))
		dst.DrawImage(img, op)
	}
	y += 20 * u
	segs := []Seg{{"Waves ", art.Gray}, {fmt.Sprint(len(s.Waves)), art.White}, {"   Credits ", art.Gray},
		{"@", art.Yellow}, {fmt.Sprint(s.StartCredits), art.Yellow}}
	a.drawSegsIcons(dst, segs, x, y, u, 1)
	y += 17 * u
	var cs ebiten.ColorScale
	if open {
		best := a.best(s)
		first, again := game.StarPay(s.Tier)
		rules := []string{"Win the level", "Keep half of the base", "Keep 90% of the base"}
		for i, t := range rules {
			for j := 0; j <= i; j++ {
				ic := "star0"
				if best > i {
					ic = "star"
				}
				a.drawIcon(dst, ic, x+j*12*u, y-2*u, u, 1)
			}
			col := art.Gray
			if best > i {
				col = art.White
			}
			a.drawText(dst, t, x+40*u, y, u, col, 1)
			// What the star pays: in full the first time, a little on replays.
			pay, pc, pa := fmt.Sprint(first), art.Pink, 1.0
			if a.save.Paid[s.ID] > i {
				pay, pc, pa = fmt.Sprint(again), art.Gray, 0.6
			}
			pw := a.textWidth(pay, u)
			a.drawText(dst, pay, x+w-pw, y, u, pc, 1)
			a.drawIcon(dst, "crystal", x+w-pw-12*u, y-2*u, u, pa)
			y += 13 * u
		}
	} else {
		a.drawIcon(dst, "lock", x, y-2*u, u, 1)
		a.drawText(dst, lockText(s), x+15*u, y, u, art.Amber, 1)
		y += 13 * u
		cs.Scale(0.35, 0.35, 0.4, 1)
	}
	// The map picture fills what is left above the button.
	room := play.R.Y - 8*u - y
	if th := min(room, int(float64(w)*0.44)); th >= 36*u {
		tw := int(float64(th) / 0.44)
		pr := Rect{x + (w-min(w, tw))/2, y + (room-th)/2 + 2*u, min(w, tw), th}
		fillRect(dst, pr.X, pr.Y, pr.W, pr.H, bgColor)
		a.drawLevelPic(dst, s, pr, cs)
	}
}

// knownTurret reports whether the player has met a turret type: it is open on
// some unlocked campaign level, or the Skirmish map (every turret) is open.
func (a *App) knownTurret(d *game.TowerDef) bool {
	if a.unlocked(game.Skirmish) {
		return true
	}
	for _, c := range game.Chapters {
		for _, l := range c.Levels {
			if a.unlocked(l) && l.Allows(d) {
				return true
			}
		}
	}
	return false
}
