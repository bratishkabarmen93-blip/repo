package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

// stripItem is one enemy type in the next-wave preview, or the letter of the
// entrance the following entries come out of.
type stripItem struct {
	game.WaveEntry
	R     Rect
	New   bool
	Badge bool
}

// entranceCols tell the burrows of a level apart: A, B, C.
var entranceCols = []color.RGBA{art.Amber, art.Lilac, art.Lime}

func entranceName(i int) string { return string(rune('A' + i)) }

func (a *App) stripVisible() bool {
	_, ok := a.world.NextWave()
	return ok && !a.modal()
}

// stripLayout returns the preview panel and its entries. On a level with
// several burrows the entries are grouped by the entrance they come out of.
func (a *App) stripLayout() (Rect, []stripItem) {
	spec, ok := a.world.NextWave()
	if !ok {
		return Rect{}, nil
	}
	u := a.u
	label := "Next wave"
	x := a.textWidth(label, u) + 10*u
	var items []stripItem
	add := func(it stripItem, w int) {
		it.R = Rect{x, 0, w, 18 * u}
		x += w + 7*u
		items = append(items, it)
	}
	n := len(a.lvl.Routes)
	for from := 0; from < n; from++ {
		entries := spec.SummaryFrom(from, n)
		if n > 1 {
			if len(entries) == 0 {
				continue
			}
			if len(items) > 0 {
				x += 5 * u // a wider gap between burrows
			}
			add(stripItem{WaveEntry: game.WaveEntry{From: from}, Badge: true}, 9*u)
			x -= 3 * u
		}
		for _, en := range entries {
			it := stripItem{WaveEntry: en, New: !a.seen[en.Def]}
			w := 20*u + a.textWidth(fmt.Sprintf("×%d", en.Count), u)
			if it.New {
				w += 4*u + a.textWidth("NEW", u)
			}
			add(it, w)
		}
	}
	pw := x - 7*u + 12*u
	panel := Rect{a.sw/2 - pw/2, a.barH + 3*u, pw, 22 * u}
	for i := range items {
		items[i].R.X += panel.X + 6*u
		items[i].R.Y = panel.Y + 2*u
	}
	return panel, items
}

// drawBadge draws the letter of an entrance in a coloured tag (9u × 11u).
func (a *App) drawBadge(dst *ebiten.Image, i, x, y int, al float64) {
	u := a.u
	col := entranceCols[i%len(entranceCols)]
	fillRect(dst, x, y, 9*u, 11*u, pm(art.Ink, al))
	fillRect(dst, x+u, y+u, 7*u, 9*u, pm(col, al))
	fillRect(dst, x+u, y+9*u, 7*u, u, pm(lerpCol(col, art.Ink, 0.4), al))
	name := entranceName(i)
	a.drawTextRaw(dst, name, x+(9*u-a.textWidth(name, u))/2, y+2*u, u, art.Ink, al, false)
}

// drawBurrowTags marks the burrows of a level with several entrances. The
// ones the coming wave uses are lit, the quiet ones dimmed.
func (a *App) drawBurrowTags(dst *ebiten.Image) {
	if len(a.lvl.Routes) < 2 || a.modal() {
		return
	}
	u := a.u
	w := a.world
	var next []bool
	if spec, ok := w.NextWave(); ok && w.Phase == game.PhasePrep {
		for i := range a.lvl.Routes {
			next = append(next, len(spec.SummaryFrom(i, len(a.lvl.Routes))) > 0)
		}
	}
	for i, rt := range a.lvl.Routes {
		bx, by := art.BurrowPos(a.lvl, rt.Spawn)
		// Beside the road: above a road that leads right, to the right of
		// one that leads down.
		var dx, dy float64 = 1, 0
		if len(rt.Cells) > 1 {
			dx, dy = float64(rt.Cells[1].X-rt.Cells[0].X), float64(rt.Cells[1].Y-rt.Cells[0].Y)
		}
		ox, oy := dy, -dx
		sx, sy := a.screenAt(bx+ox*0.62*art.CellPx, by+oy*0.62*art.CellPx, 8)
		x := int(sx) - 9*u/2
		y := int(sy) - 11*u/2
		x = max(2*u, min(x, a.sw-11*u))
		y = max(a.barH+2*u, min(y, a.sh-13*u))
		al := 0.5
		switch {
		case w.Phase == game.PhaseWave && w.Spawning(i):
			al = 1
		case i < len(next) && next[i]:
			al = 0.85 + 0.15*math.Sin(a.now*4)
		}
		a.drawBadge(dst, i, x, y, al)
	}
}
func (a *App) stripItems() []stripItem {
	if !a.stripVisible() {
		return nil
	}
	_, items := a.stripLayout()
	return items
}

func (a *App) drawStrip(dst *ebiten.Image) {
	if !a.stripVisible() {
		return
	}
	u := a.u
	panel, items := a.stripLayout()
	a.panel(dst, panel, 1)
	ty := panel.Y + (panel.H-7*u)/2
	a.drawText(dst, "Next wave", panel.X+6*u, ty, u, art.Gray, 1)
	for i, it := range items {
		if it.Badge {
			if i > 0 {
				fillRect(dst, it.R.X-7*u, panel.Y+5*u, u, panel.H-10*u, pm(art.Slate, 1))
			}
			a.drawBadge(dst, it.From, it.R.X, it.R.Y+(it.R.H-11*u)/2, 1)
			continue
		}
		lk := a.as.Looks[it.Def]
		hover := a.tipEnemy == it.Def && a.tipFrom == it.From
		if hover {
			fillRect(dst, it.R.X-2*u, it.R.Y, it.R.W+4*u, it.R.H, pm(art.Slate, 0.7))
		}
		ic := lk.icon
		b := ic.img.Bounds()
		ix := it.R.X + (18*u-b.Dx()*u)/2
		iy := it.R.Y + (it.R.H-b.Dy()*u)/2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(ix), float64(iy))
		dst.DrawImage(ic.img, op)
		x := it.R.X + 20*u
		col := art.White
		if it.Def.Boss {
			col = art.Pink
		}
		x = a.drawText(dst, fmt.Sprintf("×%d", it.Count), x, ty, u, col, 1)
		if it.New {
			a.drawText(dst, "NEW", x+4*u, ty, u, art.Amber, 0.75+0.25*math.Sin(a.now*5))
		}
	}
}

// enemyLines describes an enemy type for tooltips and cards.
func (a *App) enemyLines(d *game.EnemyDef, hpMul float64) [][]Seg {
	lc, vc := art.Gray, art.White
	hp := int(math.Round(d.HP * hpMul))
	lines := [][]Seg{{
		{"HP ", lc}, {fmt.Sprint(hp), vc},
		{"  Speed ", lc}, {decimal(d.Speed), vc},
		{"  Base damage ", lc}, {fmt.Sprintf("%d%%", d.Damage), vc},
	}}
	if d.Armor > 0 {
		lines = append(lines, []Seg{{"Armor ", art.Silver}, {fmt.Sprintf("%g", d.Armor), art.White}, {fmt.Sprintf(": each hit deals %g less", d.Armor), art.Gray}})
	}
	if d.Flying {
		lines = append(lines, []Seg{{"Flying: ", art.Amber}, {"only Skyguard turrets can hit it", art.Silver}})
	}
	if d.Heal != nil {
		lines = append(lines, []Seg{{"Repairs ", art.Lime}, {"damaged enemies around it", art.Silver}})
	}
	if b := d.Brood; b != nil {
		verb := b.Verb
		if verb == "" {
			verb = "lays"
		}
		lines = append(lines, []Seg{{"Boss: ", art.Pink}, {fmt.Sprintf("%s %d %s every %gs", verb, b.Count, game.EnemyDefs[b.Enemy].Plural, b.Period), art.Silver}})
	}
	if d.Dive > 0 {
		lines = append(lines, []Seg{{"Dives: ", art.Cyan}, {"under the ice it can't be hit and swims fast", art.Silver}})
	}
	if sp := d.Spit; sp != nil {
		lines = append(lines, []Seg{{"Surfaces: ", art.Lilac}, {fmt.Sprintf("spits %d %s onto the road ahead", sp.Count, game.EnemyDefs[sp.Enemy].Plural), art.Silver}})
	}
	if sp := d.Split; sp != nil {
		part := game.EnemyDefs[sp.Enemy]
		lines = append(lines, []Seg{{"Splits: ", art.Lilac}, {fmt.Sprintf("into %d %s when destroyed", sp.Count, part.Plural), art.Silver}})
		lines = append(lines, []Seg{{part.Name + ": ", art.Gray}, {fmt.Sprintf("HP %d  Speed %s  Damage %d%% each", int(math.Round(part.HP*hpMul*sp.HPMul)), decimal(part.Speed), part.Damage), art.Silver}})
	}
	if d.NoChill {
		lines = append(lines, []Seg{{"Frost-proof: ", art.Cyan}, {"Cryo can't slow or freeze it", art.Silver}})
	}
	if m := d.FireMul(); m > 1 {
		lines = append(lines, []Seg{{"Melts: ", art.Orange}, {fmt.Sprintf("flames and burning deal %s damage", timesText(m)), art.Silver}})
	}
	return lines
}

func (a *App) drawStripTooltip(dst *ebiten.Image) {
	if a.tipEnemy == nil || !a.stripVisible() {
		return
	}
	u := a.u
	for _, it := range a.stripItems() {
		if it.Badge || it.Def != a.tipEnemy || it.From != a.tipFrom {
			continue
		}
		head := []Seg{{it.Def.Name, art.White}}
		if it.New {
			head = append(head, Seg{"  NEW", art.Amber})
		}
		lines := append([][]Seg{head}, a.enemyLines(it.Def, it.HPMul)...)
		if len(a.lvl.Routes) > 1 {
			lines = append(lines, []Seg{{"Comes out of burrow ", art.Gray}, {entranceName(it.From), entranceCols[it.From%len(entranceCols)]}})
		}
		pw := 0
		for _, l := range lines {
			pw = max(pw, a.segsWidthIcons(l, u))
		}
		pw += 8 * u
		ph := len(lines)*11*u + 5*u
		x := it.R.X + it.R.W/2 - pw/2
		x = max(2*u, min(x, a.sw-pw-2*u))
		y := it.R.Y + it.R.H + 5*u
		a.panel(dst, Rect{x, y, pw, ph}, 1)
		for i, l := range lines {
			a.drawSegsIcons(dst, l, x+4*u, y+4*u+i*11*u, u, 1)
		}
		return
	}
}

// timesText is "double" for 2, "×1.5" otherwise.
func timesText(m float64) string {
	switch m {
	case 2:
		return "double"
	case 3:
		return "triple"
	}
	return fmt.Sprintf("×%g", m)
}

// wrap splits text into lines no wider than maxW pixels.
func (a *App) wrap(s string, maxW, sc int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if cur != "" && a.textWidth(try, sc) > maxW {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur = try
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// cardLayout places the "new enemy" card or a lesson on the right side of the screen.
type cardLayout struct {
	panel  Rect
	btn    Button
	desc   []string
	extra  [][]Seg
	alpha  float64
	def    *game.EnemyDef
	hpMul  float64
	slideX int
	// lesson cards
	lesson            *game.Lesson
	tower             *game.TowerDef
	problem, solution []string
}

// enemyRoles is a short tag line under the name on the "new enemy" card.
var enemyRoles = map[string]string{
	"crawler": "Alien · Basic", "runner": "Alien · Fast", "swarmling": "Alien · Swarm",
	"walker": "Robot · Armored", "drone": "Robot · Flying support", "queen": "Alien · Boss",
	"beetle": "Alien · Frost-proof", "splitter": "Alien · Splits", "shard": "Alien · Splinter",
	"stinger": "Alien · Flying pack", "wyrm": "Alien · Boss",
}

func (a *App) cardLayout() (cardLayout, bool) {
	if len(a.cards) == 0 || a.modal() {
		return cardLayout{}, false
	}
	u := a.u
	if l := a.cards[0].lesson; l != nil {
		return a.lessonLayout(l), true
	}
	d := a.cards[0].enemy
	cl := cardLayout{def: d, hpMul: 1}
	if a.world.Wave > 0 {
		for _, en := range a.lvl.Spec.Waves[a.world.Wave-1].Summary() {
			if en.Def == d {
				cl.hpMul = en.HPMul
			}
		}
	}
	cl.extra = a.enemyLines(d, cl.hpMul)
	pw := max(150*u, 61*u+a.textWidth(d.Name, 2*u)+6*u)
	for _, l := range cl.extra {
		pw = max(pw, a.segsWidthIcons(l, u)+12*u)
	}
	cl.desc = a.wrap(d.Desc, pw-12*u, u)
	head := 46 * u
	ph := 8*u + head + 4*u + len(cl.desc)*11*u + 3*u + len(cl.extra)*11*u + 5*u + a.btnHeight() + 6*u
	out := smooth(cardLife-0.8, cardLife, a.cardT)
	a.placeCard(&cl, pw, ph, out)
	return cl, true
}

// lessonLayout is the briefing of a tutorial level: what it unlocks, as a
// problem and its solution.
func (a *App) lessonLayout(l *game.Lesson) cardLayout {
	u := a.u
	cl := cardLayout{lesson: l, tower: game.TowerDefs[l.Turret]}
	pw := 200 * u
	switch {
	case cl.tower != nil:
		d := cl.tower
		cl.extra = [][]Seg{towerStatSegs(d, d.Levels[0], art.Gray, art.White), towerTraits[d.ID]}
	case l.Icon == "target":
		// The four modes, the one the lesson is about highlighted.
		for m := game.TargetMode(0); m < game.NumTargetModes; m++ {
			name, desc := art.White, art.Gray
			if m == game.TargetStrong {
				name, desc = art.Lime, art.Silver
			}
			cl.extra = append(cl.extra, []Seg{{m.String(), name}, {": " + targetModeDesc[m], desc}})
		}
	case l.Icon == "super":
		for _, d := range a.spec.Arsenal() {
			cl.extra = append(cl.extra, []Seg{{"◆ ", a.as.Towers[d].accent}, {d.Name + ": ", art.Gray}, {d.Super.Name, art.White}})
		}
	case l.Icon == "entrances":
		for i, rt := range a.lvl.Routes {
			cl.extra = append(cl.extra, []Seg{{"Burrow ", art.Gray}, {entranceName(i), entranceCols[i%len(entranceCols)]},
				{": " + edgeText(a.lvl, rt.Spawn), art.Silver}})
		}
	}
	for _, e := range cl.extra {
		pw = max(pw, a.segsWidthIcons(e, u)+12*u)
	}
	cl.problem = a.wrap(l.Problem, pw-12*u, u)
	cl.solution = a.wrap(l.Solution, pw-12*u, u)
	head := 46 * u
	ph := 8*u + head + 4*u + (1+len(cl.problem))*11*u + 4*u + (1+len(cl.solution))*11*u + 4*u +
		len(cl.extra)*11*u + 5*u + a.btnHeight() + 6*u
	a.placeCard(&cl, pw, ph, 0)
	return cl
}

// placeCard slides a card in from the right edge, below the wave preview.
func (a *App) placeCard(cl *cardLayout, pw, ph int, out float64) {
	u := a.u
	k := easeOut(smooth(0, 0.35, a.cardT))
	cl.alpha = math.Min(smooth(0, 0.2, a.cardT), 1-out)
	cl.slideX = int((1 - k) * float64(pw+10*u))
	x := a.sw - pw - 6*u + cl.slideX
	y := a.barH + 8*u
	if a.stripVisible() {
		panel, _ := a.stripLayout()
		if panel.X+panel.W > x-4*u {
			y = panel.Y + panel.H + 6*u
		}
	}
	cl.panel = Rect{x, y, pw, ph}
	b := Button{ID: "gotit", Label: "Got it", Key: "Enter", Enabled: true, Primary: true}
	bw := a.buttonWidth(b)
	b.R = Rect{x + pw - bw - 6*u, y + ph - a.btnHeight() - 6*u, bw, a.btnHeight()}
	cl.btn = b
}

// drawPortrait draws a sprite centred in a dark frame.
func (a *App) drawPortrait(dst *ebiten.Image, pr Rect, img *ebiten.Image, cs ebiten.ColorScale, al float64) {
	u := a.u
	fillRect(dst, pr.X, pr.Y, pr.W, pr.H, pm(art.Navy, 0.9*al))
	fillRect(dst, pr.X, pr.Y+pr.H-u, pr.W, u, pm(art.Slate, al))
	if img == nil {
		return
	}
	b := img.Bounds()
	sc := u
	if b.Dx() <= 11 {
		sc = 3 * u // small icons
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(sc), float64(sc))
	op.GeoM.Translate(float64(pr.X+(pr.W-b.Dx()*sc)/2), float64(pr.Y+(pr.H-b.Dy()*sc)/2))
	op.ColorScale = cs
	op.ColorScale.ScaleAlpha(float32(al))
	dst.DrawImage(img, op)
}

func (a *App) drawCard(dst *ebiten.Image) {
	cl, ok := a.cardLayout()
	if !ok {
		return
	}
	if cl.lesson != nil {
		a.drawLesson(dst, cl)
		return
	}
	u := a.u
	al := cl.alpha
	r := cl.panel
	a.panel(dst, r, al)
	// The amber top edge shrinks as the card's time runs out.
	left := 1 - a.cardT/cardLife
	fillRect(dst, r.X, r.Y, r.W, 2*u, pm(art.Slate, al))
	fillRect(dst, r.X, r.Y, int(float64(r.W)*left), 2*u, pm(art.Amber, al))
	pr := Rect{r.X + 6*u, r.Y + 8*u, 48 * u, 38 * u}
	lk := a.as.Looks[cl.def]
	a.drawPortrait(dst, pr, lk.card.img, ebiten.ColorScale{}, al)

	tx := pr.X + pr.W + 7*u
	a.drawText(dst, "New enemy", tx, r.Y+9*u, u, art.Amber, al)
	big := 2 * u
	name := cl.def.Name
	if a.textWidth(name, big) > r.X+r.W-tx-6*u {
		big = u
	}
	a.drawText(dst, name, tx, r.Y+20*u, big, art.White, al)
	if role := enemyRoles[cl.def.ID]; role != "" {
		a.drawText(dst, role, tx, r.Y+20*u+7*big+5*u, u, art.Gray, al)
	}
	y := pr.Y + pr.H + 5*u
	for _, l := range cl.desc {
		a.drawText(dst, l, r.X+6*u, y, u, art.Silver, al)
		y += 11 * u
	}
	y += 3 * u
	for _, l := range cl.extra {
		a.drawSegsIcons(dst, l, r.X+6*u, y, u, al)
		y += 11 * u
	}
	if n := len(a.cards) - 1; n > 0 {
		a.drawText(dst, fmt.Sprintf("+%d more", n), r.X+6*u, cl.btn.R.Y+(cl.btn.R.H-7*u)/2, u, art.Gray, al)
	}
	a.drawButton(dst, cl.btn, al)
}

// abilityRoles is the tag line under the name of a new ability or threat, by its icon.
var abilityRoles = map[string]string{"target": "Press T on a turret", "super": "For level 3 turrets",
	"entrances": "Watch the next-wave panel"}

// edgeText tells where on the map edge a burrow is.
func edgeText(l *game.Level, c game.Cell) string {
	along := func(v, n int, lo, hi string) string {
		switch {
		case v*3 < n:
			return lo
		case v*3 >= 2*n:
			return hi
		}
		return "in the middle"
	}
	switch {
	case c.X == 0:
		return "left edge, " + along(c.Y, l.H, "near the top", "near the bottom")
	case c.X == l.W-1:
		return "right edge, " + along(c.Y, l.H, "near the top", "near the bottom")
	case c.Y == 0:
		return "top edge, " + along(c.X, l.W, "on the left", "on the right")
	}
	return "bottom edge, " + along(c.X, l.W, "on the left", "on the right")
}

// drawLesson draws the briefing card of a tutorial level.
func (a *App) drawLesson(dst *ebiten.Image, cl cardLayout) {
	u := a.u
	al := cl.alpha
	r := cl.panel
	l := cl.lesson
	a.panel(dst, r, al)
	head, name, role := "New ability", l.Ability, abilityRoles[l.Icon]
	headCol, roleCol := art.Lime, art.Gray
	if l.Threat != "" {
		head, name, headCol = "New threat", l.Threat, art.Amber
	}
	fillRect(dst, r.X, r.Y, r.W, 2*u, pm(headCol, al))
	pr := Rect{r.X + 6*u, r.Y + 8*u, 48 * u, 38 * u}
	img := a.as.Icons[l.Icon]
	if d := game.EnemyDefs[l.Icon]; d != nil {
		// A boss briefing shows the boss itself.
		img = a.as.Looks[d].card.img
		if role == "" {
			role = enemyRoles[d.ID]
		}
	}
	if img == nil {
		img = a.as.Icons["target"]
	}
	if d := cl.tower; d != nil {
		lk := a.as.Towers[d]
		head, name, role, roleCol, img = "New turret", d.Name, d.Role, lk.accent, lk.card.img
	}
	a.drawPortrait(dst, pr, img, ebiten.ColorScale{}, al)
	tx := pr.X + pr.W + 7*u
	a.drawText(dst, head, tx, r.Y+9*u, u, headCol, al)
	big := 2 * u
	if a.textWidth(name, big) > r.X+r.W-tx-6*u {
		big = u
	}
	a.drawText(dst, name, tx, r.Y+20*u, big, art.White, al)
	if role != "" {
		a.drawText(dst, role, tx, r.Y+20*u+7*big+5*u, u, roleCol, al)
	}
	x := r.X + 6*u
	y := pr.Y + pr.H + 5*u
	block := func(title string, col color.RGBA, lines []string) {
		a.drawText(dst, title, x, y, u, col, al)
		y += 11 * u
		for _, s := range lines {
			a.drawText(dst, s, x, y, u, art.Silver, al)
			y += 11 * u
		}
		y += 4 * u
	}
	block("Problem", art.Amber, cl.problem)
	block("Solution", art.Lime, cl.solution)
	for _, e := range cl.extra {
		a.drawSegsIcons(dst, e, x, y, u, al)
		y += 11 * u
	}
	if a.world.Hold && a.world.Wave == 0 {
		a.drawText(dst, "Wave 1 waits for you", x, cl.btn.R.Y+(cl.btn.R.H-7*u)/2, u, art.Gray, al)
	} else if n := len(a.cards) - 1; n > 0 {
		a.drawText(dst, fmt.Sprintf("+%d more", n), x, cl.btn.R.Y+(cl.btn.R.H-7*u)/2, u, art.Gray, al)
	}
	a.drawButton(dst, cl.btn, al)
}

// drawBossBar shows the health of the living boss at the bottom of the screen.
func (a *App) drawBossBar(dst *ebiten.Image) {
	if a.bossT <= 0 || a.modal() {
		return
	}
	boss := a.world.Boss()
	f, hit, under := 0.0, false, false
	def := a.bossDef
	if def == nil {
		def = game.Queen
	}
	hpText := "0"
	if boss != nil {
		f = boss.HP / boss.MaxHP
		hit = boss.HitT < 0.08
		under = boss.Under
		def = boss.Def
		hpText = fmt.Sprintf("%d / %d", int(math.Ceil(boss.HP)), int(boss.MaxHP))
	}
	u := a.u
	al := a.bossT
	bw := min(240*u, a.sw-40*u)
	bh := 6 * u
	x := a.sw/2 - bw/2
	y := a.sh - 14*u
	a.panel(dst, Rect{x - 6*u, y - 15*u, bw + 12*u, bh + 21*u}, al)
	a.drawText(dst, def.Name, x, y-11*u, u, art.Pink, al)
	nx := x + a.textWidth(def.Name, u) + 6*u
	if def.Armor > 0 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(nx), float64(y-11*u))
		op.ColorScale = csAlpha(al)
		dst.DrawImage(a.as.Shield, op)
		a.drawText(dst, fmt.Sprintf("%g", def.Armor), nx+9*u, y-11*u, u, art.Silver, al)
	}
	if under {
		// Out of reach: the bar dims to ice blue while it swims.
		k := 0.6 + 0.4*math.Sin(a.gameT*4)
		t := "under the ice"
		tw := a.textWidth(t, u)
		hx := x + bw - a.textWidth(hpText, u)
		a.drawText(dst, t, max(nx+4*u, hx-tw-8*u), y-11*u, u, art.Cyan, al*(0.7+0.3*k))
	}
	a.drawText(dst, hpText, x+bw-a.textWidth(hpText, u), y-11*u, u, art.Silver, al)
	fillRect(dst, x-u, y-u, bw+2*u, bh+2*u, pm(art.Ink, al))
	fillRect(dst, x, y, bw, bh, pm(art.Plum, al))
	fw := int(float64(bw)*f + 0.5)
	col, top := art.Magenta, art.Pink
	if under {
		col, top = art.Blue, art.Cyan
	}
	if hit {
		col = lerpCol(col, art.White, 0.5)
	}
	fillRect(dst, x, y, fw, bh, pm(col, al))
	fillRect(dst, x, y, fw, u, pm(top, al))
	for i := 1; i < 10; i++ {
		fillRect(dst, x+bw*i/10, y+bh-2*u, u, 2*u, pm(art.Ink, 0.5*al))
	}
}

var _ color.RGBA
