package game

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// TestPads2: PADS=2-1 go test -run TestPads2 -v
func TestPads2(t *testing.T) {
	id := os.Getenv("PADS")
	if id == "" {
		t.Skip()
	}
	l, err := ParseLevel(LevelByID(id))
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range l.Slots {
		line := ""
		for ri, rt := range l.Routes {
			line += fmt.Sprintf(" %c:%.1f", 'A'+ri, rt.Path.Coverage(s.Center(), 3))
			if len(rt.Ice) > 0 {
				// The road a diver can be hit on (range 2.5 and 4).
				for _, r := range []float64{2.5, 4} {
					solid := 0.0
					for d := 0.0; d < rt.Path.Len; d += 0.05 {
						if pos, _ := rt.Path.At(d); !rt.UnderIce(d) && pos.Dist2(s.Center()) <= r*r {
							solid += 0.05
						}
					}
					line += fmt.Sprintf(" solid%g:%.1f", r, solid)
				}
			}
		}
		t.Logf("pad %d at (%d,%d) cov3 %.1f%s", i, s.X, s.Y, l.Coverage(s.Center(), 3), line)
	}
	for ri, rt := range l.Routes {
		t.Logf("route %c: %d cells, join %d, length %.1f", 'A'+ri, len(rt.Cells), rt.Join, rt.Path.Len)
	}
}

// TestLazy2: LAZY=2-1 go test -run TestLazy2 -v
func TestLazy2(t *testing.T) {
	id := os.Getenv("LAZY")
	if id == "" {
		t.Skip()
	}
	k := floats(os.Getenv("K"), 1)[0]
	l := variantOf(LevelByID(id), os.Getenv("ENEMY"), k)
	for _, comp := range [][]*TowerDef{{Blaster, Blaster}, {Blaster, Blaster, Blaster}, {Flamer, Blaster}, {Blaster, Railgun}, {Blaster, Blaster, Skyguard}} {
		w := NewWorld(l, 1)
		log := runToEnd(w, goodBot(l, comp))
		hp := ""
		for _, x := range log {
			hp += fmt.Sprintf("%d ", x.hp)
		}
		t.Logf("%v %s | hp by wave %s| leaks %v", len(comp), summary(w), hp, leakLog)
	}
}

// scaled copies a level, scaling the health of one enemy type by kb and of
// all the others by ko.
func scaled(spec *LevelSpec, enemy string, kb, ko float64) *Level {
	s := *spec
	s.Waves = make([]WaveSpec, len(spec.Waves))
	for i, wv := range spec.Waves {
		gs := append([]Group(nil), wv.Groups...)
		for j := range gs {
			if gs[j].Enemy == enemy {
				gs[j].HPMul *= kb
			} else {
				gs[j].HPMul *= ko
			}
		}
		s.Waves[i] = WaveSpec{Groups: gs}
	}
	l, err := ParseLevel(&s)
	if err != nil {
		panic(err)
	}
	return l
}

// TestScan2: SCAN=2-2 ENEMY=beetle KB=0.85,1 KO=1,1.1 go test -run TestScan2 -v
func TestScan2(t *testing.T) {
	id := os.Getenv("SCAN")
	if id == "" {
		t.Skip()
	}
	spec := LevelByID(id)
	var tc tutorialCase
	for _, c := range tutorialCases {
		if c.spec == spec {
			tc = c
		}
	}
	for _, kb := range floats(os.Getenv("KB"), 1) {
		for _, ko := range floats(os.Getenv("KO"), 1) {
			l := scaled(spec, os.Getenv("ENEMY"), kb, ko)
			line := ""
			for _, p := range tunePlans[id][:min(2, len(tunePlans[id]))] {
				line += " | " + p.name + " " + planStats(l, p.steps, 3)
			}
			w := NewWorld(l, 1)
			runToEnd(w, goodBot(l, []*TowerDef{Blaster, Blaster}))
			old := ""
			if tc.old != nil {
				old = " | old " + casualStats(l, tc.old, 24)
			}
			t.Logf("kb=%.2f ko=%.2f%s | 2bl %s | casual %s%s", kb, ko, line, summary(w), casualStats(l, tc.casual, 24), old)
		}
	}
}

// TestPlans2: PL=2-3 [ENEMY=walker K=1.2] go test -run TestPlans2 -v — every
// tune plan of a level on one seed: the result, base HP after each wave and
// the leaks per burrow and enemy type.
func TestPlans2(t *testing.T) {
	id := os.Getenv("PL")
	if id == "" {
		t.Skip()
	}
	l := variantOf(LevelByID(id), os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	for _, p := range tunePlans[id] {
		w := NewWorld(l, 1)
		b := planUp(l, p.steps, false)
		route := map[int]int{}
		leaks := map[string]int{}
		var hp []int
		wave := 0
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			for _, ev := range w.Events {
				switch ev.Kind {
				case EvSpawn:
					route[ev.ID] = ev.Value
				case EvLeak:
					leaks[fmt.Sprintf("%c:%s", 'A'+route[ev.ID], ev.Enemy.ID)] += ev.Value
				}
			}
			w.Events = w.Events[:0]
			if w.Wave != wave {
				wave = w.Wave
				hp = append(hp, w.BaseHP)
			}
		}
		t.Logf("%-8s %s | hp %v | leaks %v", p.name, summary(w), hp, leaks)
	}
}

// TestCasual2: CAS=2-3 [SEEDS=16] — casual runs with leaks per burrow and the
// pads of the turrets.
func TestCasual2(t *testing.T) {
	id := os.Getenv("CAS")
	if id == "" {
		t.Skip()
	}
	spec := LevelByID(id)
	l := variantOf(spec, os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	var types []*TowerDef
	for _, c := range tutorialCases {
		if c.spec == spec {
			types = c.casual
		}
	}
	n := int(floats(os.Getenv("SEEDS"), 8)[0])
	for s := int64(1); s <= int64(n); s++ {
		w := NewWorld(l, s)
		b := casualFor(l, s, types)
		route := map[int]int{}
		leaks := map[string]int{}
		var hp []int
		wave := 0
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			for _, ev := range w.Events {
				switch ev.Kind {
				case EvSpawn:
					route[ev.ID] = ev.Value
				case EvLeak:
					leaks[fmt.Sprintf("%c%d:%s", 'A'+route[ev.ID], w.Wave, ev.Enemy.ID)] += ev.Value
				}
			}
			w.Events = w.Events[:0]
			if w.Wave != wave {
				wave = w.Wave
				hp = append(hp, w.BaseHP)
			}
		}
		pads := ""
		for _, tw := range w.Towers {
			side := "A"
			if l.Routes[1].Path.Coverage(l.Slots[tw.Slot].Center(), 3) > l.Routes[0].Path.Coverage(l.Slots[tw.Slot].Center(), 3) {
				side = "B"
			}
			pads += fmt.Sprintf("%s%c%d ", side, tw.Def.ID[0], tw.Level+1)
		}
		t.Logf("seed %2d %s | hp %v | %s| leaks %v", s, summary(w), hp, pads, leaks)
	}
}

// TestSearch2: SRCH=2-4 [N=1000 SEED=1] — random build orders (pads and
// turret types, then round-robin upgrades) on one seed: how many win, and the
// best ones.
func TestSearch2(t *testing.T) {
	id := os.Getenv("SRCH")
	if id == "" {
		t.Skip()
	}
	l := variantOf(LevelByID(id), os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	n := int(floats(os.Getenv("N"), 1000)[0])
	seed := int64(floats(os.Getenv("SEED"), 1)[0])
	types := []*TowerDef{Blaster, Cryo, Flamer, Railgun, Skyguard}
	r := rand.New(rand.NewSource(7))
	type res struct {
		plan     []step
		hp, wave int
		won      bool
	}
	plans := make([][]step, n)
	for i := range plans {
		k := 6 + r.Intn(4)
		pads := r.Perm(len(l.Slots))[:k]
		var p []step
		sky := false
		for j, pad := range pads {
			d := types[r.Intn(len(types))]
			if d == Skyguard {
				if sky && r.Intn(2) == 0 {
					d = Blaster
				}
				sky = true
			}
			if j == k-1 && !sky {
				d = Skyguard
			}
			if d == Railgun && r.Intn(2) == 0 {
				p = append(p, S(pad, d))
			} else {
				p = append(p, B(pad, d))
			}
		}
		plans[i] = p
	}
	out := make([]res, n)
	var wg sync.WaitGroup
	ch := make(chan int)
	for c := 0; c < runtime.NumCPU(); c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				w := NewWorld(l, seed)
				b := planUp(l, plans[i], false)
				for j := 0; j < 400000 && w.Active(); j++ {
					if j%30 == 0 {
						b(w)
					}
					w.Update(1.0 / 60)
					w.Events = w.Events[:0]
				}
				out[i] = res{plans[i], w.BaseHP, w.Wave, w.Phase == PhaseWon}
			}
		}()
	}
	for i := 0; i < n; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()
	sort.Slice(out, func(a, b int) bool {
		if out[a].won != out[b].won {
			return out[a].won
		}
		if out[a].hp != out[b].hp {
			return out[a].hp > out[b].hp
		}
		return out[a].wave > out[b].wave
	})
	wins, three := 0, 0
	waves := map[int]int{}
	for _, o := range out {
		if o.won {
			wins++
			if o.hp >= 90 {
				three++
			}
		} else {
			waves[o.wave]++
		}
	}
	t.Logf("%d plans: %d won (%d 3*), losses by wave %v; hp p10/p50/p90 of winners: %d/%d/%d", n, wins, three, waves,
		out[wins*9/10].hp, out[wins/2].hp, out[wins/10].hp)
	for _, o := range out[:min(12, n)] {
		s := ""
		for _, st := range o.plan {
			m := ""
			if st.strong {
				m = "!"
			}
			s += fmt.Sprintf(" %d%c%s", st.slot, st.def.ID[0], m)
		}
		t.Logf("hp %3d w%d |%s", o.hp, o.wave, s)
	}
}

// parsePlan reads a build order like "5r! 4c 2f": pad, turret initial
// (b c f r s) and "!" to aim at the strongest enemy.
func parsePlan(s string) []step {
	var p []step
	for _, f := range strings.Fields(s) {
		pad := 0
		i := 0
		for i < len(f) && f[i] >= '0' && f[i] <= '9' {
			pad = pad*10 + int(f[i]-'0')
			i++
		}
		var d *TowerDef
		for _, td := range TowerOrder {
			if td.ID[0] == f[i] {
				d = td
			}
		}
		if strings.HasSuffix(f, "!") {
			p = append(p, S(pad, d))
		} else {
			p = append(p, B(pad, d))
		}
	}
	return p
}

// TestEval2: EV=2-4 P="5r! 4c 2f;1f 5c" [SEEDS=5] — custom build orders over
// several seeds.
func TestEval2(t *testing.T) {
	id := os.Getenv("EV")
	if id == "" {
		t.Skip()
	}
	l := variantOf(LevelByID(id), os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	n := int(floats(os.Getenv("SEEDS"), 5)[0])
	for _, ps := range strings.Split(os.Getenv("P"), ";") {
		steps := parsePlan(ps)
		line := ""
		for s := int64(1); s <= int64(n); s++ {
			w := NewWorld(l, s)
			b := planUp(l, steps, false)
			for j := 0; j < 400000 && w.Active(); j++ {
				if j%30 == 0 {
					b(w)
				}
				w.Update(1.0 / 60)
				w.Events = w.Events[:0]
			}
			if w.Phase == PhaseWon {
				line += fmt.Sprintf(" %3d", w.BaseHP)
			} else {
				line += fmt.Sprintf("  L%d", w.Wave)
			}
		}
		t.Logf("%-32s |%s", ps, line)
	}
}

// TestClimb2: CLIMB=2-6 P="plan;plan" [ITER=40 SEEDS=3] — hill-climbs build
// orders from the given ones (or random ones) and prints the best found.
func TestClimb2(t *testing.T) {
	id := os.Getenv("CLIMB")
	if id == "" {
		t.Skip()
	}
	l := variantOf(LevelByID(id), os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	iters := int(floats(os.Getenv("ITER"), 40)[0])
	seeds := int(floats(os.Getenv("SEEDS"), 3)[0])
	types := []*TowerDef{Blaster, Cryo, Flamer, Railgun, Skyguard}
	r := rand.New(rand.NewSource(int64(floats(os.Getenv("SEED"), 11)[0])))
	score := func(p []step) float64 {
		sum := 0.0
		for s := int64(1); s <= int64(seeds); s++ {
			w := NewWorld(l, s)
			b := planUp(l, p, false)
			for j := 0; j < 400000 && w.Active(); j++ {
				if j%30 == 0 {
					b(w)
				}
				w.Update(1.0 / 60)
				w.Events = w.Events[:0]
			}
			if w.Phase == PhaseWon {
				sum += float64(w.BaseHP)
			} else {
				sum += -200 + 20*float64(w.Wave)
			}
		}
		return sum / float64(seeds)
	}
	str := func(p []step) string {
		out := ""
		for _, s := range p {
			out += fmt.Sprintf(" %d%c", s.slot, s.def.ID[0])
			if s.strong {
				out += "!"
			}
		}
		return strings.TrimSpace(out)
	}
	mutate := func(p []step) []step {
		q := append([]step(nil), p...)
		used := map[int]bool{}
		for _, s := range q {
			used[s.slot] = true
		}
		var free []int
		for i := range l.Slots {
			if !used[i] {
				free = append(free, i)
			}
		}
		switch k := r.Intn(6); {
		case k == 0 && len(q) > 0: // another turret type
			i := r.Intn(len(q))
			q[i] = B(q[i].slot, types[r.Intn(len(types))])
		case k == 1 && len(q) > 0 && len(free) > 0: // another pad
			i := r.Intn(len(q))
			s := q[i]
			s.slot = free[r.Intn(len(free))]
			q[i] = s
		case k == 2 && len(q) > 1: // swap the order
			i, j := r.Intn(len(q)), r.Intn(len(q))
			q[i], q[j] = q[j], q[i]
		case k == 3 && len(free) > 0 && len(q) < 10: // one more turret
			i := r.Intn(len(q) + 1)
			s := B(free[r.Intn(len(free))], types[r.Intn(len(types))])
			q = append(q[:i], append([]step{s}, q[i:]...)...)
		case k == 4 && len(q) > 4: // one less
			i := r.Intn(len(q))
			q = append(q[:i], q[i+1:]...)
		default: // railgun priority
			for _, i := range r.Perm(len(q)) {
				if q[i].def == Railgun {
					q[i].strong = !q[i].strong
					break
				}
			}
		}
		return q
	}
	starts := strings.Split(os.Getenv("P"), ";")
	for _, ps := range starts {
		cur := parsePlan(ps)
		best := score(cur)
		for it := 0; it < iters; it++ {
			n := max(4, runtime.NumCPU())
			cands := make([][]step, n)
			sc := make([]float64, n)
			for i := range cands {
				cands[i] = mutate(cur)
				if r.Intn(3) == 0 {
					cands[i] = mutate(cands[i])
				}
			}
			var wg sync.WaitGroup
			for i := range cands {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					sc[i] = score(cands[i])
				}(i)
			}
			wg.Wait()
			bi := 0
			for i := range sc {
				if sc[i] > sc[bi] {
					bi = i
				}
			}
			if sc[bi] >= best {
				cur, best = cands[bi], sc[bi]
			}
		}
		t.Logf("%6.1f | %s   (from %s)", best, str(cur), ps)
	}
}
