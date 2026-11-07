package game

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// runRes is the outcome of one simulated game.
type runRes struct {
	won   bool
	hp    int
	stars int
	sum   string
	leaks map[string]int // base damage by enemy type
	waves map[int]int    // base damage by wave
}

// simulate plays one game with the bot; safe to call from several goroutines.
func simulate(l *Level, seed int64, b bot) runRes {
	w := NewWorld(l, seed)
	leaks := map[string]int{}
	waves := map[int]int{}
	for i := 0; i < 400000 && w.Active(); i++ {
		if b != nil && i%30 == 0 {
			b(w)
		}
		w.Update(1.0 / 60)
		for _, ev := range w.Events {
			if ev.Kind == EvLeak {
				leaks[ev.Enemy.ID] += ev.Value
				waves[w.Wave] += ev.Value
			}
		}
		w.Events = w.Events[:0]
	}
	return runRes{w.Phase == PhaseWon, w.BaseHP, w.Stars(), summary(w), leaks, waves}
}

// parallel calls f(0..n-1) on every CPU.
func parallel(n int, f func(i int)) {
	var wg sync.WaitGroup
	ch := make(chan int)
	for k := 0; k < runtime.NumCPU(); k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				f(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()
}

// withDamage applies DMGK (multiplier for every non-boss enemy), BOSSD (boss
// damage) and DMG="crawler=8,runner=5" overrides; the returned func restores.
func withDamage() func() {
	saved := map[*EnemyDef]int{}
	for _, d := range EnemyDefs {
		saved[d] = d.Damage
	}
	if k := floats(os.Getenv("DMGK"), 1)[0]; k != 1 {
		for _, d := range EnemyDefs {
			if !d.Boss {
				d.Damage = int(float64(d.Damage)*k + 0.5)
			}
		}
	}
	if s := os.Getenv("BOSSD"); s != "" {
		v, _ := strconv.Atoi(s)
		for _, d := range EnemyDefs {
			if d.Boss {
				d.Damage = v
			}
		}
	}
	for _, kv := range strings.Split(os.Getenv("DMG"), ",") {
		if id, v, ok := strings.Cut(kv, "="); ok {
			n, _ := strconv.Atoi(v)
			EnemyDefs[id].Damage = n
		}
	}
	return func() {
		for d, v := range saved {
			d.Damage = v
		}
	}
}

// casualLine runs n casual players and sums them up.
func casualLine(l *Level, types []*TowerDef, n int) (string, int) {
	rs := make([]runRes, n)
	parallel(n, func(i int) { rs[i] = simulate(l, int64(i+1), casualFor(l, int64(i+1), types)) })
	wins, hp := 0, 0
	st := [4]int{}
	leaks := map[string]int{}
	waves := map[int]int{}
	lostAt := map[int]int{}
	for _, r := range rs {
		if r.won {
			wins++
			hp += r.hp
		} else {
			lostAt[waveOf(r.sum)]++
		}
		st[r.stars]++
		for k, v := range r.leaks {
			leaks[k] += v
		}
		for k, v := range r.waves {
			waves[k] += v
		}
	}
	var ids []string
	for k := range leaks {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(a, b int) bool { return leaks[ids[a]] > leaks[ids[b]] })
	lk := ""
	for i, id := range ids {
		if i < 4 {
			lk += fmt.Sprintf(" %s %.1f", id, float64(leaks[id])/float64(n))
		}
	}
	if os.Getenv("WAVES") != "" {
		lk += " | by wave"
		for i := 1; i <= len(l.Spec.Waves); i++ {
			lk += fmt.Sprintf(" %.0f", float64(waves[i])/float64(n))
		}
		lk += fmt.Sprintf(" | lost at %v", lostAt)
	}
	return fmt.Sprintf("%2d/%d %3d%% hp %3d | 3*%2d 2*%2d 1*%2d 0*%2d | leaks/run%s", wins, n, wins*100/n, hp/max(1, wins), st[3], st[2], st[1], st[0], lk), wins
}

// TestReport: REPORT=1 [SEEDS=24 LV=1-1,2-3 DMGK=2 BOSSD=50 DMG=crawler=8]
// — every level: casual players, the tuned plans, lazy players and no defence.
func TestReport(t *testing.T) {
	if os.Getenv("REPORT") == "" {
		t.Skip()
	}
	defer withDamage()()
	n := int(floats(os.Getenv("SEEDS"), 24)[0])
	only := os.Getenv("LV")
	chapWins := map[byte][2]int{}
	cases := append(slices.Clone(tutorialCases), tutorialCase{spec: Skirmish, casual: []*TowerDef{bl, bl, cr, fl, rl, sk}})
	for _, c := range cases {
		id := c.spec.ID
		if only != "" && !strings.Contains(only, id) || only == "" && c.spec == Skirmish {
			continue
		}
		l, err := ParseLevel(tweak(c.spec))
		if err != nil {
			t.Fatal(err)
		}
		line, wins := casualLine(l, c.casual, n)
		cw := chapWins[id[0]]
		chapWins[id[0]] = [2]int{cw[0] + wins, cw[1] + n}
		t.Logf("%s %-14s casual %s", id, c.spec.Name, line)
		if c.old != nil && os.Getenv("OLD") != "" {
			ol, _ := casualLine(l, c.old, n)
			t.Logf("   %-14s old    %s", "", ol)
		}
		plans := tunePlans[id]
		var bots []bot
		var names []string
		for _, p := range plans {
			lv := l
			if p.name == "nosuper" {
				lv = noSuper(tweak(c.spec))
			}
			bots = append(bots, planUp(lv, p.steps, false))
			names = append(names, p.name)
		}
		pads := padOrder(l, 3, 0, nil)
		bots = append(bots, goodBot(l, []*TowerDef{Blaster, Blaster}), planBot([]step{B(pads[0], Blaster), B(pads[1], Blaster), B(pads[2], Blaster)}), nil)
		names = append(names, "2bl+up", "3bl", "none")
		rs := make([]runRes, len(bots))
		parallel(len(bots), func(i int) {
			lv := l
			if names[i] == "nosuper" {
				lv = noSuper(tweak(c.spec))
			}
			rs[i] = simulate(lv, 1, bots[i])
		})
		var parts []string
		for i, r := range rs {
			s := fmt.Sprintf("%s %s w%d %d%%", names[i], map[bool]string{true: "won", false: "LOST"}[r.won], waveOf(r.sum), r.hp)
			if r.won {
				s += fmt.Sprintf(" %d*", r.stars)
			}
			if os.Getenv("PL") != "" && len(r.leaks) > 0 {
				s += fmt.Sprintf(" %v%v", r.leaks, r.waves)
			}
			parts = append(parts, s)
		}
		t.Logf("   plans: %s", strings.Join(parts, " | "))
	}
	for _, ch := range []byte{'1', '2'} {
		if cw := chapWins[ch]; cw[1] > 0 {
			t.Logf("chapter %c casual wins %d/%d = %d%%", ch, cw[0], cw[1], cw[0]*100/cw[1])
		}
	}
}

// waveOf reads the wave number back from a summary line.
func waveOf(sum string) int {
	i := strings.Index(sum, " w")
	if i < 0 {
		return 0
	}
	v := 0
	for _, c := range sum[i+2:] {
		if c < '0' || c > '9' {
			break
		}
		v = v*10 + int(c-'0')
	}
	return v
}

// tweak copies a level spec with WAVEK="4=1.2,5=1.1" (health of every group
// in a wave) and ENEMY=runner K=0.9 (health of one enemy type) applied.
func tweak(spec *LevelSpec) *LevelSpec {
	s := *spec
	s.Waves = make([]WaveSpec, len(spec.Waves))
	wk := map[int]float64{}
	for _, kv := range strings.Split(os.Getenv("WAVEK"), ",") {
		if a, b, ok := strings.Cut(kv, "="); ok {
			i, _ := strconv.Atoi(a)
			wk[i], _ = strconv.ParseFloat(b, 64)
		}
	}
	s.StartCredits = int(float64(s.StartCredits) * floats(os.Getenv("CREDK"), 1)[0])
	s.WaveBonus = int(float64(s.WaveBonus) * floats(os.Getenv("BONUSK"), 1)[0])
	enemy := os.Getenv("ENEMY")
	k := floats(os.Getenv("K"), 1)[0]
	for i, wv := range spec.Waves {
		gs := make([]Group, len(wv.Groups))
		copy(gs, wv.Groups)
		for j := range gs {
			if m, ok := wk[i+1]; ok {
				gs[j].HPMul *= m
			}
			if gs[j].Enemy == enemy || enemy == "all" {
				gs[j].HPMul *= k
			}
		}
		s.Waves[i] = WaveSpec{Groups: gs}
	}
	return &s
}
