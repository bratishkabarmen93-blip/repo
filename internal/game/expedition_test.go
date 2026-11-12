package game

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func expCampaign() []*LevelSpec {
	var out []*LevelSpec
	for _, c := range Chapters {
		out = append(out, c.Levels...)
	}
	return out
}

// expKnow is what a player knows after winning the first n campaign levels.
func expKnow(n int) Knowledge {
	won := map[*LevelSpec]bool{}
	for _, s := range expCampaign()[:n] {
		won[s] = true
	}
	return Learn(func(s *LevelSpec) bool { return won[s] })
}

func TestExpeditionValid(t *testing.T) {
	seeds := 150
	if testing.Short() {
		seeds = 25
	}
	camp := expCampaign()
	seen := map[string]int{}
	for n := 1; n <= len(camp); n++ {
		k := expKnow(n)
		for seed := int64(1); seed <= int64(seeds); seed++ {
			e := NewExpedition(seed, k)
			s := e.Spec
			where := fmt.Sprintf("after %s, seed %d", camp[n-1].ID, seed)
			l, err := ParseLevel(s)
			if err != nil {
				t.Fatalf("%s: %v\n%s", where, err, strings.Join(s.Map, "\n"))
			}
			for _, d := range s.Arsenal() {
				if !k.hasTurret(d) {
					t.Errorf("%s: turret %s is not known yet", where, d.ID)
				}
			}
			for _, d := range s.Enemies() {
				switch {
				case !k.knows(d) && !slices.Contains(Offspring, d):
					t.Errorf("%s: enemy %s is not known yet", where, d.ID)
				case d.Flying && !k.hasTurret(Skyguard):
					t.Errorf("%s: %s flies, but there is no Skyguard", where, d.ID)
				case d.Stealth && !k.hasTurret(Beacon):
					t.Errorf("%s: %s hides in the fog, but there is no Beacon", where, d.ID)
				case (d == Bloater || d == Stalker) && e.World != "Titan":
					t.Errorf("%s: %s off Titan, on %s", where, d.ID, e.World)
				case d.Boss && d != e.Boss:
					t.Errorf("%s: stray boss %s", where, d.ID)
				}
			}
			if e.Boss != nil && map[string]*EnemyDef{"Mars": Queen, "Europa": Wyrm, "Titan": Kraken}[e.World] != e.Boss {
				t.Errorf("%s: %s on %s", where, e.Boss.ID, e.World)
			}
			if e.World == "Europa" && !k.Europa || (s.Biome == "ice") != (e.World == "Europa") {
				t.Errorf("%s: world %s, biome %q", where, e.World, s.Biome)
			}
			if e.World == "Titan" && !k.Titan || (s.Biome == "titan") != (e.World == "Titan") || (len(s.Fog) >= 2) != (e.World == "Titan") {
				t.Errorf("%s: world %s, biome %q, %d fog banks", where, e.World, s.Biome, len(s.Fog))
			}
			for _, b := range s.Fog {
				for _, p := range b.Stops {
					if len(b.Stops) < 2 || b.R < 1.5 || p.X < 0 || p.Y < 0 || p.X > float64(len(s.Map[0])) || p.Y > float64(len(s.Map)) {
						t.Errorf("%s: fog bank %+v off the map", where, b)
					}
				}
			}
			if l.Entrances() > k.Entrances || l.EastGate() && !k.Fronts {
				t.Errorf("%s: %d entrances (east gate %v) is more than known", where, l.Entrances(), l.EastGate())
			}
			if s.NoTarget == k.Target || s.NoSuper == k.Super {
				t.Errorf("%s: abilities do not match what is known", where)
			}
			if len(l.Slots) < 8 {
				t.Errorf("%s: only %d pads", where, len(l.Slots))
			}
			for i, r := range l.Routes {
				if len(r.Cells) < 22 {
					t.Errorf("%s: road %d has only %d cells", where, i, len(r.Cells))
				}
			}
			for wi, wv := range s.Waves {
				for _, g := range wv.Groups {
					if g.Count < 1 || g.HPMul <= 0 || g.Interval <= 0 || g.From < 0 || g.From >= l.Entrances() {
						t.Errorf("%s: wave %d: bad group %+v", where, wi+1, g)
					}
				}
			}
			if e2 := NewExpedition(seed, k); !reflect.DeepEqual(e, e2) {
				t.Fatalf("%s: the same seed gave another expedition", where)
			}
			boss := "no boss"
			if e.Boss != nil {
				boss = e.Boss.ID
			}
			seen[e.World+" "+e.Layout]++
			seen[e.Theme]++
			seen[boss]++
			seen[s.Tier.String()]++
		}
	}
	var keys []string
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%-22s %d", k, seen[k])
	}
}

// TestExpeditionShow prints expeditions: EXPSHOW=<stage>:<seed> go test -run TestExpeditionShow -v
func TestExpeditionShow(t *testing.T) {
	v := os.Getenv("EXPSHOW")
	if v == "" {
		t.Skip("EXPSHOW=<stage>:<seed> prints an expedition")
	}
	for _, one := range strings.Split(v, ",") {
		st, sd, _ := strings.Cut(one, ":")
		n, _ := strconv.Atoi(st)
		seed, _ := strconv.ParseInt(sd, 10, 64)
		e := NewExpedition(seed, expKnow(n))
		s := e.Spec
		t.Logf("%s (%s, %s, %s, threat %s): %s\n  %s", s.Name, e.World, e.Layout, e.Theme, s.Tier, s.Desc, strings.Join(s.Map, "\n  "))
		t.Logf("credits %d, bonus %d, prep %v/%v, turrets %v", s.StartCredits, s.WaveBonus, s.FirstPrep, s.Prep, s.Turrets)
		for i, wv := range s.Waves {
			var gs []string
			for _, g := range wv.Groups {
				gs = append(gs, fmt.Sprintf("%d %s x%.2f @%.1fs via %d", g.Count, g.Enemy, g.HPMul, g.Delay, g.From))
			}
			t.Logf("  wave %d (%.0f): %s", i+1, expBudget(wv), strings.Join(gs, ", "))
		}
	}
}

// TestExpeditionCurves prints the wave threats of the campaign: EXPCURVES=1
func TestExpeditionCurves(t *testing.T) {
	if os.Getenv("EXPCURVES") == "" {
		t.Skip("EXPCURVES=1 prints the campaign wave threats")
	}
	for i, s := range expCampaign() {
		var bs []string
		for _, wv := range s.Waves {
			bs = append(bs, fmt.Sprintf("%.0f", expBudget(wv)))
		}
		l, n := expRoadLen(s)
		pl, _ := ParseLevel(s)
		t.Logf("%-2d %-13s %-6s cr %3d bonus %2d road %2.0f x%d pads %2d: %s", i+1, s.ID, s.Tier, s.StartCredits, s.WaveBonus, l, n, len(pl.Slots), strings.Join(bs, " "))
	}
}

// expGoodComp is the build order of a good player for an expedition.
func expGoodComp(l *Level, types []*TowerDef) []*TowerDef {
	flyers := false
	for _, d := range l.Spec.Enemies() {
		flyers = flyers || d.Flying
	}
	pattern := []*TowerDef{bl, bl, cr, fl, rl, sk, bl, rl, fl, sk, bl, cr, rl, fl, bl, sk}
	var comp []*TowerDef
	for _, d := range pattern {
		if slices.Contains(types, d) && (d != Skyguard || flyers) {
			comp = append(comp, d)
		}
	}
	for len(comp) < len(l.Slots) {
		comp = append(comp, bl)
	}
	return comp[:len(l.Slots)]
}

func expStages() []int {
	var out []int
	for i := range expCampaign() {
		out = append(out, i+1)
	}
	if v := os.Getenv("EXPSTAGES"); v != "" {
		out = nil
		for _, f := range strings.Split(v, ",") {
			n, _ := strconv.Atoi(f)
			out = append(out, n)
		}
	}
	return out
}

// TestExpeditionRef measures the campaign with the same bots, as the yardstick
// of the expedition balance: EXPREF=1 go test -run TestExpeditionRef -v
func TestExpeditionRef(t *testing.T) {
	if os.Getenv("EXPREF") == "" {
		t.Skip("EXPREF=1 measures the campaign levels")
	}
	camp := expCampaign()
	for _, n := range expStages() {
		s := camp[n-1]
		l, _ := ParseLevel(s)
		types := s.Arsenal()
		const runs = 12
		res := make([]runRes, runs+1)
		parallel(runs+1, func(i int) {
			if i == runs {
				res[i] = simulate(l, 3, goodBot(l, expGoodComp(l, types)))
				return
			}
			res[i] = simulate(l, int64(i%2+1), casualFor(l, int64(100+i), types))
		})
		wins, stars := 0, 0
		for _, r := range res[:runs] {
			if r.won {
				wins++
				stars += r.stars
			}
		}
		t.Logf("%-2d %-14s %-6s casual %3d%% (%d stars)  good %v %d", n, s.ID, s.Tier, wins*100/runs, stars, res[runs].won, res[runs].stars)
	}
}

func expEnv(name string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil {
		return v
	}
	return def
}

// TestExpeditionDebug plays one expedition with the bots and shows it:
// EXPDBG=13:7249:2 go test -run TestExpeditionDebug -v (stage:seed:tier)
func TestExpeditionDebug(t *testing.T) {
	v := strings.Split(os.Getenv("EXPDBG"), ":")
	if len(v) != 3 {
		t.Skip("EXPDBG=stage:seed:tier plays one expedition")
	}
	n, _ := strconv.Atoi(v[0])
	seed, _ := strconv.ParseInt(v[1], 10, 64)
	tier, _ := strconv.Atoi(v[2])
	expTune.Scale = expEnv("EXPSCALE", 1)
	defer func() { expTune.Scale = 1 }()
	e := newExpedition(seed, expKnow(n), Tier(tier))
	l, err := ParseLevel(e.Spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s · %s · %s · %s, credits %d bonus %d base %d pads %d", e.Spec.Name, e.Layout, e.Theme, Tier(tier),
		e.Spec.StartCredits, e.Spec.WaveBonus, e.Spec.BaseHP, len(l.Slots))
	for _, row := range e.Spec.Map {
		t.Log(row)
	}
	for i, rt := range l.Routes {
		t.Logf("route %d: %d cells from %v, join %v", i, len(rt.Cells), rt.Spawn, rt.Join)
	}
	for j, wv := range e.Spec.Waves {
		var gs []string
		for _, g := range wv.Groups {
			gs = append(gs, fmt.Sprintf("%d %s x%.2f via %d", g.Count, g.Enemy, g.HPMul, g.From))
		}
		t.Logf("wave %d (%.0f): %s", j+1, expBudget(wv), strings.Join(gs, ", "))
	}
	types := e.Spec.Arsenal()
	for c := 0; c < 3; c++ {
		t.Logf("casual %d: %+v", c, simulate(l, int64(c+1), expCasual(l, int64(c), types)))
	}
	t.Logf("good: %+v", simulate(l, 3, goodBot(l, expGoodComp(l, types))))
}

// expCasual is the casual bot of an expedition: on a map whose roads never
// meet it defends each road, as a player who sees both would.
func expCasual(l *Level, seed int64, types []*TowerDef) bot {
	// Without fog to clear, a casual player builds no Beacons.
	if len(l.Spec.Fog) == 0 {
		types = slices.DeleteFunc(slices.Clone(types), func(d *TowerDef) bool { return d == Beacon })
	}
	if l.EastGate() {
		return expSplitBot(l, rand.New(rand.NewSource(seed)), types)
	}
	return casualFor(l, seed, types)
}

// expSplitBot is splitCasualBot, except that only turrets that shoot the
// ground count as cover of a road: a Skyguard or a Cryo alone does not hold
// one, and Skyguards go where the flyers come.
func expSplitBot(l *Level, r *rand.Rand, types []*TowerDef) bot {
	cov := make([][]float64, len(l.Slots)) // pad -> route -> road seen
	for i, s := range l.Slots {
		k := 0.6 + 0.8*r.Float64()
		for _, rt := range l.Routes {
			cov[i] = append(cov[i], k*rt.Path.Coverage(s.Center(), 3))
		}
	}
	foes := map[*TowerDef]map[string]float64{
		Railgun: {"walker": 1}, Skyguard: {"drone": 1, "stinger": 0.3}, Flamer: {"beetle": 1, "swarmling": 0.2},
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
				if !t.Def.AntiAir && t.Def != Cryo {
					cover[ri] += cov[t.Slot][ri]
				}
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

// expCasualRate is how often the casual bot wins expeditions of a tier.
func expCasualRate(k Knowledge, tier Tier, seeds int) float64 {
	wins := make([]int, seeds)
	parallel(seeds, func(i int) {
		e := newExpedition(int64(7001+i*31), k, tier)
		l, err := ParseLevel(e.Spec)
		if err != nil {
			panic(err)
		}
		types := e.Spec.Arsenal()
		for c := 0; c < 2; c++ {
			if simulate(l, int64(c+1), expCasual(l, int64(i*7+c), types)).won {
				wins[i]++
			}
		}
	})
	sum := 0
	for _, w := range wins {
		sum += w
	}
	return float64(sum) / float64(2*seeds)
}

// TestExpeditionCalibrate fits expPressure so that the casual bot wins Medium
// expeditions as often as the target: EXPCAL=1 go test -run TestExpeditionCalibrate -v
// (EXPTARGET=0.8, EXPTIER=2, EXPSEEDS=30, EXPSTAGES=1,2).
func TestExpeditionCalibrate(t *testing.T) {
	if os.Getenv("EXPCAL") == "" {
		t.Skip("EXPCAL=1 fits the expedition pressure")
	}
	seeds := int(expEnv("EXPSEEDS", 30))
	target := expEnv("EXPTARGET", 0.8)
	tier := Tier(expEnv("EXPTIER", float64(TierMedium)))
	defer func() { expTune.Scale = 1 }()
	var table []string
	for _, n := range expStages() {
		k := expKnow(n)
		lo, hi := expEnv("EXPLO", 0.3), expEnv("EXPHI", 3)
		for it := 0; it < 7; it++ {
			mid := math.Sqrt(lo * hi)
			expTune.Scale = mid
			if expCasualRate(k, tier, seeds) >= target {
				lo = mid
			} else {
				hi = mid
			}
		}
		m := math.Sqrt(lo * hi)
		expTune.Scale = m
		got := expCasualRate(k, tier, seeds)
		expTune.Scale = 1
		table = append(table, fmt.Sprintf("%.0f", expPressureAt(n)*m))
		t.Logf("%-2d x%.2f -> %.0f (casual %.0f%%)", n, m, expPressureAt(n)*m, 100*got)
	}
	t.Logf("expPressure = []float64{%s}", strings.Join(table, ", "))
}

// TestExpeditionBalance plays generated expeditions with the test bots:
// EXPBAL=1 go test -run TestExpeditionBalance -v (EXPSTAGES=1,7,14 and
// EXPSEEDS=12 narrow it down).
func TestExpeditionBalance(t *testing.T) {
	expTune.Scale = expEnv("EXPSCALE", 1)
	defer func() { expTune.Scale = 1 }()
	if os.Getenv("EXPBAL") == "" {
		t.Skip("EXPBAL=1 runs the expedition balance")
	}
	seeds := 12
	if v, err := strconv.Atoi(os.Getenv("EXPSEEDS")); err == nil {
		seeds = v
	}
	type out struct {
		e      *Expedition
		casual [2]runRes
		good   runRes
	}
	camp := expCampaign()
	for _, n := range expStages() {
		k := expKnow(n)
		for tier := TierLow; tier <= TierHigh; tier++ {
			res := make([]out, seeds)
			parallel(seeds, func(i int) {
				e := newExpedition(int64(7001+i*31), k, tier)
				l, err := ParseLevel(e.Spec)
				if err != nil {
					panic(err)
				}
				types := e.Spec.Arsenal()
				res[i].e = e
				for c := range res[i].casual {
					res[i].casual[c] = simulate(l, int64(c+1), expCasual(l, int64(i*7+c), types))
				}
				res[i].good = simulate(l, 3, goodBot(l, expGoodComp(l, types)))
			})
			cw, cs, gw, gs := 0, 0, 0, 0
			var lost []string
			by := map[string][2]int{}
			for _, r := range res {
				w := 0
				for _, c := range r.casual {
					if c.won {
						cw++
						cs += c.stars
						w++
					}
				}
				if r.good.won {
					gw++
					gs += r.good.stars
				}
				boss := ""
				if r.e.Boss != nil {
					boss = "+" + r.e.Boss.ID
				}
				if !r.good.won {
					lost = append(lost, fmt.Sprintf("%d:%s/%s%s", r.e.Seed, r.e.Layout, r.e.Theme, boss))
				}
				for _, key := range []string{r.e.Layout, r.e.Theme, "boss" + boss} {
					v := by[key]
					v[0] += w
					v[1] += len(r.casual)
					by[key] = v
				}
			}
			var keys, parts []string
			for key := range by {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				parts = append(parts, fmt.Sprintf("%s %d/%d", key, by[key][0], by[key][1]))
			}
			cr := float64(cw) / float64(2*seeds)
			t.Logf("%-2d %-6s %-6s casual %3.0f%% (%.1f stars/win)  good %3.0f%% (%.1f)  | %s | good lost: %s",
				n, camp[n-1].ID, tier, 100*cr, float64(cs)/float64(max(cw, 1)),
				100*float64(gw)/float64(seeds), float64(gs)/float64(max(gw, 1)), strings.Join(parts, ", "), strings.Join(lost, " "))
		}
	}
}

// Pack bugs keep the campaign's tight spacing: splash turrets (Skyguard
// missiles against Stingers, Flamers against Swarmlings) are meant to hit
// several of them at once.
func TestExpeditionPacks(t *testing.T) {
	camp := expCampaign()
	for _, id := range []string{"swarmling", "stinger"} {
		var gaps []float64
		for _, s := range camp {
			for _, wv := range s.Waves {
				for _, g := range wv.Groups {
					if g.Enemy == id && g.Count >= 4 {
						gaps = append(gaps, g.Interval)
					}
				}
			}
		}
		if len(gaps) == 0 {
			t.Fatalf("the campaign has no %s packs", id)
		}
		sort.Float64s(gaps)
		med := gaps[len(gaps)/2]
		lo, hi := 0.7*med, 1.3*med
		n := 0
		for k := 1; k <= len(camp); k++ {
			know := expKnow(k)
			for seed := int64(1); seed <= 40; seed++ {
				for wi, wv := range NewExpedition(seed, know).Spec.Waves {
					for _, g := range wv.Groups {
						if g.Enemy != id {
							continue
						}
						n++
						if g.Interval < lo || g.Interval > hi {
							t.Errorf("after %s, seed %d, wave %d: %d %s %.2fs apart, the campaign's usual is %.2fs",
								camp[k-1].ID, seed, wi+1, g.Count, id, g.Interval, med)
						}
					}
				}
			}
		}
		t.Logf("%s: campaign median %.2fs apart, %d expedition groups checked", id, med, n)
	}
}

// One top-level Skyguard against a pack of 14 Stingers on Stinger Ridge, from
// every pad that sees enough of the road: the campaign's tight spacing lets
// its splash missiles take several at once.
func TestExpeditionSkyguardPack(t *testing.T) {
	var src *LevelSpec
	for _, s := range expCampaign() {
		if s.ID == "2-5" {
			src = s
		}
	}
	if src == nil {
		t.Fatal("no level 2-5")
	}
	top := len(Skyguard.Levels) - 1
	reach := Skyguard.Levels[top].Range
	// shot is how many of the 14 one Skyguard shoots down on average.
	shot := func(gaps []float64) (float64, int) {
		sum, runs, pads := 0, 0, 0
		for _, gap := range gaps {
			spec := *src
			spec.BaseHP = 100000
			spec.Waves = []WaveSpec{{Groups: []Group{g("stinger", 14, gap, 0, 1)}}}
			l, err := ParseLevel(&spec)
			if err != nil {
				t.Fatal(err)
			}
			pads = 0
			for slot, c := range l.Slots {
				if l.Path.Coverage(c.Center(), reach) < 4 {
					continue
				}
				pads++
				for seed := int64(1); seed <= 3; seed++ {
					w := NewWorld(l, seed)
					if w.PlaceTower(slot, Skyguard, top) == nil {
						t.Fatal("cannot place the Skyguard")
					}
					w.SkipTimer()
					for i := 0; i < 60*120 && w.Active(); i++ {
						w.Update(1.0 / 60)
					}
					sum += w.Kills
					runs++
				}
			}
		}
		if runs == 0 {
			t.Fatal("no pad sees the road")
		}
		return float64(sum) / float64(runs), pads
	}
	loose, pads := shot([]float64{0.45, 0.5, 0.55, 0.6, 0.65, 0.7})
	tight, _ := shot([]float64{0.25, 0.26, 0.27, 0.28, 0.29, 0.3})
	t.Logf("14 stingers against one top Skyguard (%d pads): %.1f shot down on average 0.45-0.7s apart (v0.15.0), %.1f 0.25-0.3s apart (the campaign's, v0.15.1)",
		pads, loose, tight)
	if tight <= loose {
		t.Errorf("a tight pack should lose more stingers to splash: %.1f against %.1f", tight, loose)
	}
}

// Titan joins the expeditions once its fog has been met: methane fog over
// every map of it (clear of the base), its own bugs once known and, once
// beaten, the Kraken in the last wave.
func TestExpeditionTitan(t *testing.T) {
	for _, c := range []struct {
		n             int
		titan, beacon bool
		kraken        bool
	}{{14, false, false, false}, {15, true, false, false}, {16, true, true, false}, {21, true, true, true}} {
		k := expKnow(c.n)
		if k.Titan != c.titan || k.hasTurret(Beacon) != c.beacon {
			t.Fatalf("stage %d: Titan %v, Beacon %v", c.n, k.Titan, k.hasTurret(Beacon))
		}
		seen := map[string]int{}
		for seed := int64(1); seed <= 120; seed++ {
			e := NewExpedition(seed, k)
			seen[e.World]++
			if e.World != "Titan" {
				continue
			}
			if n := len(e.Spec.Fog); n < 2 || n > 4 {
				t.Errorf("stage %d, seed %d: %d fog banks", c.n, seed, n)
			}
			var base Vec
			nb := 0.0
			for y, row := range e.Spec.Map {
				for x, ch := range row {
					if ch == 'B' {
						base, nb = base.Add(Vec{float64(x) + 0.5, float64(y) + 0.5}), nb+1
					}
				}
			}
			base = base.Mul(1 / nb)
			for _, b := range e.Spec.Fog {
				for _, p := range b.Stops {
					if p.Dist(base) < 3.4 {
						t.Errorf("stage %d, seed %d: fog rests on the base at %v", c.n, seed, p)
					}
				}
			}
			if e.Boss == Kraken {
				seen["kraken"]++
			}
			seen["theme "+e.Theme]++
			for _, d := range e.Spec.Enemies() {
				seen[d.ID]++
			}
		}
		t.Logf("after %s: %v", expCampaign()[c.n-1].ID, seen)
		if (seen["Titan"] > 0) != c.titan || (seen["kraken"] > 0) != c.kraken {
			t.Errorf("stage %d: %d Titan expeditions, %d with the Kraken", c.n, seen["Titan"], seen["kraken"])
		}
		if c.n == 21 && (seen["stalker"] == 0 || seen["bloater"] == 0 || seen["theme Ambush"] == 0) {
			t.Errorf("after 3-7 Titan's own bugs should come: %v", seen)
		}
	}
}

// Only Skyguards answer flyers: a flock that cannot grow any more does not
// turn into a wall of hit points either (v0.15.1 sent stingers at 5.9x).
func TestExpeditionFlocks(t *testing.T) {
	camp := expCampaign()
	top := map[string]float64{}
	for _, s := range camp {
		for _, wv := range s.Waves {
			for _, g := range wv.Groups {
				top[g.Enemy] = max(top[g.Enemy], g.HPMul)
			}
		}
	}
	for n := 1; n <= len(camp); n++ {
		k := expKnow(n)
		for seed := int64(1); seed <= 60; seed++ {
			for tier := TierLow; tier <= TierHigh; tier++ {
				for wi, wv := range newExpedition(seed, k, tier).Spec.Waves {
					for _, g := range wv.Groups {
						if EnemyDefs[g.Enemy].Flying && g.HPMul > 2.25*top[g.Enemy] {
							t.Errorf("after %s, seed %d, %s, wave %d: %d %s at %.2fx, the campaign's toughest are %.2fx",
								camp[n-1].ID, seed, tier, wi+1, g.Count, g.Enemy, g.HPMul, top[g.Enemy])
						}
					}
				}
			}
		}
	}
}

// Walkers come in few, tough robots as in the campaign: a thin share of the
// budget sends fewer of them, not a crowd that melts under light fire.
func TestExpeditionWalkers(t *testing.T) {
	camp := expCampaign()
	for n := 1; n <= len(camp); n++ {
		k := expKnow(n)
		var hm []float64
		for seed := int64(1); seed <= 120; seed++ {
			for _, wv := range NewExpedition(seed, k).Spec.Waves {
				for _, g := range wv.Groups {
					if g.Enemy == Walker.ID {
						hm = append(hm, g.HPMul)
					}
				}
			}
		}
		if len(hm) < 20 {
			continue
		}
		sort.Float64s(hm)
		if med := hm[len(hm)/2]; med < 1.1 {
			t.Errorf("after %s: walkers at a median %.2fx hit points, the campaign's are 1.5-2.5x", camp[n-1].ID, med)
		}
	}
}
