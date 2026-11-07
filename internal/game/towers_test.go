package game

import (
	"math"
	"testing"
)

// finish runs the world until no turret is under construction.
func finish(w *World) {
	for i := 0; i < 60*10; i++ {
		busy := false
		for _, t := range w.Towers {
			busy = busy || t.Busy
		}
		if !busy {
			return
		}
		w.Update(1.0 / 60)
	}
}

// arena is a world in the wave phase with no scripted spawns.
func arena(t *testing.T) *World {
	w := NewWorld(mustLevel(t), 1)
	w.Credits = 5000
	w.SkipTimer()
	w.pending = w.pending[:0]
	return w
}

func TestConstructionCancelSell(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	start := w.Credits
	tw, err := w.Build(0, Blaster)
	if err != nil || !tw.Busy || w.Credits != start-60 {
		t.Fatal("build must start a construction job", err)
	}
	if w.Upgrade(tw) != ErrBusy || w.Sell(tw) != ErrBusy {
		t.Fatal("a turret under construction cannot be upgraded or sold")
	}
	if err := w.Cancel(tw); err != nil || w.Credits != start || w.SlotTower[0] != nil || len(w.Towers) != 0 {
		t.Fatal("cancelling a build must refund everything and free the pad", err)
	}
	tw, _ = w.Build(0, Blaster)
	w.Update(Blaster.Levels[0].Build - 0.1)
	if !tw.Busy {
		t.Fatal("still building")
	}
	w.Update(0.2)
	if tw.Busy || tw.Spent != 60 {
		t.Fatal("the build must be done", tw.Busy, tw.Spent)
	}
	before := w.Credits
	if err := w.Upgrade(tw); err != nil || !tw.Busy || tw.Level != 0 || tw.JobLevel() != 1 {
		t.Fatal("upgrade must start a job", err)
	}
	w.Update(1)
	if err := w.Cancel(tw); err != nil || tw.Busy || tw.Level != 0 || w.Credits != before {
		t.Fatal("cancelling an upgrade keeps the old level and refunds it")
	}
	w.Upgrade(tw)
	finish(w)
	if tw.Level != 1 || tw.Spent != 130 {
		t.Fatal("upgrade must finish", tw.Level, tw.Spent)
	}
	before = w.Credits
	if err := w.Sell(tw); err != nil || w.Credits != before+91 || w.SlotTower[0] != nil {
		t.Fatal("selling returns 70%", w.Credits-before)
	}
}

func TestSuperLevel(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	w.Credits = 5000
	tw, _ := w.Build(3, Flamer)
	finish(w)
	for i := 0; i < 3; i++ {
		if err := w.Upgrade(tw); err != nil {
			t.Fatal(err)
		}
		finish(w)
	}
	if !tw.HasSuper() || tw.Level != 3 || tw.Stats() != Flamer.Levels[2] {
		t.Fatal("the fourth level adds the super and keeps level 3 stats")
	}
	if w.Upgrade(tw) != ErrMaxLevel {
		t.Fatal("nothing after the super")
	}
}

func TestTargetModes(t *testing.T) {
	w := arena(t)
	tw, _ := w.Build(5, Blaster)
	finish(w)
	near := w.spawn(Crawler, 3, 13, false)  // further along the road
	far := w.spawn(Crawler, 1, 11.5, false) // weaker, behind
	if w.pickTarget(tw, 9) != near {
		t.Fatal("First picks the enemy closest to the base")
	}
	tw.Mode = TargetLast
	if w.pickTarget(tw, 9) != far {
		t.Fatal("Last picks the enemy furthest from the base")
	}
	tw.Mode = TargetStrong
	if w.pickTarget(tw, 9) != near {
		t.Fatal("Strong picks the toughest enemy")
	}
	// Strong goes for the toughest kind of enemy, the leading one first, even
	// when it is already hurt.
	big := w.spawn(Walker, 1, 11, false)
	lead := w.spawn(Walker, 1, 12, false)
	lead.HP = 10
	if w.pickTarget(tw, 9) != lead {
		t.Fatal("Strong picks the leading walker over bugs and fresher walkers")
	}
	big.alive, lead.alive = false, false
	w.CycleMode(tw)
	if tw.Mode != TargetClose {
		t.Fatal("cycle goes to Close")
	}
	w.CycleMode(tw)
	if tw.Mode != TargetFirst {
		t.Fatal("cycle wraps around")
	}
}

func TestCryoSlowsAndFreezes(t *testing.T) {
	w := arena(t)
	tw, _ := w.Build(5, Cryo)
	finish(w)
	e := w.spawn(Crawler, 50, 12.5, false)
	q := w.spawn(Queen, 1, 12.5, false)
	w.Update(0.1)
	if e.SlowT <= 0 || math.Abs(e.SpeedMul()-0.6) > 1e-9 {
		t.Fatal("a frost wave slows by 40%", e.SlowT, e.SpeedMul())
	}
	if math.Abs(q.SpeedMul()-0.8) > 1e-9 {
		t.Fatal("the queen takes half the slow", q.SpeedMul())
	}
	tw.Level = 3 // grant the super directly
	tw.Pulses = 2
	tw.Cooldown = 0
	w.Update(0.05)
	if e.FreezeT <= 0 || e.SpeedMul() != 0 || q.FreezeT > 0 {
		t.Fatal("every third wave freezes, but not the queen")
	}
	d := e.Dist
	w.Update(0.5)
	if e.Dist != d {
		t.Fatal("frozen enemies do not move")
	}
}

func TestFlamerBurnsAndArmorSoaks(t *testing.T) {
	w := arena(t)
	w.Build(5, Flamer)
	finish(w)
	c := w.spawn(Crawler, 50, 12.2, false)
	k := w.spawn(Walker, 5, 12.4, false)
	c.Offset, k.Offset = 0, 0
	w.Update(0.6)
	if c.BurnT <= 0 || c.HP >= c.MaxHP {
		t.Fatal("flames set enemies alight", c.BurnT, c.HP)
	}
	lostC := c.MaxHP - c.HP
	lostK := k.MaxHP - k.HP
	if lostK >= lostC*0.5 {
		t.Fatalf("armour must soak most of the fire: crawler -%.1f walker -%.1f", lostC, lostK)
	}
	if c.HitT < 0.5 {
		t.Fatal("fire damage must not flash like a hit")
	}
}

func TestRailIgnoresArmorAndPierces(t *testing.T) {
	w := arena(t)
	tw, _ := w.Build(5, Railgun)
	finish(w)
	k := w.spawn(Walker, 5, 12.5, false)
	k.Offset = 0
	for i := 0; i < 120 && k.HP == k.MaxHP; i++ {
		w.Update(1.0 / 60)
	}
	if got := k.MaxHP - k.HP; got != Railgun.Levels[0].Damage {
		t.Fatalf("the beam ignores armour: dealt %.1f", got)
	}
	// With the super a beam hits a whole column of enemies.
	w2 := arena(t)
	tw, _ = w2.Build(5, Railgun)
	finish(w2)
	tw.Level = 3
	tw.Mode = TargetStrong
	var line []*Enemy
	for i := 0; i < 4; i++ {
		e := w2.spawn(Crawler, 50, 12.2+0.08*float64(i), false)
		e.Offset = 0
		line = append(line, e)
	}
	hitAny := func() bool {
		for _, e := range line {
			if e.HP < e.MaxHP {
				return true
			}
		}
		return false
	}
	for i := 0; i < 240 && !hitAny(); i++ {
		w2.Update(1.0 / 60)
	}
	hit := 0
	for _, e := range line {
		if e.HP < e.MaxHP {
			hit++
		}
	}
	if hit < 3 {
		t.Fatalf("one piercing beam should hit the whole group, hit %d", hit)
	}
}

func TestSkyguardShootsDronesOnly(t *testing.T) {
	w := arena(t)
	w.Build(3, Skyguard)
	finish(w)
	d := w.spawn(Drone, 1, 20, false)
	c := w.spawn(Crawler, 1, 21, false)
	for i := 0; i < 60*20 && d.alive; i++ {
		w.Update(1.0 / 60)
	}
	if d.alive && d.HP == d.MaxHP {
		t.Fatal("skyguard must hit the drone")
	}
	if c.alive && c.HP < c.MaxHP {
		t.Fatal("skyguard must ignore ground enemies")
	}
}

func TestRicochet(t *testing.T) {
	w := arena(t)
	tw, _ := w.Build(5, Blaster)
	finish(w)
	tw.Level = 3
	var es []*Enemy
	for i := 0; i < 3; i++ {
		e := w.spawn(Walker, 5, 12.4+0.5*float64(i), false)
		e.Offset = 0
		es = append(es, e)
	}
	for i := 0; i < 90; i++ {
		w.Update(1.0 / 60)
	}
	hit := 0
	for _, e := range es {
		if e.HP < e.MaxHP {
			hit++
		}
	}
	if hit != 3 {
		t.Fatalf("one bolt should chain over all three walkers, hit %d", hit)
	}
}

func TestNapalmPools(t *testing.T) {
	w := arena(t)
	tw, _ := w.Build(5, Flamer)
	finish(w)
	tw.Level = 3
	for i := 0; i < 6; i++ {
		w.spawn(Walker, 50, 12.2+0.3*float64(i), false)
	}
	for i := 0; i < 60*2; i++ {
		w.Update(1.0 / 60)
	}
	if len(w.Patches) == 0 {
		t.Fatal("napalm must leave burning pools")
	}
}
