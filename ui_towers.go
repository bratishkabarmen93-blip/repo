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

const menuOpenTime = 0.15 // seconds the build menu takes to unfold

// menuHover is the build menu entry under the cursor, or nil.
func (a *App) menuHover() *game.TowerDef {
	if a.menuSlot < 0 {
		return nil
	}
	if id, ok := strings.CutPrefix(a.hoverBtn, "build:"); ok {
		return game.TowerDefs[id]
	}
	if a.menuKey >= 0 && a.menuKey < len(game.TowerOrder) && a.world.CanBuild(game.TowerOrder[a.menuKey]) {
		return game.TowerOrder[a.menuKey]
	}
	return nil
}

// buildMenu is the ring of turret options around an empty pad.
type buildMenu struct {
	items  []Button
	bounds Rect // the fully open menu
	cx, cy int  // ring centre on screen
	dx, dy int  // how far the ring was pushed to stay on screen
}

func (a *App) buildMenuLayout() (buildMenu, bool) {
	w := a.world
	if a.menuSlot < 0 || !w.Active() || a.modal() {
		return buildMenu{}, false
	}
	u := a.u
	P := float64(art.CellPx)
	c := a.lvl.Slots[a.menuSlot].Center()
	sx, sy := a.screenAt(c.X*P, c.Y*P, art.PadH+6)
	bw, bh := 32*u, 36*u
	// The ring leaves the pad and the turret preview visible at any zoom.
	r := max(37*u, int(18*a.cam.Z)+bh/2+3*u)
	arsenal := a.spec.Arsenal()
	n := len(arsenal)
	dirs := make([][2]float64, n)
	minX, minY, maxX, maxY := 0, 0, 0, 0
	for i := range dirs {
		// One option sits on top, two left and right, more go round the ring.
		ang := -math.Pi/2 + float64(i)*2*math.Pi/float64(n)
		if n == 2 {
			ang = math.Pi * float64(1-i)
		}
		dirs[i] = [2]float64{math.Cos(ang), math.Sin(ang)}
		x := int(math.Round(dirs[i][0]*float64(r))) - bw/2
		y := int(math.Round(dirs[i][1]*float64(r))) - bh/2
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x+bw), max(maxY, y+bh)
	}
	m := 3 * u
	cx := max(m-minX, min(int(sx), a.sw-m-maxX))
	cy := max(a.barH+m-minY, min(int(sy), a.sh-m-maxY))
	open := easeOut(clampF(a.menuT/menuOpenTime, 0, 1))
	rr := float64(r) * (0.6 + 0.4*open)
	bm := buildMenu{cx: cx, cy: cy, dx: cx - int(sx), dy: cy - int(sy), bounds: Rect{cx + minX, cy + minY, maxX - minX, maxY - minY}}
	for i, def := range arsenal {
		x := cx + int(math.Round(dirs[i][0]*rr)) - bw/2
		y := cy + int(math.Round(dirs[i][1]*rr)) - bh/2
		cost := def.Cost()
		bm.items = append(bm.items, Button{
			ID: "build:" + def.ID, R: Rect{x, y, bw, bh}, Label: def.Name, Key: strconv.Itoa(towerKey(def)),
			Cost: cost, Enabled: w.Credits >= cost, Icon: def.ID,
		})
	}
	return bm, true
}

// towerKey is the number key that builds a turret type.
func towerKey(d *game.TowerDef) int {
	for i, t := range game.TowerOrder {
		if t == d {
			return i + 1
		}
	}
	return 0
}

func (a *App) drawBuildMenu(dst *ebiten.Image) {
	bm, ok := a.buildMenuLayout()
	if !ok {
		return
	}
	u := a.u
	al := smooth(0, menuOpenTime, a.menuT)
	for _, b := range bm.items {
		lk := a.as.Towers[game.TowerDefs[b.Icon]]
		r := b.R
		hover := a.hoverBtn == b.ID
		down := hover && a.pressBtn == b.ID && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
		bg, edge := art.Navy, art.Slate
		if hover {
			bg, edge = art.Slate, art.Cyan
			if !b.Enabled {
				edge = art.Red
			}
		}
		fillRect(dst, r.X-u, r.Y-u, r.W+2*u, r.H+2*u, pm(art.Ink, al))
		fillRect(dst, r.X, r.Y, r.W, r.H, pm(bg, 0.94*al))
		fillRect(dst, r.X, r.Y, r.W, u, pm(edge, al))
		if hover {
			fillRect(dst, r.X, r.Y+r.H-u, r.W, u, pm(edge, al))
			fillRect(dst, r.X, r.Y, u, r.H, pm(edge, al))
			fillRect(dst, r.X+r.W-u, r.Y, u, r.H, pm(edge, al))
		}
		oy := 0
		if down {
			oy = u
		}
		img := lk.icon.img
		ib := img.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(r.X+(r.W-ib.Dx()*u)/2), float64(r.Y+3*u+oy))
		if !b.Enabled {
			op.ColorScale.Scale(0.45, 0.45, 0.55, 1)
		}
		op.ColorScale.ScaleAlpha(float32(al))
		dst.DrawImage(img, op)
		kc := art.Gray
		if hover {
			kc = art.White
		}
		a.drawText(dst, b.Key, r.X+3*u, r.Y+3*u, u, kc, al)
		cs := fmt.Sprint(b.Cost)
		cw := 12*u + a.textWidth(cs, u)
		x := r.X + (r.W-cw)/2
		y := r.Y + r.H - 10*u + oy
		a.drawIcon(dst, "cr", x, y-2*u, u, al)
		cc := art.Yellow
		if !b.Enabled {
			cc = art.Red
		}
		a.drawText(dst, cs, x+12*u, y, u, cc, al)
	}
	if def := a.menuHover(); def != nil {
		a.drawBuildTip(dst, def, bm)
	}
}

// drawBuildTip describes the hovered build option beside the menu.
func (a *App) drawBuildTip(dst *ebiten.Image, def *game.TowerDef, bm buildMenu) {
	u := a.u
	lk := a.as.Towers[def]
	st := def.Levels[0]
	stat := towerStatSegs(def, st, art.Gray, art.White)
	wrapW := max(a.segsWidthIcons(stat, u), 130*u)
	lines := [][]Seg{{{def.Name, art.White}, {"  " + def.Role, lk.accent}}}
	for _, l := range a.wrap(def.Desc, wrapW, u) {
		lines = append(lines, []Seg{{l, art.Silver}})
	}
	lines = append(lines, stat, towerTraits[def.ID])
	cost := def.Cost()
	cc := art.Yellow
	if a.world.Credits < cost {
		cc = art.Red
	}
	lines = append(lines, []Seg{{"Cost ", art.Gray}, {"@", cc}, {fmt.Sprint(cost), cc}, {"  Builds in ", art.Gray}, {seconds(st.Build), art.White}})
	if a.world.Credits < cost {
		lines = append(lines, []Seg{{"Not enough credits", art.Red}})
	}
	pw, ph := a.linesSize(lines)
	x := bm.bounds.X + bm.bounds.W + 4*u
	y := bm.cy - ph/2
	if x+pw > a.sw-2*u {
		x = bm.bounds.X - pw - 4*u
	}
	if x < 2*u {
		x = bm.cx - pw/2
		y = bm.bounds.Y + bm.bounds.H + 4*u
		if y+ph > a.sh-2*u {
			y = bm.bounds.Y - ph - 4*u
		}
	}
	x = max(2*u, min(x, a.sw-pw-2*u))
	y = max(a.barH+2*u, min(y, a.sh-ph-2*u))
	a.drawLines(dst, lines, x, y, 1)
}

func (a *App) linesSize(lines [][]Seg) (int, int) {
	u := a.u
	pw := 0
	for _, l := range lines {
		pw = max(pw, a.segsWidthIcons(l, u))
	}
	return pw + 8*u, len(lines)*11*u + 5*u
}

func (a *App) drawLines(dst *ebiten.Image, lines [][]Seg, x, y int, al float64) {
	u := a.u
	pw, ph := a.linesSize(lines)
	a.panel(dst, Rect{x, y, pw, ph}, al)
	for i, l := range lines {
		a.drawSegsIcons(dst, l, x+4*u, y+4*u+i*11*u, u, al)
	}
}

// towerButtons are the actions under the selected turret and the row they fill.
func (a *App) towerButtons() ([]Button, Rect) {
	t := a.sel
	if t == nil || !a.world.Active() || a.modal() {
		return nil, Rect{}
	}
	var bs []Button
	if t.Busy {
		bs = append(bs, Button{ID: "cancel", Label: "Cancel", Key: "X", Cost: t.WorkCost, Enabled: true})
	} else {
		up := Button{ID: "upgrade", Label: "Upgrade", Key: "U"}
		if t.MaxLevel() {
			up.Label, up.Key = "Max level", ""
		} else {
			if t.NextIsSuper() {
				up.Label = "Super"
			}
			up.Cost = t.UpgradeCost()
			up.Enabled = a.world.Credits >= up.Cost
		}
		bs = append(bs, up)
		if t.Def.Aims() && a.world.CanTarget() {
			bs = append(bs, Button{ID: "target", Label: "Target: " + t.Mode.String(), Key: "T", Enabled: true})
		}
		sell := Button{ID: "sell", Label: "Sell", Key: "X", Cost: t.SellValue(), Enabled: true}
		if a.sellArm == t.ID {
			sell.Label, sell.Danger = "Confirm", true
		}
		bs = append(bs, sell)
	}
	u := a.u
	gap, bh := 3*u, a.btnHeight()
	total := -gap
	for i := range bs {
		bs[i].R.W = a.buttonWidth(bs[i])
		total += bs[i].R.W + gap
	}
	sx, sy := a.screenAt(t.Pos.X*art.CellPx, t.Pos.Y*art.CellPx, art.PadH)
	y := int(sy + 15*a.cam.Z + float64(2*u))
	if y+bh > a.sh-2*u {
		y = int(sy-(towerTopZ-art.PadH)*art.HZ*a.cam.Z) - bh - 2*u
	}
	x := int(sx) - total/2
	x = max(2*u, min(x, a.sw-total-2*u))
	y = max(a.barH+2*u, min(y, a.sh-bh-2*u))
	row := Rect{x, y, total, bh}
	for i := range bs {
		bs[i].R = Rect{x, y, bs[i].R.W, bh}
		x += bs[i].R.W + gap
	}
	return bs, row
}

// towerStatSegs sums up one level of a turret type in a line.
func towerStatSegs(d *game.TowerDef, st game.TowerLevel, lc, vc color.RGBA) []Seg {
	rng := []Seg{{"  Range ", lc}, {decimal(st.Range), vc}}
	switch d.Attack {
	case game.AtkPulse:
		return append([]Seg{{"Slow ", lc}, {fmt.Sprintf("%d%%", int(math.Round(st.Slow*100))), vc}, {"  Damage ", lc}, {num(st.Damage), vc}}, rng...)
	case game.AtkFlame:
		return append([]Seg{{"Fire ", lc}, {num(st.DPS()) + "/s", vc}, {"  Burn ", lc}, {num(st.Burn) + "/s", vc}}, rng...)
	}
	return append([]Seg{{"Damage ", lc}, {num(st.Damage), vc}, {"  Rate ", lc}, {decimal(st.FireRate) + "/s", vc}}, rng...)
}

// towerTraits name what a turret type can hit and what makes it special.
var towerTraits = map[string][]Seg{
	"blaster":  {{"Ground only", art.Gray}},
	"cryo":     {{"Ground only · area slow", art.Gray}},
	"flamer":   {{"Ground only · area burn", art.Gray}},
	"railgun":  {{"Ground only · ", art.Gray}, {"ignores armor", art.Lilac}},
	"skyguard": {{"Air only", art.Amber}, {" · splash damage", art.Gray}},
}

var targetModeDesc = [game.NumTargetModes]string{
	"the enemy closest to the base", "the enemy furthest from the base", "the toughest enemy", "the nearest enemy",
}

func num(v float64) string { return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) }

func seconds(v float64) string { return num(v) + "s" }

// levelTag is "Lv 2/3" or "Super" in the turret's accent colour.
func (a *App) levelTag(t *game.Tower) (string, color.RGBA) {
	if t.HasSuper() {
		return "Super", a.as.Towers[t.Def].accent
	}
	return fmt.Sprintf("Lv %d/%d", t.Level+1, len(t.Def.Levels)), art.Cyan
}

// jobText names the construction job of a busy turret.
func jobText(t *game.Tower) string {
	switch {
	case t.Fresh:
		return "Building..."
	case t.JobLevel() >= len(t.Def.Levels):
		return "Installing " + t.Def.Super.Name + "..."
	}
	return fmt.Sprintf("Upgrading to Lv %d...", t.JobLevel()+1)
}

func jobSegs(t *game.Tower) []Seg {
	return []Seg{{jobText(t) + " ", art.Amber}, {fmt.Sprintf("%d%%", int(t.Progress()*100)), art.White}}
}

func (a *App) superLines(d *game.TowerDef, head string, wrapW int) [][]Seg {
	lk := a.as.Towers[d]
	lines := [][]Seg{{{head, lk.accent}, {d.Super.Name, art.White}}}
	for _, l := range a.wrap(d.Super.Desc, wrapW, a.u) {
		lines = append(lines, []Seg{{l, art.Silver}})
	}
	return lines
}

// towerLines describe the selected turret and the action under the cursor.
func (a *App) towerLines(t *game.Tower) [][]Seg {
	u := a.u
	tag, tc := a.levelTag(t)
	stat := towerStatSegs(t.Def, t.Stats(), art.Gray, art.White)
	lines := [][]Seg{{{t.Def.Name + " ", art.White}, {tag, tc}}, stat, towerTraits[t.Def.ID]}
	wrapW := max(a.segsWidthIcons(stat, u), 130*u)
	if t.HasSuper() {
		lines = append(lines, a.superLines(t.Def, "◆ ", wrapW)...)
	}
	if t.Busy {
		lines = append(lines, jobSegs(t))
		if a.hoverBtn == "cancel" {
			lines = append(lines, []Seg{{"Cancel for a full refund: ", art.Gray}, {"@", art.Yellow}, {fmt.Sprint(t.WorkCost), art.Yellow}})
		} else {
			lines = append(lines, []Seg{{"The turret is offline until it is done", art.Gray}})
		}
		return lines
	}
	sell := []Seg{{"@", art.Yellow}, {fmt.Sprint(t.SellValue()), art.Yellow}}
	switch a.hoverBtn {
	case "upgrade":
		if t.MaxLevel() {
			break
		}
		dur := t.Def.Super.Build
		if t.NextIsSuper() {
			lines = append(lines, a.superLines(t.Def, "Super: ", wrapW)...)
		} else {
			nx := t.NextStats()
			dur = nx.Build
			lines = append(lines, append([]Seg{{"Next: ", art.Lime}}, towerStatSegs(t.Def, nx, art.Green, art.Lime)...))
		}
		lines = append(lines, []Seg{{"Takes ", art.Gray}, {seconds(dur), art.White}, {" · offline while upgrading", art.Gray}})
		if a.world.Credits < t.UpgradeCost() {
			lines = append(lines, []Seg{{"Not enough credits", art.Red}})
		}
	case "sell":
		if a.sellArm == t.ID {
			lines = append(lines, append([]Seg{{"Click again to sell for ", art.Amber}}, sell...))
		} else {
			lines = append(lines, append(append([]Seg{{"Sell for ", art.Gray}}, sell...), Seg{" (70% of its cost)", art.Gray}))
		}
	case "target":
		lines = append(lines, []Seg{{"Aims at ", art.Gray}, {targetModeDesc[t.Mode], art.White}},
			[]Seg{{"Click or press T to switch", art.Gray}})
	}
	if a.sellArm == t.ID && a.hoverBtn != "sell" {
		lines = append(lines, append([]Seg{{"Press X again to sell for ", art.Amber}}, sell...))
	}
	return lines
}

// drawTowerUI draws the turret tooltips and the build menu.
func (a *App) drawTowerUI(dst *ebiten.Image) {
	if a.modal() {
		return
	}
	u := a.u
	w := a.world
	P := float64(art.CellPx)
	if t := a.sel; t != nil {
		_, row := a.towerButtons()
		sx, sy := a.screenAt(t.Pos.X*P, t.Pos.Y*P, art.PadH)
		a.drawTooltip(dst, a.towerLines(t), sx, sy, row)
		if b := a.button("sell"); b != nil && a.sellArm == t.ID {
			left := clampF(1-(a.now-a.sellT)/sellConfirmTime, 0, 1)
			fillRect(dst, b.R.X, b.R.Y+b.R.H+u, int(float64(b.R.W)*left), u, art.Red)
		}
	}
	if s := a.hoverSlot; s >= 0 && s != a.menuSlot {
		if t := w.SlotTower[s]; t == nil {
			c := a.lvl.Slots[s].Center()
			sx, sy := a.screenAt(c.X*P, c.Y*P, art.PadH)
			keys := "Click or press 1"
			if n := a.buildKeys(); n > 1 {
				keys = fmt.Sprintf("Click or press 1-%d", n)
			}
			lines := [][]Seg{{{"Build a turret", art.White}}, {{keys, art.Gray}}}
			a.drawTooltip(dst, lines, sx, sy, Rect{})
		} else if t != a.sel {
			tag, tc := a.levelTag(t)
			lines := [][]Seg{{{t.Def.Name + " ", art.White}, {tag, tc}}}
			if t.Busy {
				lines = append(lines, jobSegs(t))
			}
			sx, sy := a.screenAt(t.Pos.X*P, t.Pos.Y*P, art.PadH)
			a.drawTooltip(dst, lines, sx, sy, Rect{})
		}
	}
	a.drawBuildMenu(dst)
}
