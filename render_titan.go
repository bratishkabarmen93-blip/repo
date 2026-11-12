package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"outpost/internal/art"
	"outpost/internal/game"
)

// ---- Titan: methane fog, Beacons and the Kraken's tentacles ----------------------

var (
	fogDisc, lightDisc *ebiten.Image       // soft white discs: puffs of fog, and the light that cuts them
	fogLayer           *ebiten.Image       // the fog of the map, drawn apart so light can cut holes in it
	grabStart          = map[int]float64{} // game time each turret was seized, by turret ID

	fogCol   = art.Hex(0xd6b98a)
	fogHi    = art.Hex(0xeedbb2)
	fogPlanC = art.Hex(0xfff0c8)
	lightCol = art.Hex(0xffe2a0)
	armDark  = art.Hex(0x3e1f3c)
)

// discImage bakes a white disc whose alpha fades out between r0 (relative to
// the radius) and the rim.
func discImage(r0 float64) *ebiten.Image {
	const n = 96
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := (float64(x)+0.5)/n*2 - 1
			dy := (float64(y)+0.5)/n*2 - 1
			al := 1 - smooth(r0, 1, math.Hypot(dx, dy))
			v := byte(math.Round(al * 255))
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, v
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}

func titanDiscs() {
	if fogDisc == nil {
		fogDisc = discImage(0.25)
		lightDisc = discImage(0.72)
	}
}

// beaconLight is the radius (cells) of a Beacon's light.
func beaconLight(w *game.World, tw *game.Tower) float64 { return w.Range(tw) }

// groundDisc draws img as a flat disc of radius r (world pixels) lying at
// height z over the world point (x, y).
func (a *App) groundDisc(dst, img *ebiten.Image, x, y, z, r float64, cs ebiten.ColorScale, blend ebiten.Blend) {
	n := float64(img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-n/2, -n/2)
	op.GeoM.Scale(2*r/n*a.cam.Z, 2*r*art.GY/n*a.cam.Z)
	sx, sy := a.screenAt(x, y, z)
	op.GeoM.Translate(sx, sy)
	op.ColorScale = cs
	op.Filter = ebiten.FilterLinear
	op.Blend = blend
	dst.DrawImage(img, op)
}

// drawFog veils the map in Titan's methane fog: drifting banks and the puffs
// Bloaters leave behind, with holes where Beacons shine. Then it shows where
// the banks will drift, and the lanterns glowing through it all.
func (a *App) drawFog(dst *ebiten.Image) {
	w := a.world
	if len(w.Clouds) > 0 && a.sw > 0 && a.sh > 0 {
		titanDiscs()
		if fogLayer == nil || fogLayer.Bounds().Dx() != a.sw || fogLayer.Bounds().Dy() != a.sh {
			if fogLayer != nil {
				fogLayer.Deallocate()
			}
			fogLayer = ebiten.NewImage(a.sw, a.sh)
		}
		fogLayer.Clear()
		P := float64(art.CellPx)
		t := a.gameT
		for _, c := range w.Clouds {
			k := c.Strength()
			if k <= 0 {
				continue
			}
			x, y, r := c.Pos.X*P, c.Pos.Y*P, c.R*P
			z := a.ground(x, y) + 6
			seed := float64(c.ID) * 1.7
			a.groundDisc(fogLayer, fogDisc, x, y, z, r*1.1, csCol(fogCol, 0.6*k), ebiten.BlendSourceOver)
			// Puffs roll slowly around the core.
			for i := 0; i < 6; i++ {
				fi := float64(i)
				ang := t*0.12*(1+0.15*float64(i%3)) + fi/6*2*math.Pi + seed
				d := r * (0.5 + 0.08*math.Sin(t*0.5+fi*2.1+seed))
				px, py := x+math.Cos(ang)*d, y+math.Sin(ang)*d
				a.groundDisc(fogLayer, fogDisc, px, py, z+2+1.5*math.Sin(t*0.7+fi), r*0.62, csCol(fogHi, 0.32*k), ebiten.BlendSourceOver)
			}
		}
		// The light of the Beacons clears the fog.
		for _, tw := range w.Towers {
			if tw.Def.Attack != game.AtkBeacon || !tw.Working() {
				continue
			}
			x, y := tw.Pos.X*P, tw.Pos.Y*P
			a.groundDisc(fogLayer, lightDisc, x, y, a.ground(x, y)+6, beaconLight(w, tw)*P*1.06, csAlpha(1), ebiten.BlendDestinationOut)
		}
		dst.DrawImage(fogLayer, &ebiten.DrawImageOptions{})
		a.drawFogPlan(dst)
	}
	a.drawLamps(dst)
}

// drawFogPlan shows where the banks of fog will drift: while the players
// prepare, the drift the next wave brings (pulsing); during a wave, the drift
// under way (faint).
func (a *App) drawFogPlan(dst *ebiten.Image) {
	w := a.world
	if !w.HasFog() || !w.Active() {
		return
	}
	P := float64(art.CellPx)
	prep := w.Phase == game.PhasePrep
	lw := math.Max(1, 1.3*a.cam.Z)
	for _, m := range w.FogPlan() {
		if m.From.Dist2(m.To) < 0.04 {
			continue
		}
		al := 0.5 + 0.25*math.Sin(a.gameT*3)
		if !prep {
			al = 0.28
		}
		col := pm(fogPlanC, al)
		x, y := m.To.X*P, m.To.Y*P
		cx, cy := a.screenAt(x, y, a.ground(x, y))
		dashedEllipse(dst, cx, cy, m.R*P*a.cam.Z, m.R*P*art.GY*a.cam.Z, lw, col, a.gameT*0.8)
		fx, fy := m.From.X*P, m.From.Y*P
		sx, sy := a.screenAt(fx, fy, a.ground(fx, fy))
		dashedArrow(dst, sx, sy, cx, cy, lw, col, a.gameT*0.8)
	}
}

// drawLamps makes the lanterns of the Beacons glow, fog or not.
func (a *App) drawLamps(dst *ebiten.Image) {
	P := float64(art.CellPx)
	for _, tw := range a.world.Towers {
		if tw.Def.Attack != game.AtkBeacon || tw.Busy {
			continue
		}
		k := 1.0
		if tw.HeldT > 0 {
			k = 0.3
		}
		_, lz := art.TurretTip("beacon", tw.Level)
		x, y := tw.Pos.X*P, tw.Pos.Y*P
		z := art.PadH + lz
		fl := 0.85 + 0.15*math.Sin(a.gameT*5.3+float64(tw.ID)*1.9)
		a.put(dst, a.as.Glow["amber"], x, y, z, 0.9, 0.9, csAlpha(0.75*k*fl), true)
		if tw.HasSuper() {
			a.put(dst, a.as.Glow["white"], x, y, z, 0.45, 0.45, csAlpha(0.5*k*fl), true)
		}
	}
}

// drawBeaconLight lights the ground around every Beacon and marks the edge of
// its light.
func (a *App) drawBeaconLight(dst *ebiten.Image) {
	w := a.world
	P := float64(art.CellPx)
	titanDiscs()
	for _, tw := range w.Towers {
		if tw.Def.Attack != game.AtkBeacon || tw.Busy {
			continue
		}
		k := 1.0
		if tw.HeldT > 0 {
			k = 0.2
		}
		x, y := tw.Pos.X*P, tw.Pos.Y*P
		g := a.ground(x, y)
		r := beaconLight(w, tw) * P
		fl := 0.9 + 0.1*math.Sin(a.gameT*5.3+float64(tw.ID)*1.9)
		a.groundDisc(dst, fogDisc, x, y, g, r*1.05, csCol(lightCol, 0.13*k*fl), ebiten.BlendLighter)
		cx, cy := a.screenAt(x, y, g)
		dashedEllipse(dst, cx, cy, r*a.cam.Z, r*art.GY*a.cam.Z, math.Max(1, a.cam.Z), pm(lightCol, 0.3*k), -a.gameT*0.4)
	}
}

// drawTentacles draws the Kraken's tentacles wrapped around the turrets it
// holds: they lash out when it grabs and slide back as the turret breaks free.
func (a *App) drawTentacles(dst *ebiten.Image) {
	w := a.world
	P := float64(art.CellPx)
	for _, tw := range w.Towers {
		if tw.HeldT <= 0 {
			continue
		}
		var src *game.Enemy
		for _, e := range w.Enemies {
			if e.ID == tw.HeldBy && e.Alive() {
				src = e
				break
			}
		}
		if src == nil {
			continue
		}
		reach := smooth(0, 0.3, a.gameT-grabStart[tw.ID]) * smooth(0, 0.45, tw.HeldT)
		if reach <= 0 {
			continue
		}
		ex, ey := src.Pos.X*P, src.Pos.Y*P
		tx, ty := tw.Pos.X*P, tw.Pos.Y*P
		x0, y0 := a.screenAt(ex, ey, a.ground(ex, ey)+5)
		x1, y1 := a.screenAt(tx, ty, art.PadH+7)
		dx, dy := x1-x0, y1-y0
		l := math.Hypot(dx, dy)
		if l < 1 {
			continue
		}
		nx, ny := -dy/l, dx/l
		const n = 16
		var pts [n + 1][2]float64
		for i := 0; i <= n; i++ {
			s := float64(i) / n * reach
			wob := math.Sin(s*math.Pi) * l * 0.1 * math.Sin(a.gameT*5+s*7+float64(tw.ID))
			arch := math.Sin(s*math.Pi) * l * 0.18
			pts[i] = [2]float64{x0 + dx*s + nx*wob, y0 + dy*s + ny*wob - arch}
		}
		z := a.cam.Z
		for i := 0; i < n; i++ {
			wd := (5.5 - 3.5*float64(i)/n) * z
			strokeSeg(dst, pts[i], pts[i+1], wd+1.5*z, armDark)
			strokeSeg(dst, pts[i], pts[i+1], wd, art.Purple)
			strokeSeg(dst, pts[i], pts[i+1], wd*0.35, art.Magenta)
		}
		if reach > 0.95 {
			// The tip coils around the turret.
			strokeEllipse(dst, x1, y1+2*z, 8*z, 4.5*z, 2.2*z, art.Purple)
		}
	}
}

// strokeSeg strokes one round-capped segment (screen space).
func strokeSeg(dst *ebiten.Image, p0, p1 [2]float64, w float64, col color.RGBA) {
	var p vector.Path
	p.MoveTo(float32(p0[0]), float32(p0[1]))
	p.LineTo(float32(p1[0]), float32(p1[1]))
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(col)
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(w), LineCap: vector.LineCapRound}, op)
}

// dashedEllipse strokes a dashed ellipse (screen space); phase turns the
// dashes around it.
func dashedEllipse(dst *ebiten.Image, cx, cy, rx, ry, w float64, col color.RGBA, phase float64) {
	n := max(12, int((rx+ry)*0.3)) * 2
	ph := math.Mod(phase, 2)
	var p vector.Path
	for i := 0; i < n; i += 2 {
		for k := 0; k <= 4; k++ {
			s := (float64(i) + ph + float64(k)/4) / float64(n) * 2 * math.Pi
			x, y := float32(cx+math.Cos(s)*rx), float32(cy+math.Sin(s)*ry)
			if k == 0 {
				p.MoveTo(x, y)
			} else {
				p.LineTo(x, y)
			}
		}
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(col)
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(w)}, op)
}

// dashedArrow strokes a dashed line from (x0, y0) to (x1, y1) with a head.
func dashedArrow(dst *ebiten.Image, x0, y0, x1, y1, w float64, col color.RGBA, phase float64) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l < 6*w {
		return
	}
	ux, uy := dx/l, dy/l
	dash := 4 * w
	var p vector.Path
	for s := math.Mod(phase, 1)*2*dash - 2*dash; s < l; s += 2 * dash {
		a, b := math.Max(0, s), math.Min(l-w, s+dash)
		if b > a {
			p.MoveTo(float32(x0+ux*a), float32(y0+uy*a))
			p.LineTo(float32(x0+ux*b), float32(y0+uy*b))
		}
	}
	h := 4 * w
	p.MoveTo(float32(x1-ux*h-uy*h*0.6), float32(y1-uy*h+ux*h*0.6))
	p.LineTo(float32(x1), float32(y1))
	p.LineTo(float32(x1-ux*h+uy*h*0.6), float32(y1-uy*h-ux*h*0.6))
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(col)
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(w)}, op)
}
