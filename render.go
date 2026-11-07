package main

import (
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
	"outpost/internal/game"
)

var (
	bgColor = art.Hex(0x100c18)
	black   = color.RGBA{0, 0, 0, 255}
)

const (
	towerTopZ = art.PadH + art.PivotZ + art.MuzzleZ + 5
	muzzleH   = art.PadH + art.PivotZ + art.MuzzleZ
	// A point at height z casts its shadow this far along the ground (see art.Light).
	shadowDX = 0.861
	shadowDY = 0.444
)

func smooth(a, b, x float64) float64 { return game.Smoothstep(a, b, x) }

func easeOutBack(t float64) float64 {
	t = clampF(t, 0, 1)
	const c1, c3 = 1.70158, 2.70158
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func easeOut(t float64) float64 { return 1 - (1-t)*(1-t) }

func csAlpha(a float64) ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleAlpha(float32(clampF(a, 0, 1)))
	return c
}

func csCol(col color.RGBA, a float64) ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleWithColor(pm(col, a))
	return c
}

// at maps a world point (pixels, z up) to camera space (terrain image pixels).
func (a *App) at(x, y, z float64) (float64, float64) {
	px, py := art.Project(x, y, z)
	return px, py + float64(a.as.TI.OY)
}

// screenAt maps a world point to screen pixels.
func (a *App) screenAt(x, y, z float64) (float64, float64) { return a.cam.ToScreen(a.at(x, y, z)) }

// ground returns the terrain height at a world point.
func (a *App) ground(x, y float64) float64 { return a.as.TI.Height(x, y) }

// blit draws img with its anchor (ox, oy) on the world point, scaled by s around it.
func (a *App) blit(dst, img *ebiten.Image, x, y, z, ox, oy, s float64, cs ebiten.ColorScale, add bool) {
	op := &a.op
	*op = ebiten.DrawImageOptions{}
	op.GeoM.Translate(-ox, -oy)
	op.GeoM.Scale(s*a.cam.Z, s*a.cam.Z)
	sx, sy := a.screenAt(x, y, z)
	op.GeoM.Translate(sx, sy)
	op.ColorScale = cs
	op.Filter = ebiten.FilterPixelated
	if add {
		op.Blend = ebiten.BlendLighter
	}
	dst.DrawImage(img, op)
}

// put draws img centred on the world point.
func (a *App) put(dst, img *ebiten.Image, x, y, z, sx, sy float64, cs ebiten.ColorScale, add bool) {
	b := img.Bounds()
	op := &a.op
	*op = ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(sx*a.cam.Z, sy*a.cam.Z)
	px, py := a.screenAt(x, y, z)
	op.GeoM.Translate(px, py)
	op.ColorScale = cs
	op.Filter = ebiten.FilterPixelated
	if add {
		op.Blend = ebiten.BlendLighter
	}
	dst.DrawImage(img, op)
}

func (a *App) Draw(screen *ebiten.Image) {
	defer func() {
		if r := recover(); r != nil {
			a.err = crashed(r)
		}
	}()
	if a.sw == 0 {
		return
	}
	screen.Fill(bgColor)
	if a.scr == scrColony {
		a.drawColony(screen)
	} else {
		a.drawWorld(screen)
	}
	switch a.scr {
	case scrColony:
	case scrMenu:
		a.drawMenu(screen)
	case scrChapters:
		a.drawChapters(screen)
	case scrLevels:
		a.drawLevels(screen)
	default:
		a.drawUI(screen)
	}
	if a.popup != panelNone {
		a.drawPanel(screen)
	}
	a.drawNote(screen)
	a.drawFade(screen)
}

// drawNote shows the toast message at the bottom of the screen.
func (a *App) drawNote(dst *ebiten.Image) {
	if len(a.notes) == 0 || a.noteT <= 0 {
		return
	}
	u := a.u
	n := a.notes[0]
	if a.scr == scrColony && a.onBridge() {
		n.text = strings.TrimSuffix(n.text, bridgeHint)
	}
	al := smooth(0, 0.6, a.noteT) * smooth(noteTime, noteTime-0.3, a.noteT)
	w := 12*u + a.textWidth(n.text, u) + 12*u
	r := Rect{a.sw/2 - w/2, a.sh - 44*u, w, 17 * u}
	if a.scr == scrGame && a.world != nil && (a.world.Phase == game.PhaseWon || a.world.Phase == game.PhaseLost) {
		r.Y = a.barH + 4*u // the result panel takes the middle of the screen
	}
	a.panel(dst, r, al)
	a.drawIcon(dst, n.icon, r.X+6*u, r.Y+3*u, u, al)
	a.drawText(dst, n.text, r.X+19*u, r.Y+5*u, u, art.White, al)
}

const (
	kTower = iota
	kEnemy
	kCorpse
	kBase
	kGhost
	kWyrm
)

type drawItem struct {
	key  float64
	kind int
	i    int
}

// strokeEllipse draws a ground ring (screen space).
func strokeEllipse(dst *ebiten.Image, cx, cy, rx, ry, w float64, col color.RGBA) {
	var p vector.Path
	const n = 56
	for i := 0; i < n; i++ {
		t := float64(i) / n * 2 * math.Pi
		x, y := float32(cx+math.Cos(t)*rx), float32(cy+math.Sin(t)*ry)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(col)
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(w)}, op)
}

func (a *App) shadowLayer() *ebiten.Image {
	if a.shadowImg == nil || a.shadowImg.Bounds().Dx() != a.sw || a.shadowImg.Bounds().Dy() != a.sh {
		if a.shadowImg != nil {
			a.shadowImg.Deallocate()
		}
		a.shadowImg = ebiten.NewImage(a.sw, a.sh)
	}
	a.shadowImg.Clear()
	return a.shadowImg
}

func (a *App) drawWorld(dst *ebiten.Image) {
	cam, w, as := a.cam, a.world, a.as
	P := float64(art.CellPx)
	t := a.gameT

	a.drawGround(dst)

	// Live enemies; wyrms under the ice show through it.
	a.enemies = a.enemies[:0]
	for _, e := range w.Enemies {
		if e.Alive() {
			a.enemies = append(a.enemies, e)
		}
	}
	a.layWyrms()
	a.drawUnderIce(dst)

	// Goo decals fade out slowly.
	for _, d := range a.fx.Decals {
		al := 0.85 * (1 - smooth(d.Life-5, d.Life, d.T)) * smooth(0, 0.15, d.T)
		sx := 1.0
		if d.Flip {
			sx = -1
		}
		a.put(dst, d.Img, d.X, d.Y, d.Z, sx*d.S, d.S, csAlpha(al), false)
	}

	a.drawPatches(dst)

	// Alien burrow glow, brighter for a moment when an enemy crawls out.
	for i, rt := range a.lvl.Routes {
		bx, by := art.BurrowPos(a.lvl, rt.Spawn)
		pulse := 0.22 + 0.08*math.Sin(t*2.2+float64(i)*1.7) + 0.35*a.spawnPulse[min(i, len(a.spawnPulse)-1)]
		a.put(dst, as.Glow["purple"], bx, by, 2, 1.1, 0.8, csAlpha(pulse), true)
	}

	// Shadows of moving objects, merged in one layer so overlaps do not stack.
	sl := a.shadowLayer()
	for _, tw := range w.Towers {
		a.towerShadow(sl, tw)
	}
	a.wyrmShadows(sl)
	for _, e := range a.enemies {
		if isWyrm(e.Def) {
			continue
		}
		x, y := e.Pos.X*P, e.Pos.Y*P
		lk, spr, s, al := a.enemySprite(e)
		if lk.hover > 0 {
			// Flying: the shadow falls further along the light and is a bit fainter.
			dz := a.hoverZ(e, lk)
			x, y = x+dz*shadowDX, y+dz*shadowDY
			al *= 0.75
		}
		a.blit(sl, spr.sh, x, y, a.ground(x, y), spr.sx, spr.sy, s, csAlpha(al), false)
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(art.ShadowAlpha)
	dst.DrawImage(sl, op)

	// Ground markers and rings.
	a.drawMarkers(dst)
	a.drawRingFX(dst)

	// Objects back to front.
	items := a.items[:0]
	for i, tw := range w.Towers {
		items = append(items, drawItem{tw.Pos.Y * P, kTower, i})
	}
	items = a.wyrmItems(items)
	for i, e := range a.enemies {
		if isWyrm(e.Def) {
			continue
		}
		k := e.Pos.Y * P
		if e.Def.Flying {
			k += 10 // flies above whatever stands right next to it
		}
		items = append(items, drawItem{k, kEnemy, i})
	}
	for i, c := range a.fx.Corpses {
		items = append(items, drawItem{c.Y, kCorpse, i})
	}
	bc := a.lvl.BaseCenter()
	items = append(items, drawItem{bc.Y*P + 0.5, kBase, 0})
	if g, _ := a.ghost(); g >= 0 {
		items = append(items, drawItem{a.lvl.Slots[g].Center().Y * P, kGhost, g})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].key < items[j].key })
	a.items = items
	for _, it := range items {
		switch it.kind {
		case kTower:
			a.drawTower(dst, w.Towers[it.i])
		case kEnemy:
			a.drawEnemy(dst, a.enemies[it.i])
		case kCorpse:
			a.drawCorpse(dst, &a.fx.Corpses[it.i])
		case kBase:
			a.drawBase(dst)
		case kGhost:
			_, def := a.ghost()
			a.drawGhost(dst, it.i, def)
		case kWyrm:
			a.drawWyrmPart(dst, it.i)
		}
	}

	a.drawBeams(dst)

	a.drawRails(dst)
	a.drawProjectiles(dst)

	a.drawParticles(dst)

	// Health bars above damaged enemies (the boss has its own bar in the HUD).
	for _, e := range a.enemies {
		if e.HP >= e.MaxHP || e.Def.Boss {
			continue
		}
		lk := as.Looks[e.Def]
		f := e.HP / e.MaxHP
		al := smooth(0, 0.55, e.Left())
		x, y := e.Pos.X*P, e.Pos.Y*P
		sx, sy := a.screenAt(x, y, a.enemyZ(e, lk)+lk.topZ+3)
		a.drawHPBar(dst, sx, sy, lk.barW*cam.Z, math.Max(2, 1.8*cam.Z), f, e.Def.Armor > 0, e.HealT < 0.4, al)
	}

	a.drawFloaters(dst)
}

// drawGround draws the baked ground of the map; the plateau floats over the
// background with a soft shadow.
func (a *App) drawGround(dst *ebiten.Image) {
	cam := a.cam
	x0, y0 := cam.ToScreen(0, float64(a.as.TI.OY))
	x1, y1 := cam.ToScreen(cam.MapW, cam.MapH)
	for i := 3; i >= 1; i-- {
		o := float32(float64(i) * 3 * cam.Z)
		vector.FillRect(dst, float32(x0)+o, float32(y0)+o*1.5, float32(x1-x0), float32(y1-y0)-o, pm(black, 0.15), false)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(cam.Z, cam.Z)
	tx, ty := cam.ToScreen(0, 0)
	op.GeoM.Translate(tx, ty)
	op.Filter = ebiten.FilterPixelated
	dst.DrawImage(a.as.Terrain, op)
}

// drawRingFX draws the expanding rings on the ground.
func (a *App) drawRingFX(dst *ebiten.Image) {
	for _, r := range a.fx.Rings {
		k := r.T / r.Life
		e := 1 - (1-k)*(1-k)
		rad := (r.R0 + (r.R1-r.R0)*e) * a.cam.Z
		sx, sy := a.screenAt(r.X, r.Y, r.Z)
		strokeEllipse(dst, sx, sy, rad, rad*art.GY, r.Width*a.cam.Z*(1-0.5*k), pm(r.Col, 0.8*(1-k)))
	}
}

// drawParticles draws the particles: solid ones first, glowing ones on top.
func (a *App) drawParticles(dst *ebiten.Image) {
	as := a.as
	for pass := 0; pass < 2; pass++ {
		for _, p := range a.fx.Parts {
			if p.Glow != (pass == 1) {
				continue
			}
			k := p.T / p.Life
			sz := p.Size + (p.Size1-p.Size)*k
			al := 1 - k*k
			if p.Smoke {
				al = 0.6 * smooth(0, 0.15, k) * (1 - k)
			}
			if p.Plus {
				a.put(dst, as.Plus, p.X, p.Y, p.Z, 1, 1, csCol(p.Col, al), p.Glow)
				continue
			}
			if p.Flake {
				a.put(dst, as.Flake, p.X, p.Y, p.Z, sz, sz, csCol(p.Col, al), p.Glow)
				continue
			}
			a.put(dst, as.White, p.X, p.Y, p.Z, sz, sz, csCol(p.Col, al), p.Glow)
		}
	}
}

// drawFloaters draws the rising texts.
func (a *App) drawFloaters(dst *ebiten.Image) {
	for _, fl := range a.fx.Floaters {
		k := fl.T / fl.Life
		sx, sy := a.screenAt(fl.X, fl.Y, fl.Z)
		sy -= 12 * a.cam.Z * easeOut(k)
		al := 1 - smooth(0.6, 1, k)
		sc := a.u
		if fl.Big {
			sc += max(1, a.u/2)
		}
		tw := a.textWidth(fl.Text, sc)
		a.drawText(dst, fl.Text, int(sx)-tw/2, int(sy)-4*sc, sc, fl.Col, al)
	}
}

// drawHPBar draws a small health bar centred at (sx, sy) with an optional armour badge.
func (a *App) drawHPBar(dst *ebiten.Image, sx, sy, bw, bh, f float64, armored, healed bool, al float64) {
	bx := sx - bw/2
	o := math.Max(1, a.cam.Z*0.6)
	vector.FillRect(dst, float32(bx-o), float32(sy-o), float32(bw+2*o), float32(bh+2*o), pm(art.Ink, 0.85*al), false)
	col := art.Lime
	if f < 0.3 {
		col = art.Red
	} else if f < 0.6 {
		col = art.Amber
	}
	if healed {
		col = lerpCol(col, art.White, 0.45)
	}
	vector.FillRect(dst, float32(bx), float32(sy), float32(bw*f), float32(bh), pm(col, al), false)
	if armored {
		img := a.as.Shield
		b := img.Bounds()
		op := &a.op
		*op = ebiten.DrawImageOptions{}
		k := math.Max(1, math.Round(a.cam.Z*0.75))
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(math.Round(bx-o-float64(b.Dx())*k), math.Round(sy+bh/2-float64(b.Dy())*k/2))
		op.ColorScale = csAlpha(al)
		dst.DrawImage(img, op)
	}
}

// hoverZ is the flight height of a flying enemy, with a gentle bob.
func (a *App) hoverZ(e *game.Enemy, lk *enemyLook) float64 {
	if lk.hover == 0 {
		return 0
	}
	return lk.hover + 1.6*math.Sin(a.gameT*2.4+float64(e.ID)*1.3)
}

// enemyZ is the height of an enemy's origin (ground level, or flight height).
func (a *App) enemyZ(e *game.Enemy, lk *enemyLook) float64 {
	P := float64(art.CellPx)
	return a.ground(e.Pos.X*P, e.Pos.Y*P) + a.hoverZ(e, lk)
}

// enemyFrame picks the animation frame of an enemy.
func (a *App) enemyFrame(e *game.Enemy, lk *enemyLook) int {
	if lk.fps > 0 {
		return int(a.gameT*lk.fps+float64(e.ID)*1.7) % lk.frames
	}
	return int(e.Dist*lk.perCell+float64(e.ID%5)) % lk.frames
}

// enemySprite picks the frame of an enemy and its emerge/arrive fade.
func (a *App) enemySprite(e *game.Enemy) (*enemyLook, sprite, float64, float64) {
	lk := a.as.Looks[e.Def]
	h, ok := a.heads[e.ID]
	if !ok {
		h = e.Dir.Angle()
	}
	em := smooth(0, 0.35, e.Age)
	al := em * smooth(0, 0.55, e.Left())
	return lk, lk.spr[angleIndex(h, enemyAngles)][a.enemyFrame(e, lk)], 0.65 + 0.35*em, al
}

func (a *App) drawEnemy(dst *ebiten.Image, e *game.Enemy) {
	P := float64(art.CellPx)
	x, y := e.Pos.X*P, e.Pos.Y*P
	lk, spr, s, al := a.enemySprite(e)
	z := a.enemyZ(e, lk)
	a.blit(dst, spr.img, x, y, z, spr.ox, spr.oy, s, csAlpha(al), false)
	if e.HitT < 0.1 {
		// Big armoured robots are hit all the time: keep their flash subtle.
		k := 0.8
		if lk.robot {
			k = 0.4
		}
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.White, (1-e.HitT/0.1)*k*al), true)
	}
	if e.HealT < 0.45 {
		k := 1 - e.HealT/0.45
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.Lime, k*0.45*al), true)
	}
	switch {
	case e.FreezeT > 0:
		// Frozen solid: an icy coat over the model.
		k := smooth(0, 0.25, e.FreezeT)
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.Hex(0xbfefff), 0.55*k*al), false)
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.Cyan, 0.25*k*al), true)
	case e.SlowT > 0:
		k := smooth(0, 0.3, e.SlowT)
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.Blue, 0.32*k*al), true)
	}
	if e.BurnT > 0 {
		k := smooth(0, 0.3, e.BurnT) * (0.75 + 0.25*math.Sin(a.gameT*21+float64(e.ID)))
		a.blit(dst, spr.sil, x, y, z, spr.ox, spr.oy, s, csCol(art.Orange, 0.22*k*al), true)
	}
	if e.Def.Heal != nil {
		// The repair emitter under the drone glows, brighter right after a pulse.
		k := 0.35 + 0.65*math.Max(0, 1-e.PulseT/0.5)
		a.put(dst, a.as.Glow["green"], x, y, z+lk.bodyZ-2, 0.45, 0.4, csAlpha(0.55*k*al), true)
	}
}

// drawCorpse draws the pop of a dying alien or the cooling wreck of a robot.
func (a *App) drawCorpse(dst *ebiten.Image, c *Corpse) {
	spr := c.Look.spr[c.Heading][c.Frame]
	if c.Spr != nil {
		spr = *c.Spr
	}
	if c.T < c.Hold {
		// Waiting for its turn to burst, glowing brighter and brighter.
		h := c.T / c.Hold
		a.blit(dst, spr.img, c.X, c.Y, c.Z, spr.ox, spr.oy, 1, csAlpha(1), false)
		a.blit(dst, spr.sil, c.X, c.Y, c.Z, spr.ox, spr.oy, 1, csCol(art.Hex(0xbfefff), 0.2+0.45*h*h), true)
		return
	}
	k := (c.T - c.Hold) / c.Life
	if c.Wreck {
		// Burnt-out wreck: dark, sinking into the ground and fading.
		var cs ebiten.ColorScale
		v := float32(0.32 * (1 - 0.4*k))
		al := float32(1 - smooth(0.55, 1, k))
		cs.Scale(v*al, v*0.85*al, v*0.85*al, al)
		z := c.Z - 5*smooth(0.3, 1, k)
		a.blit(dst, spr.img, c.X, c.Y, z, spr.ox, spr.oy, 1, cs, false)
		if k < 0.35 {
			a.blit(dst, spr.sil, c.X, c.Y, z, spr.ox, spr.oy, 1, csCol(art.Orange, 0.7*(1-k/0.35)), true)
		}
		return
	}
	s := 1 + 0.35*k
	flash := c.Flash
	if flash.A == 0 {
		flash = art.Yellow
	}
	a.blit(dst, spr.img, c.X, c.Y, c.Z, spr.ox, spr.oy, s, csAlpha(1-k), false)
	a.blit(dst, spr.sil, c.X, c.Y, c.Z, spr.ox, spr.oy, s, csCol(flash, (1-k)*0.9), true)
}

// drawBeams draws repair rays from drones to the enemies they patch up.
func (a *App) drawBeams(dst *ebiten.Image) {
	if len(a.fx.Beams) == 0 {
		return
	}
	P := float64(art.CellPx)
	find := func(id int) *game.Enemy {
		for _, e := range a.enemies {
			if e.ID == id {
				return e
			}
		}
		return nil
	}
	for _, b := range a.fx.Beams {
		src, dstE := find(b.From), find(b.To)
		if src == nil || dstE == nil {
			continue
		}
		ls, ld := a.as.Looks[src.Def], a.as.Looks[dstE.Def]
		x0, y0 := a.screenAt(src.Pos.X*P, src.Pos.Y*P, a.enemyZ(src, ls)+ls.bodyZ-2)
		x1, y1 := a.screenAt(dstE.Pos.X*P, dstE.Pos.Y*P, a.enemyZ(dstE, ld)+ld.bodyZ)
		k := b.T / b.Life
		al := (1 - k) * smooth(0, 0.08, b.T)
		w := float32(math.Max(1, a.cam.Z*0.9))
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), w*2.2, pm(art.Green, 0.35*al), true)
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), w, pm(art.Lime, 0.9*al), true)
	}
}

// boltZ interpolates a bolt's height from the muzzle to the target's body.
func (a *App) boltZ(p *game.Projectile) float64 {
	P := float64(art.CellPx)
	bodyZ := 4.5
	if lk := a.as.Looks[p.Target]; lk != nil {
		bodyZ = lk.bodyZ + lk.hover
	}
	zEnd := a.ground(p.Aim.X*P, p.Aim.Y*P) + bodyZ
	done := p.Pos.Dist(p.Start)
	left := p.Aim.Dist(p.Pos)
	f := 1.0
	if done+left > 1e-6 {
		f = done / (done + left)
	}
	return muzzleH + (zEnd-muzzleH)*f
}

func (a *App) drawBase(dst *ebiten.Image) {
	as, w := a.as, a.world
	P := float64(art.CellPx)
	t := a.gameT
	bc := a.lvl.BaseCenter()
	cx, cy := bc.X*P, bc.Y*P
	lost := w.Phase == game.PhaseLost
	var baseCS ebiten.ColorScale
	if lost {
		baseCS.Scale(0.45, 0.4, 0.42, 1)
	}
	b := as.baseOf(a.lvl)
	a.blit(dst, b.img, cx, cy, 0, b.ox, b.oy, 1, baseCS, false)
	ri := angleIndex(t*1.3, radarAngles)
	if lost {
		ri = angleIndex(0.7, radarAngles)
	}
	r := as.Radar[ri]
	rp := art.RadarPos
	a.blit(dst, r.img, cx+rp.X, cy+rp.Y, rp.Z, r.ox, r.oy, 1, baseCS, false)
	if !lost {
		for i, l := range art.BaseLights {
			on := 0.5 + 0.5*math.Sin(t*2.6+float64(i)*2.1)
			on = smooth(0.35, 0.8, on)
			a.put(dst, as.White, cx+l.X, cy+l.Y, l.Z, 2.4, 1.6, csCol(art.Lime, 0.25+0.75*on), false)
			a.put(dst, as.Glow["green"], cx+l.X, cy+l.Y, l.Z, 0.35, 0.3, csAlpha(0.5*on), true)
		}
		dome := 0.12 + 0.06*math.Sin(t*1.7)
		if w.Phase == game.PhaseWon {
			dome = 0.35 + 0.2*math.Sin(a.now*5)
		}
		d := art.DomeCenter
		a.put(dst, as.Glow["cyan"], cx+d.X, cy+d.Y, d.Z, 1.4, 1.2, csAlpha(dome), true)
	}
	if a.baseFlash > 0 {
		a.blit(dst, b.sil, cx, cy, 0, b.ox, b.oy, 1, csCol(art.Red, 0.55*a.baseFlash), false)
	}
}

func ringRadius(rangeCells float64) int { return int(math.Round(rangeCells * art.CellPx)) }

func (a *App) ring(dst *ebiten.Image, x, y, rangeCells float64, col color.RGBA, al float64) {
	m := a.as.Ring(ringRadius(rangeCells), col)
	a.blit(dst, m.img, x, y, 0, m.ox, m.oy, 1, csAlpha(al), false)
}

func (a *App) padMarker(dst *ebiten.Image, x, y float64, cs ebiten.ColorScale) {
	m := a.as.PadMarker
	a.blit(dst, m.img, x, y, art.PadH, m.ox, m.oy, 1, cs, false)
}

// ghost is the pad that shows a build preview and the turret type it shows, or -1.
func (a *App) ghost() (int, *game.TowerDef) {
	if !a.world.Active() || a.menuSlot < 0 {
		return -1, nil
	}
	if def := a.menuHover(); def != nil {
		return a.menuSlot, def
	}
	return -1, nil
}

func (a *App) drawMarkers(dst *ebiten.Image) {
	w := a.world
	P := float64(art.CellPx)
	for _, tw := range w.Towers {
		if tw.Busy {
			a.drawJobRing(dst, tw)
		}
	}
	if a.sel != nil {
		t := a.sel
		x, y := t.Pos.X*P, t.Pos.Y*P
		a.ring(dst, x, y, t.Stats().Range, art.Cyan, 0.9)
		if a.hoverBtn == "upgrade" && !t.Busy && !t.MaxLevel() && t.NextStats().Range > t.Stats().Range {
			a.ring(dst, x, y, t.NextStats().Range, art.Lime, 0.9)
		}
		p := 0.75 + 0.25*math.Sin(a.now*5)
		a.padMarker(dst, x, y, csCol(art.Cyan, p))
	}
	if a.menuSlot >= 0 && w.Active() {
		c := a.lvl.Slots[a.menuSlot].Center()
		x, y := c.X*P, c.Y*P
		if def := a.menuHover(); def != nil {
			col := art.White
			if w.Credits < def.Cost() {
				col = art.Red
			}
			a.ring(dst, x, y, def.Levels[0].Range, col, 0.75)
		}
		p := 0.75 + 0.25*math.Sin(a.now*6)
		a.padMarker(dst, x, y, csCol(art.Cyan, p))
	}
	if a.hoverSlot < 0 || !w.Active() || a.hoverSlot == a.menuSlot {
		return
	}
	c := a.lvl.Slots[a.hoverSlot].Center()
	x, y := c.X*P, c.Y*P
	if tw := w.SlotTower[a.hoverSlot]; tw != nil {
		if tw != a.sel {
			a.ring(dst, x, y, tw.Stats().Range, art.White, 0.6)
			a.padMarker(dst, x, y, csAlpha(0.6))
		}
		return
	}
	p := 0.7 + 0.3*math.Sin(a.now*6)
	a.padMarker(dst, x, y, csCol(art.White, p))
}
