package game

import (
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestLevelsParse(t *testing.T) {
	ids := map[string]bool{}
	for _, spec := range AllLevels() {
		l, err := ParseLevel(spec)
		if err != nil {
			t.Fatalf("%s: %v", spec.ID, err)
		}
		if ids[spec.ID] {
			t.Fatalf("duplicate level id %s", spec.ID)
		}
		ids[spec.ID] = true
		for i, r := range l.Routes {
			if sp := r.Spawn; sp.X != 0 && sp.Y != 0 && sp.X != l.W-1 && sp.Y != l.H-1 {
				t.Errorf("%s: burrow %d must be on the edge of the map", spec.ID, i+1)
			}
			// The base has a gate on the west side, and one on the east side
			// when a road comes from there: roads stay on their side.
			last := r.Cells[len(r.Cells)-1]
			mid := l.BaseMin.Y + l.BaseSize/2
			east := r.EndDir.X < 0
			switch {
			case !east && (last.X != l.BaseMin.X-1 || last.Y != mid),
				east && (last.X != l.BaseMin.X+l.BaseSize || last.Y != mid):
				t.Errorf("%s: road %d must enter a base gate (west or east side, middle row)", spec.ID, i+1)
			}
			for _, c := range r.Cells {
				if !east && c.X >= l.BaseMin.X || east && c.X < l.BaseMin.X+l.BaseSize {
					t.Errorf("%s: road cell %v crosses to the other side of the base", spec.ID, c)
				}
			}
		}
		used := make([]bool, len(l.Routes))
		for wi, wv := range spec.Waves {
			for _, g := range wv.Groups {
				if g.From < 0 || g.From >= len(l.Routes) {
					t.Errorf("%s: wave %d sends %s through missing entrance %d", spec.ID, wi+1, g.Enemy, g.From)
					continue
				}
				used[g.From] = true
				if EnemyDefs[g.Enemy] == nil {
					t.Errorf("%s: wave %d: unknown enemy %s", spec.ID, wi+1, g.Enemy)
				}
			}
		}
		for i, u := range used {
			if !u {
				t.Errorf("%s: no wave uses entrance %d", spec.ID, i+1)
			}
		}
		for _, d := range spec.Arsenal() {
			if _, ok := TowerDefs[d.ID]; !ok {
				t.Errorf("%s: unknown turret %s", spec.ID, d.ID)
			}
		}
		if spec.Lesson != nil && spec.Lesson.Turret != "" && !spec.Allows(TowerDefs[spec.Lesson.Turret]) {
			t.Errorf("%s: the lesson turret must be buildable", spec.ID)
		}
		t.Logf("%-6s %-15s %2dx%-2d path %5.1f pads %2d waves %d", spec.ID, spec.Name, l.W, l.H, l.Path.Len, len(l.Slots), len(spec.Waves))
	}
}

func TestLocks(t *testing.T) {
	l, _ := ParseLevel(Landing)
	w := NewWorld(l, 1)
	w.Credits = 1000
	if _, err := w.Build(0, Cryo); err != ErrLocked {
		t.Fatal("cryo is locked on the first level")
	}
	tw, err := w.Build(0, Blaster)
	if err != nil {
		t.Fatal(err)
	}
	finish(w)
	w.CycleMode(tw)
	if tw.Mode != TargetFirst {
		t.Fatal("target priority is locked")
	}
	for i := 0; i < 2; i++ {
		if err := w.Upgrade(tw); err != nil {
			t.Fatal(err)
		}
		finish(w)
	}
	if !tw.MaxLevel() || tw.NextIsSuper() || w.Upgrade(tw) != ErrMaxLevel {
		t.Fatal("level 3 is the top level while supers are locked")
	}
	w.Hold = true
	t0 := w.Timer
	w.Update(1)
	if w.Timer != t0 {
		t.Fatal("hold keeps the first wave timer")
	}
}

func TestStars(t *testing.T) {
	w := &World{Phase: PhaseWon, BaseMaxHP: 100}
	for _, c := range [][2]int{{100, 3}, {90, 3}, {89, 2}, {50, 2}, {49, 1}, {1, 1}} {
		w.BaseHP = c[0]
		if s := w.Stars(); s != c[1] {
			t.Errorf("hp %d: %d stars, want %d", c[0], s, c[1])
		}
	}
	w.Phase = PhaseLost
	if w.Stars() != 0 {
		t.Error("a loss has no stars")
	}
}

// --- tutorial balance -----------------------------------------------------

// padOrder sorts the pads by how much road they see within r, best first.
func padOrder(l *Level, r float64, noise float64, rng *rand.Rand) []int {
	type pad struct {
		i int
		v float64
	}
	var pads []pad
	for i, s := range l.Slots {
		v := l.Coverage(s.Center(), r)
		if rng != nil {
			v *= 1 - noise + 2*noise*rng.Float64()
		}
		pads = append(pads, pad{i, v})
	}
	sort.SliceStable(pads, func(a, b int) bool { return pads[a].v > pads[b].v })
	out := make([]int, len(pads))
	for i, p := range pads {
		out[i] = p.i
	}
	return out
}

// goodBot builds the composition on the best pads (in order), then keeps
// upgrading the weakest turret, the best pads first. Railguns aim at the
// strongest enemy when the level allows it.
func goodBot(l *Level, comp []*TowerDef) bot {
	pads := padOrder(l, 3, 0, nil)
	return func(w *World) {
		for {
			n := len(w.Towers)
			if n < len(comp) && n < len(pads) {
				// Build the next turret once the earlier ones are working, or when rich.
				if _, err := w.Build(pads[n], comp[n]); err != nil {
					return
				}
				if comp[n] == Railgun && w.CanTarget() {
					w.SlotTower[pads[n]].Mode = TargetStrong
				}
				continue
			}
			var best *Tower
			for _, p := range pads {
				t := w.SlotTower[p]
				if t == nil || t.Busy || t.MaxLevel() {
					continue
				}
				if best == nil || t.Level < best.Level {
					best = t
				}
			}
			if best == nil || w.Upgrade(best) != nil {
				return
			}
		}
	}
}

// splitCasual lists the levels whose roads never meet: an average player who
// read the briefing builds on the road that has the least cover so far.
var splitCasual = map[string]bool{}

// casualFor is an average player with a random mix of the given turret types.
func casualFor(l *Level, seed int64, types []*TowerDef) bot {
	r := rand.New(rand.NewSource(seed))
	pads := padOrder(l, 3, 0.4, r)
	if splitCasual[l.Spec.ID] && os.Getenv("NOSPLIT") == "" {
		return splitCasualBot(l, r, types)
	}
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
				if n < 1 || ((want == Cryo || want == Skyguard) && count(w, want) >= 2) {
					want = Blaster
				}
				// A player who met walkers or drones before builds their counter early.
				for _, c := range counters {
					if n == c.by && count(w, c.turret) == 0 && hasEnemy(l, c.enemy) && slices.Contains(types, c.turret) {
						want = c.turret
					}
				}
			}
			if n < len(pads) && (n < 3 || r.Intn(3) == 0) {
				t, err := w.Build(pads[n], want)
				if err != nil {
					return
				}
				strongRail(w, t)
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
				if n < len(pads) {
					if t, err := w.Build(pads[n], want); err == nil {
						strongRail(w, t)
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

// strongRail sets a new Railgun to Strong, as the target priority lesson says.
func strongRail(w *World, t *Tower) {
	if t.Def == Railgun && w.CanTarget() && os.Getenv("CASUALFIRST") == "" {
		t.Mode = TargetStrong
	}
}

// counters are the turrets an experienced player builds against an enemy type,
// as their build number "by" at the latest.
var counters = []struct {
	enemy  string
	turret *TowerDef
	by     int
}{{"walker", Railgun, 2}, {"drone", Skyguard, 3}}

func hasEnemy(l *Level, id string) bool {
	for _, wv := range l.Spec.Waves {
		for _, g := range wv.Groups {
			if g.Enemy == id {
				return true
			}
		}
	}
	return false
}

func summary(w *World) string {
	lv := ""
	for _, t := range w.Towers {
		lv += fmt.Sprintf("%c%d", t.Def.ID[0], t.Level+1)
	}
	return fmt.Sprintf("%s w%d hp%3d %s", map[Phase]string{PhaseWon: "WON ", PhaseLost: "LOST"}[w.Phase], w.Wave, w.BaseHP, lv)
}

func casualStats(l *Level, types []*TowerDef, seeds int) string {
	wins, hp := 0, 0
	stars := [4]int{}
	for s := int64(1); s <= int64(seeds); s++ {
		w := NewWorld(l, s)
		runToEnd(w, casualFor(l, s, types))
		if w.Phase == PhaseWon {
			wins++
			hp += w.BaseHP
		}
		stars[w.Stars()]++
	}
	return fmt.Sprintf("%2d/%d wins avg hp %3d stars 0:%d 1:%d 2:%d 3:%d", wins, seeds, hp/max(1, wins), stars[0], stars[1], stars[2], stars[3])
}

type tutorialCase struct {
	spec    *LevelSpec
	good    []*TowerDef // a sensible composition with the new turret
	without []*TowerDef // the same budget without it
	casual  []*TowerDef // random picks of an average player
	old     []*TowerDef // random picks without the new turret
}

var bl, cr, fl, rl = Blaster, Cryo, Flamer, Railgun

var tutorialCases = []tutorialCase{
	{Landing, []*TowerDef{bl, bl, bl, bl}, nil, []*TowerDef{bl}, nil},
	{Canyon, []*TowerDef{bl, cr, bl, bl, bl}, []*TowerDef{bl, bl, bl, bl, bl},
		[]*TowerDef{bl, bl, cr}, []*TowerDef{bl}},
	{Hatchery, []*TowerDef{bl, fl, cr, bl, fl, bl}, []*TowerDef{bl, bl, cr, bl, bl, bl},
		[]*TowerDef{bl, bl, cr, fl, fl}, []*TowerDef{bl, bl, cr}},
	{IronPass, []*TowerDef{bl, rl, cr, bl, rl, bl}, []*TowerDef{bl, bl, cr, bl, fl, bl},
		[]*TowerDef{bl, bl, cr, fl, rl, rl}, []*TowerDef{bl, bl, cr, fl}},
	{Convoy, []*TowerDef{bl, rl, bl, cr, rl, fl, bl}, []*TowerDef{bl, bl, bl, cr, bl, fl, bl},
		[]*TowerDef{bl, bl, cr, fl, rl, rl}, nil},
	{Valley, []*TowerDef{bl, sk, bl, rl, cr, bl, sk, fl}, []*TowerDef{bl, bl, bl, rl, cr, bl, bl, fl},
		[]*TowerDef{bl, bl, cr, fl, rl, sk, sk}, []*TowerDef{bl, bl, cr, fl, rl}},
	{HiveHeart, []*TowerDef{fl, bl, rl, fl, sk, cr, rl, bl, sk, bl}, nil,
		[]*TowerDef{bl, bl, cr, fl, fl, rl, rl, sk}, nil},
}

// TestTutorialBalance: BAL=1 go test -run TestTutorialBalance -v
func TestTutorialBalance(t *testing.T) {
	only := os.Getenv("BAL")
	for _, c := range tutorialCases {
		if only == "" && testing.Short() {
			continue
		}
		if only != "" && only != "1" && !strings.Contains(only, c.spec.ID) {
			continue
		}
		l, _ := ParseLevel(c.spec)
		run := func(b bot) *World {
			w := NewWorld(l, 1)
			runToEnd(w, b)
			return w
		}
		plans := tunePlans[c.spec.ID]
		good := run(goodBot(l, c.good))
		if len(plans) > 0 {
			good = run(planUp(l, plans[0].steps, false))
		}
		t.Logf("       good leaks %v", leakLog)
		lazy := run(goodBot(l, []*TowerDef{Blaster, Blaster}))
		lazyW := NewWorld(l, 1)
		runToEnd(lazyW, planBot([]step{B(padOrder(l, 3, 0, nil)[0], Blaster), B(padOrder(l, 3, 0, nil)[1], Blaster), B(padOrder(l, 3, 0, nil)[2], Blaster)}))
		none := run(nil)
		goodLeaks := fmt.Sprint(leakLog)
		_ = goodLeaks
		t.Logf("%s %-13s good %-32s | 2bl+up %-22s | 3bl %s | none %s", c.spec.ID, c.spec.Name, summary(good), summary(lazy), summary(lazyW), summary(none))
		if len(plans) > 1 {
			lv := l
			if plans[1].name == "nosuper" {
				lv = noSuper(c.spec)
			}
			w := NewWorld(lv, 1)
			runToEnd(w, planUp(lv, plans[1].steps, false))
			t.Logf("       without new: %s leaks %v", summary(w), leakLog)
		}
		t.Logf("       casual %s", casualStats(l, c.casual, 24))
		if c.old != nil {
			t.Logf("       casual old %s", casualStats(l, c.old, 24))
		}
		if os.Getenv("BAL") == "" {
			if good.Stars() < 3 {
				t.Errorf("%s: a sensible defence must get three stars", c.spec.ID)
			}
			if none.Phase != PhaseLost {
				t.Errorf("%s: no defence must lose", c.spec.ID)
			}
		}
	}
}

// splitCasualBot is casualFor on a level with separate roads: each new turret
// goes to the best free pad (by its noisy view of the road) of the road with
// the least cover.
func splitCasualBot(l *Level, r *rand.Rand, types []*TowerDef) bot {
	cov := make([][]float64, len(l.Slots)) // pad -> route -> road seen
	for i, s := range l.Slots {
		k := 0.6 + 0.8*r.Float64()
		for _, rt := range l.Routes {
			cov[i] = append(cov[i], k*rt.Path.Coverage(s.Center(), 3))
		}
	}
	// The player reads the next-wave panel: a counter turret goes to the road
	// that sends the most of its enemy (fewer of the same turret there first).
	foes := map[*TowerDef]map[string]float64{
		Railgun: {"walker": 1}, Skyguard: {"drone": 1}, Flamer: {"beetle": 1, "swarmling": 0.2},
	}
	weight := func(d *TowerDef, ri int) float64 {
		v := 0.0
		for _, wv := range l.Spec.Waves {
			for _, g := range wv.Groups {
				if g.entrance(len(l.Routes)) == ri {
					v += foes[d][g.Enemy] * float64(g.Count)
				}
			}
		}
		return v
	}
	next := func(w *World, d *TowerDef) int {
		cover := make([]float64, len(l.Routes))
		same := make([]float64, len(l.Routes))
		for _, t := range w.Towers {
			for ri := range l.Routes {
				cover[ri] += cov[t.Slot][ri]
				if t.Def == d && cov[t.Slot][ri] > 0 {
					same[ri]++
				}
			}
		}
		weak := 0
		for ri := range cover {
			if cover[ri] < cover[weak] {
				weak = ri
			}
		}
		if foes[d] != nil {
			bestV := 0.0
			for ri := range l.Routes {
				if v := weight(d, ri) / (1 + same[ri]); v > bestV {
					weak, bestV = ri, v
				}
			}
		}
		best := -1
		for i := range l.Slots {
			if w.SlotTower[i] == nil && cov[i][weak] > 0 && (best < 0 || cov[i][weak] > cov[best][weak]) {
				best = i
			}
		}
		if best < 0 {
			for i := range l.Slots {
				if w.SlotTower[i] == nil {
					return i
				}
			}
		}
		return best
	}
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
				if n < 1 || ((want == Cryo || want == Skyguard) && count(w, want) >= 2) {
					want = Blaster
				}
				for _, c := range counters {
					if n == c.by && count(w, c.turret) == 0 && hasEnemy(l, c.enemy) && slices.Contains(types, c.turret) {
						want = c.turret
					}
				}
			}
			free := n < len(l.Slots)
			if free && (n < 3 || r.Intn(3) == 0) {
				t, err := w.Build(next(w, want), want)
				if err != nil {
					return
				}
				strongRail(w, t)
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
				if free {
					if t, err := w.Build(next(w, want), want); err == nil {
						strongRail(w, t)
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
