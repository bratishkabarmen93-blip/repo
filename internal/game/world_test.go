package game

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func mustLevel(t *testing.T) *Level {
	t.Helper()
	l, err := ParseLevel(Level1)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestLevel1Parses(t *testing.T) {
	l := mustLevel(t)
	if len(l.Slots) != 10 || l.BaseSize != 3 {
		t.Fatalf("slots=%d base=%d", len(l.Slots), l.BaseSize)
	}
	if l.Path.Len < 40 || l.Path.Len > 60 {
		t.Fatalf("unexpected path length %.2f", l.Path.Len)
	}
	t.Logf("path length %.2f cells, %d corners, %d smoothed points", l.Path.Len, len(l.Corners), len(l.Path.Pts))
}

func TestTimerAndSkip(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	if w.Phase != PhasePrep || w.Timer != Level1.FirstPrep {
		t.Fatal("game must start with the preparation timer")
	}
	w.Update(1)
	if !w.SkipTimer() || w.Phase != PhaseWave || w.Wave != 1 {
		t.Fatal("skip must start wave 1")
	}
	if w.SkipTimer() {
		t.Fatal("cannot skip while a wave is running")
	}
}

func TestNoDefenseLoses(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	runToEnd(w, nil)
	if w.Phase != PhaseLost {
		t.Fatalf("expected a loss, got phase %d", w.Phase)
	}
}

func TestArmorAndFlying(t *testing.T) {
	if d := Walker.EffDamage(10); d != 4 {
		t.Fatalf("walker should take 10-6=4, got %g", d)
	}
	if d := Walker.EffDamage(5); d != 1 {
		t.Fatalf("armour must let 20%% through, got %g", d)
	}
	if Blaster.CanTarget(Drone) || !Blaster.CanTarget(Walker) {
		t.Fatal("blasters shoot ground enemies only")
	}
	w := NewWorld(mustLevel(t), 1)
	w.Build(1, Blaster)
	w.Build(3, Blaster)
	w.SkipTimer()
	w.pending = w.pending[:0]
	d := w.spawn(Drone, 1, 0, false)
	for i := 0; i < 60*80 && d.alive; i++ {
		w.Update(1.0 / 60)
		if d.HP < d.MaxHP {
			t.Fatal("a drone got hit by a ground turret")
		}
	}
	if w.Leaks != 1 {
		t.Fatalf("the drone should reach the base, leaks=%d", w.Leaks)
	}
}

func TestHealAndBrood(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	w.SkipTimer()
	w.pending = w.pending[:0]
	c := w.spawn(Walker, 1, 3, false)
	c.HP = 40
	w.spawn(Drone, 1, 3, false)
	w.Update(1.5)
	if c.HP <= 40 {
		t.Fatal("the drone must repair the walker next to it")
	}
	q := w.spawn(Queen, 1, 5, false)
	n := len(w.Enemies)
	w.Update(Queen.Brood.Period + 0.1)
	if len(w.Enemies) < n+Queen.Brood.Count || q.HP != q.MaxHP {
		t.Fatalf("the queen must lay %d swarmlings, enemies %d -> %d", Queen.Brood.Count, n, len(w.Enemies))
	}
}

func TestWaves(t *testing.T) {
	if len(Level1.Waves) != 8 {
		t.Fatal("level 1 has 8 waves")
	}
	for i, wv := range Level1.Waves {
		for _, g := range wv.Groups {
			if EnemyDefs[g.Enemy] == nil {
				t.Fatalf("wave %d: unknown enemy %q", i+1, g.Enemy)
			}
		}
		t.Logf("wave %d: %d enemies %v", i+1, wv.Total(), wv.Summary())
	}
	if !Level1.Waves[7].Has(Queen) {
		t.Fatal("the queen comes in the final wave")
	}
}

func TestBuildAndUpgrade(t *testing.T) {
	w := NewWorld(mustLevel(t), 1)
	tw, err := w.Build(0, Blaster)
	if err != nil || w.Credits != Level1.StartCredits-Blaster.Levels[0].Cost {
		t.Fatal("build failed", err)
	}
	if _, err := w.Build(0, Blaster); err != ErrSlot {
		t.Fatal("pad must be occupied")
	}
	if err := w.Upgrade(tw); err != ErrBusy {
		t.Fatal("cannot upgrade during construction", err)
	}
	finish(w)
	if err := w.Upgrade(tw); err != nil || tw.JobLevel() != 1 {
		t.Fatal("upgrade failed", err)
	}
	finish(w)
	if tw.Level != 1 {
		t.Fatal("upgrade did not finish", tw.Level)
	}
	if err := w.Upgrade(tw); err != ErrNoCredits {
		t.Fatal("expected not enough credits", err)
	}
}

// --- balance simulation -------------------------------------------------

type bot func(w *World)

// step is one move of a scripted player: build def on a pad, or upgrade it (def == nil).
type step struct {
	slot   int
	def    *TowerDef
	strong bool // aim at the strongest enemy
}

func B(slot int, d *TowerDef) step { return step{slot: slot, def: d} }
func U(slot int) step              { return step{slot: slot} }

// S builds a turret that aims at the strongest enemy.
func S(slot int, d *TowerDef) step { return step{slot: slot, def: d, strong: true} }

// planBot follows a build order, saving up for each move in turn.
func planBot(plan []step) bot {
	i := 0
	return func(w *World) {
		for i < len(plan) {
			s := plan[i]
			if s.def != nil {
				if w.SlotTower[s.slot] == nil {
					t, err := w.Build(s.slot, s.def)
					if err != nil {
						return
					}
					if s.strong && w.CanTarget() {
						t.Mode = TargetStrong
					}
				}
			} else if t := w.SlotTower[s.slot]; t != nil && !t.MaxLevel() {
				if err := w.Upgrade(t); err != nil {
					return
				}
			}
			i++
		}
	}
}

// Thoughtful mixed defence: a turret for every job.
var planMixed = []step{
	B(5, Blaster), B(1, Blaster), B(3, Flamer), U(5), U(1), B(6, Cryo), B(4, Skyguard), B(2, Railgun),
	U(3), U(5), U(1), U(4), U(2), U(6), B(7, Blaster), U(3), U(2), U(7), U(7), U(5), U(3), U(4), U(4),
	B(9, Blaster), U(9), U(9), U(6), U(6), U(2), U(1), U(7), B(8, Skyguard), U(8), U(8), B(0, Blaster), U(0), U(0), U(9),
}

// Two railguns against armour, supers on the key turrets.
var planRails = []step{
	B(5, Blaster), B(1, Blaster), B(3, Flamer), U(5), U(1), B(6, Railgun), B(4, Skyguard), U(6), B(2, Railgun),
	U(3), U(5), U(1), U(4), U(2), U(6), U(6), B(7, Cryo), U(3), U(3), U(5), U(2), U(5), U(1), U(4),
	B(9, Blaster), U(9), U(9), U(2), B(8, Skyguard), U(8), U(7), U(7), U(1), B(0, Blaster), U(0), U(0), U(4), U(9), U(8),
}

// Frost in the middle of the loop, fire and rails around it.
var planCryo = []step{
	B(1, Blaster), B(3, Blaster), B(5, Cryo), U(1), U(3), B(6, Flamer), B(4, Skyguard), B(2, Railgun), U(5),
	U(6), U(1), U(3), U(4), U(2), B(7, Railgun), U(5), U(6), U(2), U(7), U(5), U(4), U(7), U(6), U(1), U(3), U(2),
	B(9, Blaster), U(9), U(9), B(8, Skyguard), U(8), B(0, Blaster), U(0), U(0), U(7), U(1), U(3), U(4), U(9), U(8),
}

// Blasters only (plus nothing against drones).
var planBlasters = []step{
	B(5, Blaster), B(1, Blaster), B(3, Blaster), U(5), U(1), B(6, Blaster), B(2, Blaster), U(3), U(5), U(1),
	B(4, Blaster), U(6), U(2), U(3), B(7, Blaster), U(4), U(6), U(2), B(9, Blaster), U(7), U(4), U(9), B(8, Blaster),
	U(7), U(9), U(8), U(8), B(0, Blaster), U(0), U(0), U(5), U(1), U(3), U(6), U(2), U(4),
}

// Blasters with anti-air cover.
var planBlastersAA = []step{
	B(5, Blaster), B(1, Blaster), B(3, Blaster), U(5), U(1), B(6, Blaster), B(4, Skyguard), B(2, Blaster), U(3), U(5), U(1),
	U(4), U(6), U(2), U(3), B(7, Blaster), U(4), U(6), U(2), B(9, Blaster), U(7), B(8, Skyguard), U(9), U(8),
	U(7), U(9), U(8), B(0, Blaster), U(0), U(0), U(5), U(1), U(3), U(6), U(2), U(4),
}

// casualBot is an average player: decent pads (with some noise), blasters first,
// then random turret types, an anti-air turret once drones show up (usually),
// and random upgrades.
func casualBot(l *Level, seed int64) bot {
	r := rand.New(rand.NewSource(seed))
	type pad struct {
		i int
		v float64
	}
	var pads []pad
	for i, s := range l.Slots {
		pads = append(pads, pad{i, l.Coverage(s.Center(), 3) * (0.6 + 0.8*r.Float64())})
	}
	sort.Slice(pads, func(a, b int) bool { return pads[a].v > pads[b].v })
	types := []*TowerDef{Blaster, Blaster, Cryo, Flamer, Flamer, Railgun, Railgun, Skyguard}
	var want *TowerDef
	count := func(w *World, d *TowerDef) int {
		n := 0
		for _, t := range w.Towers {
			if t.Def == d {
				n++
			}
		}
		return n
	}
	return func(w *World) {
		for {
			n := len(w.Towers)
			if want == nil {
				want = types[r.Intn(len(types))]
				if n < 2 {
					want = Blaster
				}
				if (want == Cryo && count(w, Cryo) >= 2) || (want == Skyguard && count(w, Skyguard) >= 2) {
					want = Blaster
				}
				if w.Wave >= 5 && count(w, Skyguard) == 0 && r.Intn(4) > 0 {
					want = Skyguard
				}
			}
			if n < len(l.Slots) && (n < 4 || r.Intn(3) == 0) {
				if _, err := w.Build(pads[n].i, want); err != nil {
					return
				}
				want = nil
				continue
			}
			var idle []*Tower
			for _, t := range w.Towers {
				if !t.Busy && !t.MaxLevel() {
					idle = append(idle, t)
				}
			}
			if len(idle) == 0 {
				if n < len(l.Slots) {
					if _, err := w.Build(pads[n].i, want); err == nil {
						want = nil
						continue
					}
				}
				return
			}
			if w.Upgrade(idle[r.Intn(len(idle))]) != nil {
				return
			}
		}
	}
}

// lazyBot builds three blasters on the best pads and never upgrades.
func lazyBot(l *Level) bot {
	return planBot([]step{B(5, Blaster), B(1, Blaster), B(3, Blaster)})
}

type waveLog struct {
	hp, credits, spent int
}

// leakLog counts base damage per enemy type in the last simulated runs.
var leakLog = map[string]int{}

// railLog counts railgun shots by target type in the last run.
var railLog = map[string]int{}

func runToEnd(w *World, b bot) []waveLog {
	var log []waveLog
	wave := 0
	clear(leakLog)
	clear(railLog)
	for i := 0; i < 400000 && w.Active(); i++ {
		if b != nil && i%30 == 0 {
			b(w)
		}
		w.Update(1.0 / 60)
		for _, ev := range w.Events {
			if ev.Kind == EvLeak {
				leakLog[ev.Enemy.ID] += ev.Value
			}
			if ev.Kind == EvRail {
				railLog[ev.Enemy.ID]++
			}
		}
		w.Events = w.Events[:0]
		if w.Wave != wave {
			wave = w.Wave
			spent := 0
			for _, t := range w.Towers {
				spent += t.Spent
			}
			log = append(log, waveLog{w.BaseHP, w.Credits, spent})
		}
	}
	return log
}

func TestBalanceReport(t *testing.T) {
	l := mustLevel(t)
	type res struct {
		name string
		w    *World
	}
	var results []res
	run := func(name string, b bot, seed int64, verbose bool) *World {
		w := NewWorld(l, seed)
		lg := runToEnd(w, b)
		results = append(results, res{name, w})
		if verbose {
			for i, x := range lg {
				t.Logf("  %s: wave %d starts with hp=%d credits=%d invested=%d", name, i+1, x.hp, x.credits, x.spent)
			}
		}
		return w
	}
	mixed := run("mixed", planBot(planMixed), 1, true)
	blasters := run("blasters", planBot(planBlasters), 1, false)
	blastersAA := run("blast+AA", planBot(planBlastersAA), 1, true)
	lazy := run("lazy", lazyBot(l), 1, false)
	casualWins := 0
	for s := int64(1); s <= 16; s++ {
		if run(fmt.Sprintf("casual-%d", s), casualBot(l, s), s, false).Phase == PhaseWon {
			casualWins++
		}
	}
	for _, r := range results {
		types := map[string]int{}
		for _, tw := range r.w.Towers {
			types[tw.Def.ID]++
		}
		t.Logf("%-10s phase=%d wave=%d hp=%3d kills=%3d leaks=%2d credits=%4d towers=%v",
			r.name, r.w.Phase, r.w.Wave, r.w.BaseHP, r.w.Kills, r.w.Leaks, r.w.Credits, types)
	}
	if mixed.Phase != PhaseWon {
		t.Errorf("a thoughtful mixed defence must win")
	}
	if lazy.Phase == PhaseWon {
		t.Errorf("three plain turrets must not be enough to win")
	}
	_ = blasters
	_ = blastersAA
	t.Logf("casual wins: %d/16", casualWins)
}
