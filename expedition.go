package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

// ---- Expeditions ----
//
// An expedition is a battle the game makes up from what the campaign has
// shown the player (internal/game/expedition.go): only the worlds, turrets,
// abilities and enemies met so far. One is on offer at a time. Its seed is
// kept in the save, so it stays the same until it is won (the next one is
// new) or swapped for a new one after a defeat. There is no free reroll: the
// player does not get to pick what the expedition brings.

// expAbout tells what the expeditions are, on their locked card.
const expAbout = "Every expedition is a new battle made up from what you have met in the campaign: its worlds, turrets and enemies. Its stars pay more crystals than a replay."

// expCard is the card of the expeditions on the chapter select, after the skirmish.
func expCard() int { return len(game.Chapters) + 1 }

// isExpedition reports whether a level is an expedition.
func isExpedition(s *game.LevelSpec) bool { return s != nil && s.ID == game.ExpeditionID }

// expOpen reports whether expeditions can be played: once level 1-1 is won.
func (a *App) expOpen() bool {
	return a.dbg.unlock || a.best(game.Chapters[0].Levels[0]) > 0
}

// knowledge is what the campaign has shown the player: what its won levels
// brought. The first level always counts, so that unlocked test runs have a
// turret to build.
func (a *App) knowledge() game.Knowledge {
	first := game.Chapters[0].Levels[0]
	return game.Learn(func(s *game.LevelSpec) bool { return s == first || a.best(s) > 0 })
}

// expedition is the expedition on offer, made when first needed. It is made
// again from the same seed when the player has learnt something since, so
// that it draws on all of it. It is nil only if no seed gives a playable map,
// which the generator's tests rule out.
func (a *App) expedition() *game.Expedition {
	k := a.knowledge()
	for try := 0; try < 5; try++ {
		if a.save.Exp.Seed == 0 || try > 0 {
			a.rollExpedition()
		}
		key := fmt.Sprintf("%d %s", a.save.Exp.Seed, k.Key())
		if a.exp != nil && a.expKey == key {
			return a.exp
		}
		e := game.NewExpedition(a.save.Exp.Seed, k)
		l, err := game.ParseLevel(e.Spec)
		if err != nil {
			log.Printf("expedition %d: %v", a.save.Exp.Seed, err)
			continue
		}
		g, ti := art.Terrain(l)
		a.levels[game.ExpeditionID] = l
		a.as.Terrains[game.ExpeditionID] = terrainSet{toImage(g), ti}
		a.exp, a.expKey = e, key
		return e
	}
	return nil
}

// rollExpedition puts a new expedition on offer.
func (a *App) rollExpedition() {
	a.save.Exp.Seed = 1 + rand.Int63n(999_999)
	a.save.Exp.Count++
	a.save.store()
	a.exp = nil
}

// startExpedition launches the expedition on offer.
func (a *App) startExpedition() {
	if e := a.expedition(); e != nil {
		a.startLevel(e.Spec)
	}
}

// swapExpedition gives up the expedition on offer (after a defeat) and
// launches a new one.
func (a *App) swapExpedition() {
	a.rollExpedition()
	a.startExpedition()
}

// recordExpedition pays a won expedition at the expedition rate and puts a new
// one on offer: a won expedition is done, it is not replayed for crystals.
func (a *App) recordExpedition() {
	s, w := a.spec, a.world
	a.res = result{stars: w.Stars()}
	n := game.ExpeditionPay(s.Tier) * a.res.stars
	a.res.pay = game.Payout{Total: n}
	a.save.Crystals += n
	a.save.Earned += n
	a.save.Wins++
	if w.BaseHP >= w.BaseMaxHP {
		a.save.Flawless = true
	}
	if w.BaseHP*10 <= w.BaseMaxHP {
		a.save.Close = true
	}
	a.save.Exp.Won++
	a.rollExpedition() // stores the save
	a.res.trophies = a.save.claimTrophies()
}

// expPayNote tells what the crystals of a won expedition were paid for.
func expPayNote(s *game.LevelSpec) string {
	return fmt.Sprintf("%s threat, %d per star", s.Tier, game.ExpeditionPay(s.Tier))
}

// tierColor is the color of a threat tier.
func tierColor(t game.Tier) color.RGBA {
	switch t {
	case game.TierLow:
		return art.Lime
	case game.TierMedium:
		return art.Amber
	}
	return art.Red
}

// drawExpeditionDetail is the briefing of the expedition on offer: where it
// lands, how hard it is, what it brings and what its stars pay.
func (a *App) drawExpeditionDetail(dst *ebiten.Image, cl chaptersLayout) {
	u := a.u
	r := cl.detail
	a.panel(dst, r, 1)
	a.drawButton(dst, cl.open, 1)
	x, w := r.X+8*u, r.W-16*u
	y := r.Y + 8*u
	var e *game.Expedition
	if a.expOpen() {
		e = a.expedition()
	}
	if e == nil {
		a.drawText(dst, "Expeditions", x, y, u, art.Gray, 1)
		y += 11 * u
		a.drawText(dst, "Always a new map", x, y, 2*u, art.White, 1)
		y += 14*u + 6*u
		for _, l := range a.wrap(expAbout, w, u) {
			a.drawText(dst, l, x, y, u, art.Silver, 1)
			y += 11 * u
		}
		y += 8 * u
		a.drawIcon(dst, "lock", x, y-2*u, u, 1)
		a.drawText(dst, "Win level "+game.Chapters[0].Levels[0].ID+" to unlock it.", x+15*u, y, u, art.Amber, 1)
		return
	}
	s := e.Spec
	a.drawText(dst, "Expedition · "+e.World, x, y, u, art.Lime, 1)
	tsegs := []Seg{{"Threat ", art.Gray}, {s.Tier.String(), tierColor(s.Tier)}}
	a.drawSegs(dst, tsegs, x+w-a.segsWidth(tsegs, u), y, u, 1)
	y += 11 * u
	a.drawText(dst, s.Name, x, y, 2*u, art.White, 1)
	y += 14*u + 6*u
	for _, l := range a.wrap(s.Desc, w, u) {
		a.drawText(dst, l, x, y, u, art.Silver, 1)
		y += 11 * u
	}
	y += 5 * u
	// What it brings: only turrets and enemies the player has met.
	a.drawText(dst, "Turrets", x, y+7*u, u, art.Gray, 1)
	tg := 25 * u
	if n := len(s.Turrets); n > 1 {
		tg = min(tg, (w-56*u-24*u)/(n-1))
	}
	for i, id := range s.Turrets {
		d := game.TowerDefs[id]
		if d == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(u), float64(u))
		op.GeoM.Translate(float64(x+56*u+i*tg), float64(y))
		dst.DrawImage(a.as.Towers[d].icon.img, op)
	}
	y += 24 * u
	a.drawText(dst, "Enemies", x, y+3*u, u, art.Gray, 1)
	foes := s.Enemies()
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
	y += 13 * u
	if e.Boss != nil {
		a.drawSegs(dst, []Seg{{"Boss ", art.Gray}, {e.Boss.Name, art.Red}}, x, y, u, 1)
		y += 13 * u
	}
	y += 4 * u
	// What its stars pay: the same on every expedition of this threat.
	per := fmt.Sprint(game.ExpeditionPay(s.Tier))
	rules := []string{"Win the expedition", "Keep half of the base", "Keep 90% of the base"}
	for i, t := range rules {
		for j := 0; j <= i; j++ {
			a.drawIcon(dst, "star0", x+j*12*u, y-2*u, u, 1)
		}
		a.drawText(dst, t, x+40*u, y, u, art.Silver, 1)
		pw := a.textWidth(per, u)
		a.drawText(dst, per, x+w-pw, y, u, art.Pink, 1)
		a.drawIcon(dst, "crystal", x+w-pw-12*u, y-2*u, u, 1)
		y += 13 * u
	}
	// The map picture fills what is left above the button.
	room := cl.open.R.Y - 8*u - y
	if th := min(room, int(float64(w)*0.44)); th >= 36*u {
		tw := int(float64(th) / 0.44)
		pr := Rect{x + (w-min(w, tw))/2, y + (room-th)/2 + 2*u, min(w, tw), th}
		fillRect(dst, pr.X, pr.Y, pr.W, pr.H, bgColor)
		a.drawLevelPic(dst, s, pr, ebiten.ColorScale{})
	}
}
