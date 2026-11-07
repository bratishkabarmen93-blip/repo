package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
	"outpost/internal/game"
)

// headAngle is the drawn heading of a turret head: aimers follow the game,
// the cryo spins on its own.
func (a *App) headAngle(tw *game.Tower) float64 {
	if !tw.Def.Aims() {
		return a.spin[tw.ID]
	}
	return tw.Angle
}

// recoil is how far a head kicks back after a shot (world pixels).
func recoil(tw *game.Tower) float64 {
	switch tw.Def.Attack {
	case game.AtkRail:
		if tw.ShotT < 0.3 {
			return (1 - tw.ShotT/0.3) * 3.4
		}
	case game.AtkBolt:
		if tw.ShotT < 0.14 {
			return (1 - tw.ShotT/0.14) * 2.2
		}
	case game.AtkMissile:
		if tw.ShotT < 0.18 {
			return (1 - tw.ShotT/0.18) * 1.2
		}
	}
	return 0
}

// towerHead returns the head anchor (with recoil) of a turret.
func towerHead(tw *game.Tower) (float64, float64) {
	P := float64(art.CellPx)
	rec := recoil(tw)
	return tw.Pos.X*P - math.Cos(tw.Angle)*rec, tw.Pos.Y*P - math.Sin(tw.Angle)*rec
}

func towerScale(tw *game.Tower) float64 {
	if tw.BuiltT < 0.3 {
		return 0.92 + 0.08*easeOutBack(tw.BuiltT/0.3)
	}
	return 1
}

// towerShadow adds a turret's shadow to the shadow layer.
func (a *App) towerShadow(sl *ebiten.Image, tw *game.Tower) {
	lk := a.as.Towers[tw.Def]
	P := float64(art.CellPx)
	x, y := tw.Pos.X*P, tw.Pos.Y*P
	lv, cs := tw.Level, ebiten.ColorScale{}
	if tw.Busy && tw.Fresh {
		// A hologram casts a shadow only as it becomes solid.
		cs = csAlpha(tw.Progress())
	}
	s := towerScale(tw)
	b := lk.base[lv]
	a.blit(sl, b.sh, x, y, art.PadH, b.sx, b.sy, s, cs, false)
	hx, hy := towerHead(tw)
	h := lk.headAt(lv, a.headAngle(tw))
	a.blit(sl, h.sh, hx, hy, art.PadH, h.sx, h.sy, s, cs, false)
}

// blitPart draws the rows of a sprite below (or above) the image row of a world height
// cut above the anchor, for the bottom-up construction effect.
func (a *App) blitPart(dst *ebiten.Image, img *ebiten.Image, sp sprite, x, y, z, cut float64, below bool, cs ebiten.ColorScale, add bool) {
	b := img.Bounds()
	r := int(math.Round(sp.oy - cut*art.HZ))
	r = max(0, min(r, b.Dy()))
	if below {
		if r >= b.Dy() {
			return
		}
		a.blit(dst, img.SubImage(image.Rect(0, r, b.Dx(), b.Dy())).(*ebiten.Image), x, y, z, sp.ox, sp.oy-float64(r), 1, cs, add)
		return
	}
	if r <= 0 {
		return
	}
	a.blit(dst, img.SubImage(image.Rect(0, 0, b.Dx(), r)).(*ebiten.Image), x, y, z, sp.ox, sp.oy, 1, cs, add)
}

func (a *App) drawTower(dst *ebiten.Image, tw *game.Tower) {
	as := a.as
	lk := as.Towers[tw.Def]
	P := float64(art.CellPx)
	x, y := tw.Pos.X*P, tw.Pos.Y*P
	z := art.PadH
	ang := a.headAngle(tw)
	hx, hy := towerHead(tw)
	if tw.Busy {
		a.drawConstruction(dst, tw, lk, x, y, ang)
		return
	}
	s := towerScale(tw)
	lv := tw.Level
	base, head := lk.base[lv], lk.headAt(lv, ang)
	a.blit(dst, base.img, x, y, z, base.ox, base.oy, s, ebiten.ColorScale{}, false)
	a.blit(dst, head.img, hx, hy, z, head.ox, head.oy, s, ebiten.ColorScale{}, false)
	if tw.HasSuper() {
		// Supers shimmer softly in their accent colour.
		k := 0.5 + 0.5*math.Sin(a.gameT*2.6+float64(tw.ID))
		a.blit(dst, head.sil, hx, hy, z, head.ox, head.oy, s, csCol(lk.accent, 0.05+0.07*k), true)
	}
	if tw.BuiltT < 0.35 {
		k := 0.6 * (1 - tw.BuiltT/0.35)
		a.blit(dst, base.sil, x, y, z, base.ox, base.oy, s, csCol(art.White, k), true)
		a.blit(dst, head.sil, hx, hy, z, head.ox, head.oy, s, csCol(art.White, k), true)
	}
	tx, ty, tz := lk.tip(tw, lv)
	tz += z
	switch tw.Def.Attack {
	case game.AtkBolt:
		if tw.ShotT < 0.07 {
			k := 1 - tw.ShotT/0.07
			tx, ty = tx+math.Cos(tw.Angle)*1.5, ty+math.Sin(tw.Angle)*1.5
			a.put(dst, as.Glow["cyan"], tx, ty, tz, 0.6, 0.6, csAlpha(0.8*k), true)
			a.put(dst, as.Muzzle, tx, ty, tz, 0.7+0.4*k, 0.7+0.4*k, csAlpha(k), true)
		}
	case game.AtkRail:
		if tw.ShotT < 0.18 {
			k := 1 - tw.ShotT/0.18
			a.put(dst, as.Glow["violet"], tx, ty, tz, 0.9, 0.9, csAlpha(k), true)
			a.put(dst, as.Glow["white"], tx, ty, tz, 0.35, 0.35, csAlpha(k), true)
		} else if tw.TargetID != 0 && tw.Cooldown < 0.6 {
			// Charging up for the next shot.
			k := 1 - math.Max(0, tw.Cooldown)/0.6
			a.put(dst, as.Glow["violet"], tx, ty, tz, 0.2+0.35*k, 0.2+0.35*k, csAlpha(0.25+0.5*k), true)
		}
	case game.AtkMissile:
		if tw.ShotT < 0.12 {
			k := 1 - tw.ShotT/0.12
			a.put(dst, as.Glow["orange"], tx, ty, tz, 0.7, 0.7, csAlpha(0.8*k), true)
		}
	case game.AtkFlame:
		// The pilot light flickers; it flares while the flamer fires.
		k := 0.35 + 0.15*math.Sin(a.now*23+float64(tw.ID))
		if tw.ShotT < 0.25 {
			k = 0.9
		}
		a.put(dst, as.Glow["orange"], tx, ty, tz-1.5, 0.3+0.2*k, 0.3+0.2*k, csAlpha(k), true)
	case game.AtkPulse:
		k := 0.15 + 0.08*math.Sin(a.gameT*3+float64(tw.ID))
		if tw.ShotT < 0.6 {
			k += 0.75 * (1 - tw.ShotT/0.6)
		}
		a.put(dst, as.Glow["ice"], x, y, tz, 0.55, 0.5, csAlpha(k), true)
	}
}

// drawConstruction shows a turret being built bottom-up out of a hologram, or an
// upgrade replacing the old model from the bottom.
func (a *App) drawConstruction(dst *ebiten.Image, tw *game.Tower, lk *towerLook, x, y, ang float64) {
	z := art.PadH
	p := tw.Progress()
	nl := tw.JobLevel()
	top := lk.topZ[nl] + 1
	cut := easeInOut(p) * top
	nb, nh := lk.base[nl], lk.headAt(nl, ang)
	flick := 0.85 + 0.15*math.Sin(a.now*19+float64(tw.ID))
	holo := art.Cyan
	if !tw.Fresh {
		ob, oh := lk.base[tw.Level], lk.headAt(tw.Level, ang)
		a.blitPart(dst, ob.img, ob, x, y, z, cut, false, ebiten.ColorScale{}, false)
		a.blitPart(dst, oh.img, oh, x, y, z, cut, false, ebiten.ColorScale{}, false)
		a.blitPart(dst, ob.sil, ob, x, y, z, cut, false, csCol(holo, 0.12*flick), true)
		a.blitPart(dst, oh.sil, oh, x, y, z, cut, false, csCol(holo, 0.12*flick), true)
	} else {
		a.blitPart(dst, nb.sil, nb, x, y, z, cut, false, csCol(holo, 0.2*flick), true)
		a.blitPart(dst, nh.sil, nh, x, y, z, cut, false, csCol(holo, 0.2*flick), true)
	}
	a.blitPart(dst, nb.img, nb, x, y, z, cut, true, ebiten.ColorScale{}, false)
	a.blitPart(dst, nh.img, nh, x, y, z, cut, true, ebiten.ColorScale{}, false)
	a.blitPart(dst, nb.sil, nb, x, y, z, cut, true, csCol(holo, 0.18), true)
	a.blitPart(dst, nh.sil, nh, x, y, z, cut, true, csCol(holo, 0.18), true)
	// The build line.
	w := (float64(max(nb.img.Bounds().Dx(), nh.img.Bounds().Dx()))/2 - 2) * a.cam.Z
	sx, sy := a.screenAt(x, y, z+cut)
	lw := float32(math.Max(1, a.cam.Z))
	vector.StrokeLine(dst, float32(sx-w), float32(sy), float32(sx+w), float32(sy), lw*2.5, pm(holo, 0.25*flick), true)
	vector.StrokeLine(dst, float32(sx-w*0.8), float32(sy), float32(sx+w*0.8), float32(sy), lw, pm(art.White, 0.8*flick), true)
}

func easeInOut(t float64) float64 { return t * t * (3 - 2*t) }

// jobColour is the colour of a construction job: build, upgrade or super.
func jobColour(tw *game.Tower, lk *towerLook) color.RGBA {
	switch {
	case tw.Fresh:
		return art.Cyan
	case tw.JobLevel() == len(tw.Def.Levels):
		return lk.accent
	}
	return art.Amber
}

// drawJobRing draws the progress arc of a construction job around its pad.
func (a *App) drawJobRing(dst *ebiten.Image, tw *game.Tower) {
	P := float64(art.CellPx)
	lk := a.as.Towers[tw.Def]
	sx, sy := a.screenAt(tw.Pos.X*P, tw.Pos.Y*P, art.PadH)
	rx := 15.5 * a.cam.Z
	ry := rx * art.GY
	w := math.Max(1.5, 1.6*a.cam.Z)
	strokeEllipse(dst, sx, sy, rx, ry, w+2, pm(art.Ink, 0.55))
	p := tw.Progress()
	if p <= 0 {
		return
	}
	var path vector.Path
	n := max(2, int(48*p))
	for i := 0; i <= n; i++ {
		t := -math.Pi/2 + float64(i)/float64(n)*p*2*math.Pi
		px, py := float32(sx+math.Cos(t)*rx), float32(sy+math.Sin(t)*ry)
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(pm(jobColour(tw, lk), 0.95))
	vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: float32(w), LineCap: vector.LineCapRound}, op)
}

// drawGhost previews a turret type on an empty pad.
func (a *App) drawGhost(dst *ebiten.Image, slot int, def *game.TowerDef) {
	P := float64(art.CellPx)
	lk := a.as.Towers[def]
	c := a.lvl.Slots[slot].Center()
	x, y := c.X*P, c.Y*P
	al := 0.65
	if a.world.Credits < def.Cost() {
		al = 0.3
	}
	ang := -math.Pi / 2
	if near := a.lvl.NearestRoad(c); near.Dist(c) > 0.01 {
		ang = near.Sub(c).Angle()
	}
	if !def.Aims() {
		ang = a.gameT * 0.8
	}
	b, h := lk.base[0], lk.headAt(0, ang)
	a.blit(dst, b.img, x, y, art.PadH, b.ox, b.oy, 1, csAlpha(al), false)
	a.blit(dst, h.img, x, y, art.PadH, h.ox, h.oy, 1, csAlpha(al), false)
}

// --- projectiles and beams -----------------------------------------------------

// shotStartZ is the height where a projectile was fired: a muzzle, or the body
// of the enemy a ricochet bounced off.
func (a *App) shotStartZ(p *game.Projectile) float64 {
	P := float64(art.CellPx)
	if p.From != nil {
		lk := a.as.Looks[p.From]
		return a.ground(p.Start.X*P, p.Start.Y*P) + lk.bodyZ + lk.hover
	}
	lk := a.as.Towers[p.Def]
	return art.PadH + lk.tipZ[min(p.Level, towerLevels-1)]
}

// shotEndZ is the height of the target's body.
func (a *App) shotEndZ(p *game.Projectile) float64 {
	P := float64(art.CellPx)
	bodyZ := 4.5
	if lk := a.as.Looks[p.Target]; lk != nil {
		bodyZ = lk.bodyZ + lk.hover
	}
	return a.ground(p.Aim.X*P, p.Aim.Y*P) + bodyZ
}

// flightZ interpolates a projectile's height between its start and its target;
// missiles fly in an arc.
func (a *App) flightZ(p *game.Projectile, at game.Vec) float64 {
	done := at.Dist(p.Start)
	left := p.Aim.Dist(at)
	f := 1.0
	if done+left > 1e-6 {
		f = done / (done + left)
	}
	z0, z1 := a.shotStartZ(p), a.shotEndZ(p)
	z := z0 + (z1-z0)*f
	if p.Kind == game.ProjMissile {
		z += math.Min(22, 7*(done+left)) * math.Sin(math.Pi*f)
	}
	return z
}

func (a *App) drawProjectiles(dst *ebiten.Image) {
	as := a.as
	P := float64(art.CellPx)
	for _, p := range a.world.Projectiles {
		x, y := p.Pos.X*P, p.Pos.Y*P
		z := a.flightZ(p, p.Pos)
		d := p.Pos.Sub(p.Prev)
		if d.Len() < 1e-6 {
			d = p.Aim.Sub(p.Pos)
		}
		if p.Kind == game.ProjMissile {
			dz := z - a.flightZ(p, p.Prev)
			if d.Len() < 1e-6 {
				d = game.FromAngle(p.Heading)
				dz = 0
			}
			ang := math.Atan2(d.Y*P*art.GY-dz*art.HZ, d.X*P)
			wd := d.Norm()
			a.put(dst, as.Glow["orange"], x-wd.X*4, y-wd.Y*4, z, 0.35, 0.35, csAlpha(0.8), true)
			a.put(dst, as.Missile[angleIndex(ang, boltAngles)], x, y, z, 1, 1, ebiten.ColorScale{}, false)
			continue
		}
		ang := math.Atan2(d.Y*art.GY, d.X)
		wd := d.Norm()
		sc := 1.0
		if p.From != nil {
			sc = 0.8
		}
		for i := 1; i <= 3; i++ {
			k := float64(i)
			a.put(dst, as.Glow["cyan"], x-wd.X*3.2*k, y-wd.Y*3.2*k, z, (0.32-0.05*k)*sc, (0.32-0.05*k)*sc, csAlpha(0.55-0.13*k), true)
		}
		a.put(dst, as.Glow["cyan"], x, y, z, 0.55*sc, 0.55*sc, csAlpha(0.7), true)
		a.put(dst, as.Bolt[min(p.Level, 2)][angleIndex(ang, boltAngles)], x, y, z, sc, sc, ebiten.ColorScale{}, false)
	}
}

// drawRails draws fading railgun beams.
func (a *App) drawRails(dst *ebiten.Image) {
	for _, r := range a.fx.Rails {
		k := r.T / r.Life
		x0, y0 := a.screenAt(r.X0, r.Y0, r.Z0)
		x1, y1 := a.screenAt(r.X1, r.Y1, r.Z1)
		w := float32(math.Max(1, a.cam.Z))
		al := 1 - k
		wide := 4.5
		if r.Pierce {
			wide = 6.5
		}
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), w*float32(wide*(1-0.5*k)), pm(art.Violet, 0.35*al), true)
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), w*float32(2*(1-0.6*k)), pm(art.Lilac, 0.8*al), true)
		if k < 0.35 {
			vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), w*0.9, pm(art.White, 1-k/0.35), true)
		}
	}
}

// drawPatches draws burning napalm pools on the road.
func (a *App) drawPatches(dst *ebiten.Image) {
	P := float64(art.CellPx)
	for _, p := range a.world.Patches {
		x, y := p.Pos.X*P, p.Pos.Y*P
		g := a.ground(x, y)
		k := smooth(0, 0.2, p.T) * (1 - smooth(p.Life-0.8, p.Life, p.T))
		fl := 0.85 + 0.15*math.Sin(a.gameT*17+float64(p.ID))
		s := p.R * P / 13
		a.put(dst, a.as.Glow["fire"], x, y, g, s*1.2, s*1.2*art.GY, csAlpha(0.55*k), true)
		a.put(dst, a.as.Glow["orange"], x, y, g, s, s*art.GY, csAlpha(0.6*k*fl), true)
		a.put(dst, a.as.Glow["amber"], x, y, g, s*0.5, s*0.5*art.GY, csAlpha(0.5*k*fl), true)
	}
}
