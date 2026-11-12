package game

import (
	"math"
	"sort"
)

// ---- Methane fog (Titan) ----
//
// Banks of fog drift over the map from stop to stop: before wave k a bank
// rests at its stop k-1, and during the wave it drifts on to stop k (the
// stops start over when they run out). The next drift is known before the
// wave, so the player can plan for it. A turret standing in the fog loses a
// third of its range; inside the fog, Stalkers can only be aimed at by
// turrets in the light of a Beacon, and a Beacon clears the fog around itself.

// FogBank is a bank of fog and the stops it drifts along, one per wave.
type FogBank struct {
	R     float64 // radius (cells)
	Stops []Vec   // where it rests before each wave (cells), repeating
}

// Bank is a shorthand for a fog bank of radius r resting at the given stops
// (x, y pairs in cells).
func Bank(r float64, xy ...float64) FogBank {
	b := FogBank{R: r}
	for i := 0; i+1 < len(xy); i += 2 {
		b.Stops = append(b.Stops, Vec{xy[i], xy[i+1]})
	}
	return b
}

// StopFor is where the bank rests before wave n (1-based).
func (b FogBank) StopFor(n int) Vec {
	if len(b.Stops) == 0 {
		return Vec{}
	}
	return b.Stops[max(0, n-1)%len(b.Stops)]
}

const (
	// FogRange is the share of its range a turret keeps in the fog.
	FogRange = 2.0 / 3
	// FogSpeed is how fast the banks drift (cells per second).
	FogSpeed = 0.3
)

// Cloud is a bank of fog on the map, or a puff a Bloater left behind.
type Cloud struct {
	ID   int
	Pos  Vec
	R    float64
	Bank int     // index of its bank in the level's fog (-1: a puff)
	From Vec     // banks: where the current drift started
	To   Vec     // banks: where it drifts to (or rests)
	Life float64 // puffs: seconds it lasts
	T    float64 // puffs: seconds since it appeared
}

// Puff reports whether the cloud is a Bloater's puff rather than a bank.
func (c *Cloud) Puff() bool { return c.Bank < 0 }

// Strength is how thick the cloud looks right now, 0..1: puffs swell up and
// thin out again (for the game they are fog for their whole life).
func (c *Cloud) Strength() float64 {
	if !c.Puff() {
		return 1
	}
	return Smoothstep(0, 0.5, c.T) * (1 - Smoothstep(c.Life-2.5, c.Life, c.T))
}

// Drifting reports whether a bank is still on its way to its stop.
func (c *Cloud) Drifting() bool { return !c.Puff() && c.Pos.Dist2(c.To) > 1e-6 }

// initFog puts the banks of the level at their first stops.
func (w *World) initFog() {
	for i, b := range w.Lvl.Spec.Fog {
		p := b.StopFor(1)
		w.Clouds = append(w.Clouds, &Cloud{ID: w.id(), Pos: p, R: b.R, Bank: i, From: p, To: p})
	}
}

// driftFog sends every bank towards its stop for the wave that just started.
func (w *World) driftFog() {
	for _, c := range w.Clouds {
		if c.Puff() {
			continue
		}
		c.From = c.Pos
		c.To = w.Lvl.Spec.Fog[c.Bank].StopFor(w.Wave + 1)
	}
}

// updateFog moves the banks and ages the puffs.
func (w *World) updateFog(dt float64) {
	if len(w.Clouds) == 0 {
		return
	}
	cs := w.Clouds[:0]
	for _, c := range w.Clouds {
		if c.Puff() {
			if c.T += dt; c.T >= c.Life {
				continue
			}
		} else if d := c.To.Sub(c.Pos); d.Len() > 1e-9 {
			if step := FogSpeed * dt; d.Len() <= step {
				c.Pos = c.To
			} else {
				c.Pos = c.Pos.Add(d.Mul(step / d.Len()))
			}
		}
		cs = append(cs, c)
	}
	for i := len(cs); i < len(w.Clouds); i++ {
		w.Clouds[i] = nil
	}
	w.Clouds = cs
}

// FogMove is where a bank of fog will drift.
type FogMove struct {
	From, To Vec
	R        float64
}

// FogPlan shows the drift of every bank: during the preparation phase the
// drift the next wave brings, during a wave the one under way.
func (w *World) FogPlan() []FogMove {
	var out []FogMove
	for _, c := range w.Clouds {
		if c.Puff() {
			continue
		}
		m := FogMove{From: c.Pos, To: c.To, R: c.R}
		if w.Phase == PhasePrep {
			m.From, m.To = c.To, w.Lvl.Spec.Fog[c.Bank].StopFor(w.Wave+2)
		}
		out = append(out, m)
	}
	return out
}

// HasFog reports whether the level has banks of fog.
func (w *World) HasFog() bool { return len(w.Lvl.Spec.Fog) > 0 }

// Clouded reports whether a point lies inside a cloud (lit or not).
func (w *World) Clouded(p Vec) bool {
	for _, c := range w.Clouds {
		if c.Pos.Dist2(p) < c.R*c.R {
			return true
		}
	}
	return false
}

// Foggy reports whether a point lies in the fog: inside a cloud and outside
// the light of every Beacon.
func (w *World) Foggy(p Vec) bool {
	return len(w.Clouds) > 0 && w.Clouded(p) && !w.Lit(p)
}

// Working reports whether a finished turret is doing its job: not under
// construction and not seized.
func (t *Tower) Working() bool { return !t.Busy && t.HeldT <= 0 }

// Lit reports whether a point is in the light of a working Beacon.
func (w *World) Lit(p Vec) bool {
	for _, t := range w.Towers {
		if t.Def.Attack == AtkBeacon && t.Working() {
			if r := t.Stats().Range; t.Pos.Dist2(p) <= r*r {
				return true
			}
		}
	}
	return false
}

// Boosts are what the Beacons shining on a turret give it: extra range (a
// share) and extra fire rate (a share, from Overdrive). Beacons do not add
// up: the best one counts.
func (w *World) Boosts(t *Tower) (rng, haste float64) {
	if t.Def.Attack == AtkBeacon {
		return 0, 0
	}
	for _, b := range w.Towers {
		if b.Def.Attack != AtkBeacon || !b.Working() {
			continue
		}
		st := b.Stats()
		if b.Pos.Dist2(t.Pos) > st.Range*st.Range {
			continue
		}
		rng = math.Max(rng, st.Boost)
		if b.HasSuper() {
			haste = math.Max(haste, b.Def.Super.Haste)
		}
	}
	return rng, haste
}

// Range is the reach of a turret right now: a third less in the fog, more in
// the light of a Beacon (a Beacon's own light is never dimmed).
func (w *World) Range(t *Tower) float64 {
	r := t.Stats().Range
	if t.Def.Attack == AtkBeacon {
		return r
	}
	if w.Foggy(t.Pos) {
		return r * FogRange
	}
	b, _ := w.Boosts(t)
	return r * (1 + b)
}

// Sees reports whether a turret can aim at an enemy: Stalkers hidden in the
// fog only show up to turrets in the light of a Beacon.
func (w *World) Sees(t *Tower, e *Enemy) bool { return !e.Hidden || w.Lit(t.Pos) }

// updateHidden hides a Stalker that walks into the fog and shows it again
// when it leaves (or a Beacon lights it up).
func (w *World) updateHidden(e *Enemy) {
	hid := w.Foggy(e.Pos)
	if hid == e.Hidden {
		return
	}
	e.Hidden = hid
	if !hid {
		w.emit(Event{Kind: EvReveal, Pos: e.Pos, ID: e.ID, Enemy: e.Def})
		return
	}
	// The shots in flight from turrets that cannot see it lose it.
	for _, p := range w.Projectiles {
		if p.TargetID == e.ID && !p.Sight {
			p.TargetID = 0
			e.pending -= p.eff
		}
	}
	e.pending = math.Max(0, e.pending)
	w.emit(Event{Kind: EvHide, Pos: e.Pos, ID: e.ID, Enemy: e.Def})
}

// puff leaves a cloud of fog where a Bloater fell.
func (w *World) puff(src *Enemy, p *PuffSpec) {
	c := &Cloud{ID: w.id(), Pos: src.Pos, R: p.R, Bank: -1, Life: p.Life}
	w.Clouds = append(w.Clouds, c)
	w.emit(Event{Kind: EvPuff, Pos: src.Pos, ID: c.ID, R: p.R, Enemy: src.Def})
}

// blast blows up a Bloater that fire finished off: the methane burns every
// ground enemy around it (more for tougher Bloaters). Bloaters caught in it
// go up too.
func (w *World) blast(src *Enemy, b *BlastSpec) {
	dmg := b.Damage * src.MaxHP / src.Def.HP
	r2 := b.R * b.R
	var hit []*Enemy
	for _, e := range w.Enemies {
		if e != src && e.Targetable() && !e.Def.Flying && e.Pos.Dist2(src.Pos) <= r2 {
			hit = append(hit, e)
		}
	}
	w.emit(Event{Kind: EvBlast, Pos: src.Pos, ID: src.ID, R: b.R, Value: len(hit), Enemy: src.Def})
	for _, e := range hit {
		if e.alive {
			e.HitT = 0
		}
		w.hurtBy(e, e.Def.EffDamage(dmg*e.Def.FireMul()), true)
	}
}

// updateGrab lets an enemy with tentacles seize the turrets nearest to it.
func (w *World) updateGrab(e *Enemy, g *GrabSpec, dt float64) {
	if e.grabCD -= dt; e.grabCD > 0 {
		return
	}
	r2 := g.Reach * g.Reach
	var near []*Tower
	for _, t := range w.Towers {
		if t.Working() && t.Pos.Dist2(e.Pos) <= r2 {
			near = append(near, t)
		}
	}
	if len(near) == 0 {
		e.grabCD = 0 // it keeps feeling for one
		return
	}
	sort.SliceStable(near, func(i, j int) bool {
		return near[i].Pos.Dist2(e.Pos) < near[j].Pos.Dist2(e.Pos)
	})
	for _, t := range near[:min(g.Count, len(near))] {
		t.HeldT, t.HeldBy, t.TargetID = g.Hold, e.ID, 0
		w.emit(Event{Kind: EvGrab, Pos: e.Pos, To: t.Pos, ID: t.ID, Value: e.ID, R: g.Hold, Tower: t.Def, Enemy: e.Def})
	}
	e.grabCD = g.Period
}

// updateHold counts down a seized turret and lets it go when its time is up
// or the enemy holding it is gone.
func (w *World) updateHold(t *Tower, dt float64) {
	if t.HeldT <= 0 {
		return
	}
	t.HeldT -= dt
	if t.HeldT > 0 && w.enemyByID(t.HeldBy) != nil {
		return
	}
	w.emit(Event{Kind: EvRelease, Pos: t.Pos, ID: t.ID, Value: t.HeldBy, Tower: t.Def})
	t.HeldT, t.HeldBy = 0, 0
}
