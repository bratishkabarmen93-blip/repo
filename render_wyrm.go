package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

// The Ice Wyrm is drawn as a chain of baked parts that follow its head along
// the road. Between paired ice holes it swims under the frozen lake, where
// only a dark shape and the glow of its eyes give it away.

const wyrmStride = 16 // draw item index of a wyrm part: wyrm*wyrmStride + part

// wyrmPart is one piece of a wyrm laid out for this frame (world pixels).
type wyrmPart struct {
	x, y, z float64 // z is the height of the model origin
	g       float64 // ground height under it
	heading float64
	kind    int     // -1 is the head, else an art.Wyrm* body kind
	al      float64 // opacity above the ice
	under   float64 // 0 on the surface, 1 swimming under the ice
}

// wyrmLaid is a live wyrm and its parts, head first.
type wyrmLaid struct {
	e     *game.Enemy
	parts []wyrmPart
}

func isWyrm(def *game.EnemyDef) bool { return def == game.Wyrm }

// wyrmBack is how far part k (0 = head) trails behind the head, in cells.
func wyrmBack(k int) float64 {
	if k == 0 {
		return 0
	}
	return art.WyrmBody[k-1].Back
}

// iceDepth is how deep a diver is at path distance d: 0 on the surface,
// 1 under the ice, in between while it slides through a hole.
func iceDepth(rt *game.Route, d float64) float64 {
	u := 0.0
	for _, s := range rt.Ice {
		u = math.Max(u, smooth(s[0]-0.22, s[0]+0.12, d)*(1-smooth(s[1]-0.12, s[1]+0.22, d)))
	}
	return u
}

// nearestHole is the path distance of the ice hole closest to d.
func nearestHole(rt *game.Route, d float64) float64 {
	best, bd := d, math.Inf(1)
	for _, s := range rt.Ice {
		for _, h := range s {
			if math.Abs(h-d) < bd {
				best, bd = h, math.Abs(h-d)
			}
		}
	}
	return best
}

// layWyrm places the parts of a wyrm whose head is dist along route rt and
// faces head. The body slithers in a gentle wave that stays put on the
// ground, so every segment follows the same track as the head.
func (a *App) layWyrm(dst []wyrmPart, rt *game.Route, dist, head float64) []wyrmPart {
	P := float64(art.CellPx)
	phase := dist * 2 * math.Pi / 1.7
	dst = dst[:0]
	for k := 0; k <= len(art.WyrmBody); k++ {
		back := wyrmBack(k)
		d := dist - back
		pos, dir := rt.Path.At(d)
		w := phase - back*3.2
		amp := smooth(0, 0.9, back)
		pos = pos.Add(dir.Perp().Mul(0.075 * amp * math.Sin(w)))
		x, y := pos.X*P, pos.Y*P
		u := iceDepth(rt, d)
		p := wyrmPart{x: x, y: y, g: a.ground(x, y), kind: -1, heading: head, under: u}
		if k > 0 {
			p.kind = art.WyrmBody[k-1].Kind
			q := dst[k-1]
			if dx, dy := q.x-x, q.y-y; dx*dx+dy*dy > 1 {
				p.heading = math.Atan2(dy, dx)
			} else {
				p.heading = dir.Angle()
			}
		}
		p.z = p.g + 0.7*amp*math.Cos(w) - 8*u
		p.al = smooth(-0.25, 0.1, d) * (1 - smooth(0.3, 0.85, u))
		dst = append(dst, p)
	}
	return dst
}

// layWyrms lays out every live wyrm for this frame.
func (a *App) layWyrms() {
	n := 0
	for _, e := range a.enemies {
		if !isWyrm(e.Def) {
			continue
		}
		if n == len(a.wyrms) {
			a.wyrms = append(a.wyrms, wyrmLaid{})
		}
		lw := &a.wyrms[n]
		h, ok := a.heads[e.ID]
		if !ok {
			h = e.Dir.Angle()
		}
		lw.e = e
		lw.parts = a.layWyrm(lw.parts, a.lvl.Routes[e.Route], e.Dist, h)
		// Fade in from the burrow and out into the base.
		f := smooth(0, 0.35, e.Age) * smooth(0, 0.55, e.Left())
		for i := range lw.parts {
			lw.parts[i].al *= f
		}
		n++
	}
	a.wyrms = a.wyrms[:n]
}

// wyrmSprite is the baked model of one part.
func (a *App) wyrmSprite(e *game.Enemy, p *wyrmPart) sprite {
	if p.kind < 0 {
		lk := a.as.Looks[e.Def]
		return lk.spr[angleIndex(p.heading, enemyAngles)][a.enemyFrame(e, lk)]
	}
	return a.as.WyrmParts[p.kind][angleIndex(p.heading, enemyAngles)]
}

func (a *App) underLayer() *ebiten.Image {
	if a.underImg == nil || a.underImg.Bounds().Dx() != a.sw || a.underImg.Bounds().Dy() != a.sh {
		if a.underImg != nil {
			a.underImg.Deallocate()
		}
		a.underImg = ebiten.NewImage(a.sw, a.sh)
	}
	a.underImg.Clear()
	return a.underImg
}

// drawUnderIce draws the dark shapes of wyrms swimming under the frozen lakes.
func (a *App) drawUnderIce(dst *ebiten.Image) {
	some := false
	for _, lw := range a.wyrms {
		for _, p := range lw.parts {
			if p.under > 0.01 {
				some = true
			}
		}
	}
	if !some {
		return
	}
	l := a.underLayer()
	deep := art.Hex(0x050914)
	for _, lw := range a.wyrms {
		fade := smooth(0, 0.55, lw.e.Left())
		for i := len(lw.parts) - 1; i >= 0; i-- {
			p := &lw.parts[i]
			if p.under <= 0.01 {
				continue
			}
			spr := a.wyrmSprite(lw.e, p)
			// Seen through the ice from above, the body looks flatter.
			op := &a.op
			*op = ebiten.DrawImageOptions{}
			op.GeoM.Translate(-spr.ox, -spr.oy)
			op.GeoM.Scale(a.cam.Z, a.cam.Z*0.62)
			sx, sy := a.screenAt(p.x, p.y, p.g-2)
			op.GeoM.Translate(sx, sy)
			op.ColorScale = csCol(deep, p.under*fade)
			op.Filter = ebiten.FilterPixelated
			l.DrawImage(spr.sil, op)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(0.72)
	dst.DrawImage(l, op)
	// The amber eyes glow through the ice.
	for _, lw := range a.wyrms {
		h := &lw.parts[0]
		if h.under <= 0.01 {
			continue
		}
		k := h.under * (0.8 + 0.2*math.Sin(a.gameT*3+float64(lw.e.ID)))
		c, s := math.Cos(h.heading), math.Sin(h.heading)
		a.put(dst, a.as.Glow["amber"], h.x+c*5, h.y+s*5, h.g, 0.55, 0.45, csAlpha(0.3*k), true)
		for _, side := range []float64{-1, 1} {
			ex, ey := h.x+c*7-s*3.4*side, h.y+s*7+c*3.4*side
			a.put(dst, a.as.Glow["amber"], ex, ey, h.g, 0.2, 0.18, csAlpha(k), true)
		}
	}
}

// wyrmShadows adds the ground shadows of the parts above the ice.
func (a *App) wyrmShadows(sl *ebiten.Image) {
	for _, lw := range a.wyrms {
		for i := range lw.parts {
			p := &lw.parts[i]
			al := p.al * (1 - p.under)
			if al <= 0.01 {
				continue
			}
			spr := a.wyrmSprite(lw.e, p)
			if spr.sh != nil {
				a.blit(sl, spr.sh, p.x, p.y, p.g, spr.sx, spr.sy, 1, csAlpha(al), false)
			}
		}
	}
}

// wyrmItems adds every visible part to the depth-sorted draw list. Along a
// level stretch the part nearer the head is drawn on top.
func (a *App) wyrmItems(items []drawItem) []drawItem {
	for wi, lw := range a.wyrms {
		n := len(lw.parts)
		for k := range lw.parts {
			if lw.parts[k].al > 0.01 {
				items = append(items, drawItem{lw.parts[k].y + 0.02*float64(n-k), kWyrm, wi*wyrmStride + k})
			}
		}
	}
	return items
}

// drawWyrmPart draws one part of a wyrm with its hit flash and fire glow.
func (a *App) drawWyrmPart(dst *ebiten.Image, i int) {
	lw := &a.wyrms[i/wyrmStride]
	p := &lw.parts[i%wyrmStride]
	e := lw.e
	spr := a.wyrmSprite(e, p)
	al := p.al
	a.blit(dst, spr.img, p.x, p.y, p.z, spr.ox, spr.oy, 1, csAlpha(al), false)
	if e.HitT < 0.1 {
		a.blit(dst, spr.sil, p.x, p.y, p.z, spr.ox, spr.oy, 1, csCol(art.White, (1-e.HitT/0.1)*0.2*al), true)
	}
	if e.HealT < 0.45 {
		k := 1 - e.HealT/0.45
		a.blit(dst, spr.sil, p.x, p.y, p.z, spr.ox, spr.oy, 1, csCol(art.Lime, k*0.4*al), true)
	}
	if e.BurnT > 0 {
		k := smooth(0, 0.3, e.BurnT) * (0.75 + 0.25*math.Sin(a.gameT*21+float64(e.ID)+float64(i)*0.9))
		a.blit(dst, spr.sil, p.x, p.y, p.z, spr.ox, spr.oy, 1, csCol(art.Orange, 0.24*k*al), true)
	}
}

// updateWyrmFX churns the water of the holes the wyrms slide through and
// makes the ice ripple over a swimming one.
func (a *App) updateWyrmFX(dt float64) {
	P := float64(art.CellPx)
	for _, e := range a.world.Enemies {
		if !e.Alive() || e.Def.Dive == 0 {
			continue
		}
		rt := a.lvl.Routes[e.Route]
		if e.Under && a.fx.rng.Float64() < dt*6 {
			x, y := e.Pos.X*P, e.Pos.Y*P
			a.fx.Rings = append(a.fx.Rings, Ring{X: x, Y: y, Z: a.ground(x, y), R0: 4, R1: a.fx.r(12, 18), Life: 0.7, Width: 1.1, Col: pm(art.Hex(0x9fe8ff), 0.4)})
		}
		n := 1
		if isWyrm(e.Def) {
			n += len(art.WyrmBody)
		}
		for _, s := range rt.Ice {
			for _, hd := range s {
				near := false
				for k := 0; k < n; k++ {
					if d := e.Dist - wyrmBack(k); math.Abs(d-hd) < 0.3 {
						near = true
						break
					}
				}
				if near && a.fx.rng.Float64() < dt*10 {
					pos, _ := rt.Path.At(hd)
					x, y := pos.X*P, pos.Y*P
					a.fx.Churn(x, y, a.ground(x, y))
				}
			}
		}
	}
}

// wyrmKill bursts a dead wyrm apart from the tail to the head.
func (a *App) wyrmKill(ev game.Event) {
	lk := a.as.Looks[ev.Enemy]
	rt := a.lvl.Routes[ev.Route]
	h, ok := a.heads[ev.ID]
	if !ok {
		h = ev.Angle
	}
	parts := a.layWyrm(nil, rt, ev.Dist, h)
	n := len(parts)
	for k := n - 1; k >= 0; k-- {
		p := parts[k]
		if p.al < 0.05 || p.under > 0.5 {
			continue
		}
		hold := 0.075 * float64(n-1-k)
		c := Corpse{X: p.x, Y: p.y, Z: p.z, Look: lk, Hold: hold, Life: 0.22, Flash: art.Hex(0xbfefff)}
		if k == 0 {
			c.Heading, c.Frame = angleIndex(p.heading, enemyAngles), 3
			c.Life = 0.3
		} else {
			spr := a.as.WyrmParts[p.kind][angleIndex(p.heading, enemyAngles)]
			c.Spr = &spr
		}
		a.fx.Corpses = append(a.fx.Corpses, c)
		a.fx.WyrmPop(lk, p.x, p.y, p.z+lk.bodyZ*0.6, p.g, hold, k == 0)
	}
}
