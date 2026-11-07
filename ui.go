package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

// controls lists the keys and mouse actions of the current level.
func (a *App) controls() [][2]string {
	keys := "1"
	if n := a.buildKeys(); n > 1 {
		keys = fmt.Sprintf("1 - %d", n)
	}
	out := [][2]string{
		{"Left click", "build / select a turret"},
		{keys, "build a turret on the pad"},
		{"U", "upgrade the selected turret"},
	}
	if a.world.CanTarget() {
		out = append(out, [2]string{"T", "change its target priority"})
	}
	return append(out, [][2]string{
		{"X / Delete", "sell (press twice) or cancel"},
		{"Right click", "close the menu"},
		{"Mouse wheel", "zoom to cursor"},
		{"Drag / WASD", "pan the map"},
		{"Home", "show the whole map"},
		{"Space", "start the next wave now"},
		{"Enter", "close a card"},
		{"F", "game speed ×1 / ×2"},
		{"I", "this guide"},
		{"Esc / P", "pause menu"},
		{"F11", "fullscreen"},
	}...)
}

// buildKeys is the highest number key that builds something on this level.
func (a *App) buildKeys() int {
	n := 1
	for i, d := range game.TowerOrder {
		if a.world.CanBuild(d) {
			n = i + 1
		}
	}
	return n
}

func (a *App) btnHeight() int { return 13 * a.u }

// buttonWidth measures label, cost and key hint with padding.
func (a *App) buttonWidth(b Button) int {
	u := a.u
	w := a.textWidth(b.Label, u)
	if b.Cost > 0 {
		w += 4*u + 11*u + 2*u + a.textWidth(fmt.Sprint(b.Cost), u)
	}
	if b.Key != "" {
		w += 6*u + a.textWidth(b.Key, u) + 4*u
	}
	return w + 12*u
}

func (a *App) addButton(b Button) { a.buttons = append(a.buttons, b) }

// layoutUI rebuilds the clickable regions for the current state.
func (a *App) layoutUI() {
	u := a.u
	w := a.world
	a.buttons = a.buttons[:0]
	a.blockers = a.blockers[:0]
	a.tipEnemy = nil
	if a.popup != panelNone {
		a.layoutPanel()
		a.blockers = append(a.blockers, Rect{0, 0, a.sw, a.sh})
		return
	}
	switch a.scr {
	case scrMenu:
		a.layoutMenu()
		return
	case scrChapters:
		a.layoutChapters()
		return
	case scrLevels:
		a.layoutLevels()
		return
	case scrColony:
		a.layoutColony()
		return
	}
	a.blockers = append(a.blockers, Rect{0, 0, a.sw, a.barH})
	if a.modal() {
		a.layoutOverlay()
		return
	}
	bh, by := a.btnHeight(), 3*u
	x := a.sw - 5*u
	pw := 16 * u
	x -= pw
	a.addButton(Button{ID: "pause", R: Rect{x, by, pw, bh}, Label: "II", Enabled: true})
	sb := Button{ID: "speed", Label: fmt.Sprintf("×%d", a.speed), Key: "F", Enabled: true}
	sw := a.buttonWidth(sb)
	x -= sw + 3*u
	sb.R = Rect{x, by, sw, bh}
	a.addButton(sb)
	x -= pw + 3*u
	a.addButton(Button{ID: "guide", R: Rect{x, by, pw, bh}, Icon: "info", Enabled: true})

	if w.Phase == game.PhasePrep {
		b := Button{ID: "skip", Label: "Start now", Key: "Space", Enabled: true, Primary: true}
		bw := a.buttonWidth(b)
		tw := a.centerTextWidth()
		total := tw + 8*u + bw
		b.R = Rect{a.sw/2 - total/2 + tw + 8*u, by, bw, bh}
		a.addButton(b)
	}
	bs, _ := a.towerButtons()
	for _, b := range bs {
		a.addButton(b)
	}
	if bm, ok := a.buildMenuLayout(); ok {
		for _, b := range bm.items {
			a.addButton(b)
		}
	}
	if cl, ok := a.cardLayout(); ok {
		a.addButton(cl.btn)
		a.blockers = append(a.blockers, cl.panel)
	}
	if a.stripVisible() {
		panel, items := a.stripLayout()
		a.blockers = append(a.blockers, panel)
		for _, it := range items {
			if !it.Badge && it.R.Has(a.mx, a.my) {
				a.tipEnemy, a.tipFrom = it.Def, it.From
			}
		}
	}
}

func (a *App) prepText() string {
	if a.world.Hold && a.world.Wave == 0 {
		return "Waiting for you"
	}
	return fmt.Sprintf("Next wave in %ds", int(math.Ceil(a.world.Timer-1e-9)))
}

func (a *App) centerTextWidth() int {
	switch a.world.Phase {
	case game.PhasePrep:
		return 14*a.u + a.textWidth(a.prepText(), a.u)
	case game.PhaseWave:
		return a.textWidth(a.waveText(), a.u)
	}
	return 0
}

func (a *App) waveText() string {
	n := a.world.EnemiesLeft()
	if n == 1 {
		return "1 enemy left"
	}
	return fmt.Sprintf("%d enemies left", n)
}

func (a *App) endAlpha() float64 {
	if a.world.Active() {
		return 0
	}
	return smooth(0.7, 1.3, a.endT)
}

func (a *App) drawUI(dst *ebiten.Image) {
	a.drawBurrowTags(dst)
	a.drawHint(dst)
	a.drawBossBar(dst)
	a.drawTopBar(dst)
	a.drawStrip(dst)
	a.drawBanner(dst)
	a.drawCard(dst)
	a.drawTowerUI(dst)
	if !a.modal() {
		for _, b := range a.buttons {
			if !topBarButton(b.ID) && b.ID != "gotit" && !strings.HasPrefix(b.ID, "build:") {
				a.drawButton(dst, b, 1)
			}
		}
	}
	a.drawStripTooltip(dst)
	a.drawOverlay(dst)
}

func (a *App) drawButton(dst *ebiten.Image, b Button, alpha float64) {
	u := a.u
	r := b.R
	hover := a.hoverBtn == b.ID
	down := hover && a.pressBtn == b.ID && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	bg, hi, lo := art.Slate, art.Steel, art.Navy
	if b.Primary {
		bg, hi, lo = art.DeepBlue, art.Blue, art.Navy
	}
	if b.Danger {
		bg, hi, lo = art.Maroon, art.Red, art.Plum
	}
	if hover && b.Enabled {
		bg, hi = hi, art.Gray
		if b.Primary {
			hi = art.Cyan
		}
		if b.Danger {
			hi = art.Pink
		}
	}
	if !b.Enabled {
		bg, hi, lo = art.Navy, art.Slate, art.Ink
	}
	fillRect(dst, r.X-u, r.Y-u, r.W+2*u, r.H+2*u, pm(art.Ink, alpha))
	fillRect(dst, r.X, r.Y, r.W, r.H, pm(bg, alpha))
	if down {
		fillRect(dst, r.X, r.Y, r.W, u, pm(lo, alpha))
	} else {
		fillRect(dst, r.X, r.Y, r.W, u, pm(hi, alpha))
		fillRect(dst, r.X, r.Y+r.H-u, r.W, u, pm(lo, alpha))
	}
	if b.Label == "" && b.Icon != "" {
		// An icon button.
		oy := 0
		if down {
			oy = u
		}
		cs := csAlpha(alpha)
		if !b.Enabled {
			cs.Scale(0.5, 0.5, 0.55, 1)
		}
		op := &ebiten.DrawImageOptions{ColorScale: cs}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(r.X+(r.W-11*u)/2), float64(r.Y+(r.H-11*u)/2+oy))
		dst.DrawImage(a.as.Icons[b.Icon], op)
		return
	}
	content := a.buttonWidth(b) - 12*u
	x := r.X + (r.W-content)/2
	y := r.Y + (r.H-7*u)/2
	if down {
		y += u
	}
	tc := art.White
	if !b.Enabled {
		tc = art.Gray
	}
	x = a.drawText(dst, b.Label, x, y, u, tc, alpha)
	if b.Cost > 0 {
		x += 3 * u
		ic, cc := "cr", art.Yellow
		if b.CostIcon != "" {
			ic, cc = b.CostIcon, art.Pink
		}
		a.drawIcon(dst, ic, x, y-2*u, u, alpha)
		x += 13 * u
		if !b.Enabled {
			cc = art.Red
		}
		x = a.drawText(dst, fmt.Sprint(b.Cost), x, y, u, cc, alpha)
	}
	if b.Key != "" {
		x += 5 * u
		kw := a.textWidth(b.Key, u) + 4*u
		fillRect(dst, x, r.Y+2*u, kw, r.H-4*u, pm(art.Ink, 0.55*alpha))
		kc := art.Gray
		if b.Primary || hover {
			kc = art.Silver
		}
		a.drawTextRaw(dst, b.Key, x+2*u, y, u, kc, alpha, false)
	}
}

func lerpCol(a, b color.RGBA, t float64) color.RGBA {
	t = clampF(t, 0, 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

func (a *App) drawTopBar(dst *ebiten.Image) {
	u := a.u
	w := a.world
	fillRect(dst, 0, 0, a.sw, a.barH, pm(art.Ink, 0.9))
	fillRect(dst, 0, a.barH-u, a.sw, u, pm(art.Navy, 1))
	ty := (a.barH - 7*u) / 2
	iy := (a.barH - 11*u) / 2
	x := 7 * u

	a.drawIcon(dst, "hp", x, iy, u, 1)
	x += 14 * u
	hpCol := lerpCol(art.White, art.Red, a.hpFlash)
	x = a.drawText(dst, pctText(w.BaseHP, w.BaseMaxHP), x, ty, u, hpCol, 1)
	x += 4 * u
	bw, bh := 34*u, 4*u
	by := (a.barH - bh) / 2
	fillRect(dst, x-u, by-u, bw+2*u, bh+2*u, art.Ink)
	fillRect(dst, x, by, bw, bh, art.Navy)
	f := float64(w.BaseHP) / float64(w.BaseMaxHP)
	hc := art.Lime
	if f < 0.35 {
		hc = art.Red
	} else if f < 0.65 {
		hc = art.Amber
	}
	fillRect(dst, x, by, int(float64(bw)*f+0.5), bh, lerpCol(hc, art.White, a.hpFlash*0.6))
	x += bw + 14*u

	a.drawIcon(dst, "cr", x, iy, u, 1)
	x += 14 * u
	cc := lerpCol(art.Yellow, art.White, a.crGain*0.8)
	cc = lerpCol(cc, art.Red, a.crFlash)
	x = a.drawText(dst, fmt.Sprint(w.Credits), x, ty, u, cc, 1)
	x += 14 * u

	a.drawIcon(dst, "wave", x, iy, u, 1)
	x += 14 * u
	wv := w.Wave
	if w.Phase == game.PhasePrep {
		wv++
	}
	wv = max(1, min(wv, w.TotalWaves()))
	a.drawText(dst, fmt.Sprintf("Wave %d/%d", wv, w.TotalWaves()), x, ty, u, art.White, 1)

	// Centre status.
	switch w.Phase {
	case game.PhasePrep:
		tw := a.centerTextWidth()
		total := tw
		if b := a.button("skip"); b != nil {
			total = tw + 8*u + b.R.W
		}
		cx := a.sw/2 - total/2
		col := art.White
		if w.Timer < 5 {
			col = lerpCol(art.White, art.Amber, 0.5+0.5*math.Sin(a.now*8))
		}
		a.drawIcon(dst, "time", cx, iy, u, 1)
		a.drawText(dst, a.prepText(), cx+14*u, ty, u, col, 1)
	case game.PhaseWave:
		s := a.waveText()
		a.drawText(dst, s, a.sw/2-a.textWidth(s, u)/2, ty, u, art.Silver, 1)
	}
	for _, b := range a.buttons {
		if topBarButton(b.ID) {
			a.drawButton(dst, b, 1)
		}
	}
}

func topBarButton(id string) bool {
	return id == "pause" || id == "speed" || id == "skip" || id == "guide"
}

// pctText shows base integrity in percent; a standing base never reads 0%.
func pctText(hp, maxHP int) string {
	p := 0
	if hp > 0 && maxHP > 0 {
		p = max(1, hp*100/maxHP)
	}
	return fmt.Sprintf("%d%%", p)
}

// decimal formats 1 as "1.0" and 1.15 as "1.15".
func decimal(v float64) string {
	s := strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// drawTooltip places a small panel above the object (or below if there is no room).
func (a *App) drawTooltip(dst *ebiten.Image, lines [][]Seg, sx, sy float64, avoid Rect) {
	u := a.u
	pw := 0
	for _, l := range lines {
		pw = max(pw, a.segsWidthIcons(l, u))
	}
	pw += 8 * u
	ph := len(lines)*11*u + 5*u
	off := (towerTopZ-art.PadH)*art.HZ*a.cam.Z + float64(3*u)
	x := int(sx) - pw/2
	y := int(sy-off) - ph
	if avoid.W > 0 && avoid.Y < int(sy) {
		y = avoid.Y - ph - 3*u
	}
	if y < a.barH+2*u {
		y = int(sy + 15*a.cam.Z + float64(3*u))
		if avoid.W > 0 && avoid.Y > int(sy) {
			y = avoid.Y + avoid.H + 3*u
		}
	}
	x = max(2*u, min(x, a.sw-pw-2*u))
	y = max(a.barH+2*u, min(y, a.sh-ph-2*u))
	a.panel(dst, Rect{x, y, pw, ph}, 1)
	for i, l := range lines {
		a.drawSegsIcons(dst, l, x+4*u, y+4*u+i*11*u, u, 1)
	}
}

// segIcons are segments drawn as icons: "@" is the credits icon, "◆" a
// crystal, "★" an earned star and "☆" a missing one.
var segIcons = map[string]string{"@": "cr", "◆": "crystal", "★": "star", "☆": "star0"}

func (a *App) segsWidthIcons(segs []Seg, sc int) int {
	w := 0
	for _, s := range segs {
		if _, ok := segIcons[s.S]; ok {
			w += 12 * sc
			continue
		}
		w += a.textWidth(s.S, sc) + sc
	}
	return w - sc
}

func (a *App) drawSegsIcons(dst *ebiten.Image, segs []Seg, x, y, sc int, alpha float64) int {
	for _, s := range segs {
		if ic, ok := segIcons[s.S]; ok {
			a.drawIcon(dst, ic, x, y-2*sc, sc, alpha)
			x += 12 * sc
			continue
		}
		x = a.drawText(dst, s.S, x, y, sc, s.Col, alpha)
	}
	return x
}

func (a *App) drawBanner(dst *ebiten.Image) {
	b := a.banner
	if b == nil {
		return
	}
	u := a.u
	al := smooth(0, 0.25, b.T) * (1 - smooth(b.Life-0.6, b.Life, b.T))
	slide := int((1 - easeOut(smooth(0, 0.35, b.T))) * float64(6*u))
	big := 2 * u
	y := a.barH + 14*u - slide
	if a.stripVisible() {
		y += 22 * u
	}
	tw := a.textWidth(b.Title, big)
	sw := a.textWidth(b.Sub, u)
	pw := max(tw, sw) + 24*u
	ph := 7*big + 11*u + 12*u
	fillRect(dst, a.sw/2-pw/2, y-5*u, pw, ph, pm(art.Ink, 0.55*al))
	a.drawText(dst, b.Title, a.sw/2-tw/2, y, big, b.Col, al)
	a.drawText(dst, b.Sub, a.sw/2-sw/2, y+7*big+5*u, u, art.Silver, al)
}

func (a *App) drawHint(dst *ebiten.Image) {
	w := a.world
	if w.Wave > 0 || w.Phase != game.PhasePrep || a.paused {
		return
	}
	u := a.u
	keys := "key 1"
	if n := a.buildKeys(); n > 1 {
		keys = fmt.Sprintf("keys 1-%d", n)
	}
	lines := []string{
		"Click a pad to build a turret (" + keys + "), click a turret to upgrade or sell it.",
		"Wheel: zoom  ·  Drag: pan  ·  Space: start the wave now  ·  I: guide  ·  Esc: pause",
	}
	pw := 0
	for _, l := range lines {
		pw = max(pw, a.textWidth(l, u))
	}
	pw += 12 * u
	ph := len(lines)*11*u + 7*u
	x, y := a.sw/2-pw/2, a.sh-ph-6*u
	al := 0.5 + 0.5*smooth(0, 0.6, a.now)
	a.panel(dst, Rect{x, y, pw, ph}, al)
	for i, l := range lines {
		col := art.Silver
		if i == 0 {
			col = art.White
		}
		a.drawText(dst, l, a.sw/2-a.textWidth(l, u)/2, y+5*u+i*11*u, u, col, al)
	}
}

func (a *App) dim(dst *ebiten.Image, al float64) {
	fillRect(dst, 0, 0, a.sw, a.sh, pm(art.Ink, 0.62*al))
}
