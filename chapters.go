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

// ---- Chapter select ----

// The chapter select shows one card per chapter, then one for the skirmish
// (its index is len(game.Chapters)) and one for the expeditions (expCard).

func chapterCount() int { return len(game.Chapters) + 2 }

// chapterAt is the chapter of a card, nil for the skirmish and the expeditions.
func chapterAt(i int) *game.Chapter {
	if i >= 0 && i < len(game.Chapters) {
		return game.Chapters[i]
	}
	return nil
}

type chapterCard struct {
	ch    *game.Chapter // nil: the skirmish or the expeditions
	btn   Button
	thumb Rect
}

type chaptersLayout struct {
	back   Button
	titleX int
	cards  []chapterCard
	detail Rect
	open   Button
	prep   Button // the prepare window of the expedition on offer (no ID: none)
}

func (a *App) chaptersLayout() chaptersLayout {
	u := a.u
	m := 10 * u
	var cl chaptersLayout
	cl.back = Button{ID: "back", Label: "Back", Key: "Esc", Enabled: true}
	cl.back.R = Rect{m, m, a.buttonWidth(cl.back), a.btnHeight()}
	cl.titleX = m + cl.back.R.W + 12*u
	top := m + 14*u + 5*u + 11*u + 12*u
	dw := int(clampF(float64(a.sw)*0.33, float64(185*u), float64(250*u)))
	gap := 6 * u
	n := chapterCount()
	gw := a.sw - 2*m - dw - 12*u
	cw := (gw - (n-1)*gap) / n
	tw := cw - 8*u
	th := int(float64(tw) * 0.62)
	ch := 4*u + th + 4*u + 10*u + 11*u + 14*u
	y0 := top + max(0, (a.sh-m-top-ch)/3)
	for i := 0; i < n; i++ {
		x := m + i*(cw+gap)
		c := chapterCard{ch: chapterAt(i)}
		c.btn = Button{ID: fmt.Sprintf("ch:%d", i), R: Rect{x, y0, cw, ch}, Enabled: true}
		c.thumb = Rect{x + 4*u, y0 + 4*u, tw, th}
		cl.cards = append(cl.cards, c)
	}
	cl.detail = Rect{a.sw - m - dw, top, dw, a.sh - m - top}
	b := Button{ID: "open", Label: "Open", Key: "Enter", Primary: true, Enabled: a.chapterOpen(a.chap)}
	if chapterAt(a.chap) == nil {
		b.Label = "Play"
	}
	if a.chap == expCard() {
		b.Label = "Launch"
	}
	if !b.Enabled {
		b.Label, b.Key = "Locked", ""
	}
	b.R = Rect{cl.detail.X + 8*u, cl.detail.Y + cl.detail.H - a.btnHeight() - 8*u, dw - 16*u, a.btnHeight()}
	cl.open = b
	if a.chap == expCard() && b.Enabled {
		p := Button{ID: "prep", Label: "Prepare", Key: "P", Enabled: true}
		p.R = Rect{b.R.X, b.R.Y, a.buttonWidth(p), b.R.H}
		cl.open.R.X += p.R.W + 4*u
		cl.open.R.W -= p.R.W + 4*u
		cl.prep = p
	}
	return cl
}

// chapterOpen reports whether a card of the chapter select can be entered.
func (a *App) chapterOpen(i int) bool {
	if c := chapterAt(i); c != nil {
		return a.unlocked(c.Levels[0])
	}
	if i == expCard() {
		return a.expOpen()
	}
	return a.unlocked(game.Skirmish)
}

// chapterPic is the level whose map pictures a chapter card: for the
// expeditions, the one on offer.
func (a *App) chapterPic(i int) *game.LevelSpec {
	if c := chapterAt(i); c != nil {
		return c.Levels[0]
	}
	if i == expCard() {
		if a.expOpen() {
			if e := a.expedition(); e != nil {
				return e.Spec
			}
		}
		return game.Chapters[0].Levels[0]
	}
	return game.Skirmish
}

func (a *App) layoutChapters() {
	cl := a.chaptersLayout()
	a.addButton(cl.back)
	for _, c := range cl.cards {
		a.addButton(c.btn)
	}
	a.addButton(cl.open)
	if cl.prep.ID != "" {
		a.addButton(cl.prep)
	}
}

// openChapters shows the chapter select over the backdrop battle.
func (a *App) openChapters() {
	if !a.demo {
		a.startDemo()
		a.fadeIn = 1
	}
	a.scr = scrChapters
	a.paused = false
	a.popup = panelNone
	a.chap = a.defaultChapter()
}

// defaultChapter is the card of the level the player most likely wants next.
func (a *App) defaultChapter() int {
	if a.save.Last == game.ExpeditionID && a.expOpen() {
		return expCard()
	}
	s := a.nextPick()
	if s == nil {
		return 0
	}
	if c, _ := chapterOf(s); c != nil {
		return c.Num - 1
	}
	return len(game.Chapters)
}

// nextPick is the level the player most likely wants to play next: the one
// played last if it is not won yet, else the first new unlocked level.
func (a *App) nextPick() *game.LevelSpec {
	last := game.LevelByID(a.save.Last)
	if last != nil && a.best(last) == 0 && a.unlocked(last) {
		return last
	}
	for _, c := range game.Chapters {
		for _, s := range c.Levels {
			if a.unlocked(s) && a.best(s) == 0 {
				return s
			}
		}
	}
	// The whole campaign is done: point at the freshly opened skirmish.
	if sk := game.Skirmish; a.unlocked(sk) && a.best(sk) == 0 {
		return sk
	}
	return last
}

func (a *App) pickChapter(i int) {
	if i == a.chap {
		a.enterChapter()
		return
	}
	a.chap = i
}

// enterChapter opens the level select of the chosen chapter, or starts the
// skirmish or the expedition on offer.
func (a *App) enterChapter() {
	if !a.chapterOpen(a.chap) {
		return
	}
	if a.chap == expCard() {
		a.startExpedition()
		return
	}
	c := chapterAt(a.chap)
	if c == nil {
		a.startLevel(game.Skirmish)
		return
	}
	a.showLevels(c)
}

func (a *App) chapterKeys(enter bool) {
	jp := inpututil.IsKeyJustPressed
	switch {
	case jp(ebiten.KeyEscape):
		a.scr = scrMenu
	case enter:
		a.enterChapter()
	case jp(ebiten.KeyP):
		a.openPrepare()
	case jp(ebiten.KeyArrowLeft) || jp(ebiten.KeyA) || jp(ebiten.KeyArrowUp) || jp(ebiten.KeyW):
		a.chap = max(0, a.chap-1)
	case jp(ebiten.KeyArrowRight) || jp(ebiten.KeyD) || jp(ebiten.KeyArrowDown) || jp(ebiten.KeyS):
		a.chap = min(chapterCount()-1, a.chap+1)
	}
}

// totalStars sums the best results of the whole campaign.
func (a *App) totalStars() (got, max int) {
	for _, c := range game.Chapters {
		got += a.chapterStars(c)
		max += 3 * len(c.Levels)
	}
	return got, max
}

// currentChapter is the furthest chapter the player has opened.
func (a *App) currentChapter() *game.Chapter {
	cur := game.Chapters[0]
	for _, c := range game.Chapters {
		if a.unlocked(c.Levels[0]) {
			cur = c
		}
	}
	return cur
}

func (a *App) drawChapters(dst *ebiten.Image) {
	u := a.u
	fillRect(dst, 0, 0, a.sw, a.sh, pm(art.Ink, 0.72))
	cl := a.chaptersLayout()
	m := 10 * u
	a.drawButton(dst, cl.back, 1)
	a.drawText(dst, "Campaign", cl.titleX, m, 2*u, art.White, 1)
	a.drawText(dst, "Pick a chapter, the skirmish or an expedition.", cl.titleX, m+14*u+6*u, u, art.Silver, 1)
	got, all := a.totalStars()
	st := fmt.Sprintf("%d/%d", got, all)
	sw := a.textWidth(st, 2*u)
	x := a.sw - m - sw
	a.drawText(dst, st, x, m, 2*u, art.White, 1)
	a.drawIcon(dst, "star", x-26*u, m-4*u, 2*u, 1)
	for i, c := range cl.cards {
		a.drawChapterCard(dst, i, c, i == a.chap)
	}
	a.drawChapterDetail(dst, cl)
}

func (a *App) drawChapterCard(dst *ebiten.Image, i int, cc chapterCard, sel bool) {
	u := a.u
	r := cc.btn.R
	a.panel(dst, r, 1)
	tr := cc.thumb
	fillRect(dst, tr.X, tr.Y, tr.W, tr.H, bgColor)
	open := a.chapterOpen(i)
	var cs ebiten.ColorScale
	if !open {
		cs.Scale(0.3, 0.3, 0.36, 1)
	}
	a.drawLevelPic(dst, a.chapterPic(i), tr, cs)
	if !open {
		a.drawIcon(dst, "lock", tr.X+tr.W/2-11*u, tr.Y+tr.H/2-11*u, 2*u, 1)
	}
	ty := tr.Y + tr.H + 4*u
	label, name, lcol := "Skirmish", game.Skirmish.Name, art.Pink
	got, all := a.best(game.Skirmish), 3
	exp := i == expCard()
	var ex *game.Expedition
	switch c := cc.ch; {
	case c != nil:
		label, name, lcol = fmt.Sprintf("Chapter %d", c.Num), c.Name, art.Cyan
		got, all = a.chapterStars(c), 3*len(c.Levels)
	case exp:
		label, name, lcol = "Expeditions", "Always a new map", art.Lime
		if open {
			ex = a.expedition()
		}
		if ex != nil {
			name = ex.Spec.Name
		}
		got = min(a.save.Exp.Won, 1) // NEW until the first one is won
	}
	fresh := open && got == 0
	if fresh {
		al := 0.75 + 0.25*math.Sin(a.now*5)
		nw := a.textWidth("NEW", u) + 6*u
		fillRect(dst, tr.X+tr.W-nw-2*u, tr.Y+2*u, nw, 11*u, pm(art.Ink, 0.8))
		a.drawText(dst, "NEW", tr.X+tr.W-nw+u, tr.Y+4*u, u, art.Amber, al)
	}
	nameCol := art.White
	if !open {
		lcol, nameCol = art.Gray, art.Gray
	}
	a.drawText(dst, label, tr.X, ty, u, lcol, 1)
	a.drawText(dst, name, tr.X, ty+10*u, u, nameCol, 1)
	switch {
	case exp:
		if ex != nil {
			t := ex.Spec.Tier
			a.drawSegs(dst, []Seg{{"Threat ", art.Gray}, {t.String(), tierColor(t)}}, tr.X, ty+24*u, u, 1)
		}
	case cc.ch == nil:
		for j := 0; j < 3; j++ {
			ic := "star0"
			if j < got {
				ic = "star"
			}
			a.drawIcon(dst, ic, tr.X+j*12*u, ty+22*u, u, 1)
		}
	default:
		a.drawIcon(dst, "star", tr.X, ty+22*u, u, 1)
		a.drawText(dst, fmt.Sprintf("%d/%d", got, all), tr.X+13*u, ty+24*u, u, art.Silver, 1)
	}
	var edge color.RGBA
	switch {
	case sel:
		edge = art.Cyan
	case a.hoverBtn == cc.btn.ID:
		edge = art.Steel
	}
	if edge.A != 0 {
		fillRect(dst, r.X, r.Y, r.W, u, edge)
		fillRect(dst, r.X, r.Y+r.H-u, r.W, u, edge)
		fillRect(dst, r.X, r.Y, u, r.H, edge)
		fillRect(dst, r.X+r.W-u, r.Y, u, r.H, edge)
	}
}

func (a *App) drawChapterDetail(dst *ebiten.Image, cl chaptersLayout) {
	if a.chap == expCard() {
		a.drawExpeditionDetail(dst, cl)
		return
	}
	c := chapterAt(a.chap)
	if c == nil {
		a.drawSpecDetail(dst, cl.detail, game.Skirmish, cl.open)
		return
	}
	u := a.u
	r := cl.detail
	a.panel(dst, r, 1)
	a.drawButton(dst, cl.open, 1)
	open := a.chapterOpen(a.chap)
	x, w := r.X+8*u, r.W-16*u
	y := r.Y + 8*u
	a.drawText(dst, fmt.Sprintf("Chapter %d", c.Num), x, y, u, art.Cyan, 1)
	y += 11 * u
	a.drawText(dst, c.Name, x, y, 2*u, art.White, 1)
	y += 14*u + 6*u
	for _, l := range a.wrap(c.Desc, w, u) {
		a.drawText(dst, l, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 6 * u
	// The levels with their stars.
	for _, s := range c.Levels {
		col := art.White
		if !a.unlocked(s) {
			col = art.Gray
		}
		a.drawText(dst, s.ID, x, y, u, art.Cyan, 1)
		a.drawText(dst, s.Name, x+22*u, y, u, col, 1)
		best := a.best(s)
		for j := 0; j < 3; j++ {
			ic := "star0"
			if j < best {
				ic = "star"
			}
			a.drawIcon(dst, ic, x+w-(3-j)*12*u, y-2*u, u, 1)
		}
		y += 12 * u
	}
	if c.Soon > 0 {
		a.drawText(dst, fmt.Sprintf("+%d more levels coming soon", c.Soon), x, y, u, art.Gray, 1)
		y += 12 * u
	}
	y += 4 * u
	var enemies []*game.EnemyDef
	have := map[*game.EnemyDef]bool{}
	for _, s := range c.Levels {
		for _, d := range s.Enemies() {
			if !have[d] {
				have[d] = true
				enemies = append(enemies, d)
			}
		}
	}
	a.drawText(dst, "Enemies", x, y+3*u, u, art.Gray, 1)
	gap := 20 * u
	if n := len(enemies); n > 1 {
		gap = min(gap, (w-56*u-18*u)/(n-1))
	}
	for i, d := range enemies {
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
	y += 22 * u
	var cs ebiten.ColorScale
	if !open {
		a.drawIcon(dst, "lock", x, y-2*u, u, 1)
		a.drawText(dst, lockText(c.Levels[0]), x+15*u, y, u, art.Amber, 1)
		y += 13 * u
		cs.Scale(0.35, 0.35, 0.4, 1)
	}
	room := cl.open.R.Y - 8*u - y
	if th := min(room, int(float64(w)*0.44)); th >= 36*u {
		tw := int(float64(th) / 0.44)
		pr := Rect{x + (w-min(w, tw))/2, y + (room-th)/2 + 2*u, min(w, tw), th}
		fillRect(dst, pr.X, pr.Y, pr.W, pr.H, bgColor)
		a.drawLevelPic(dst, a.chapterPic(a.chap), pr, cs)
	}
}
