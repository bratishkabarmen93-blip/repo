package game

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// planUp builds the given turrets in order, then upgrades the weakest one (the
// earliest built first).
func planUp(l *Level, builds []step, top bool) bot {
	i := 0
	var order []int
	return func(w *World) {
		for {
			if i < len(builds) {
				s := builds[i]
				if s.def == nil { // upgrade the turret on that pad once
					t := w.SlotTower[s.slot]
					if t != nil && t.Busy {
						return
					}
					if t == nil || t.MaxLevel() {
						i++
						continue
					}
					if w.Upgrade(t) != nil {
						return
					}
					i++
					continue
				}
				t, err := w.Build(s.slot, s.def)
				if err != nil {
					return
				}
				if s.strong && w.CanTarget() {
					t.Mode = TargetStrong
				}
				order = append(order, s.slot)
				i++
				continue
			}
			var best *Tower
			for _, p := range order {
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

// variantOf copies a level and scales the health of one enemy type.
func variantOf(spec *LevelSpec, enemy string, k float64) *Level {
	s := *spec
	s.Waves = make([]WaveSpec, len(spec.Waves))
	for i, wv := range spec.Waves {
		gs := make([]Group, len(wv.Groups))
		copy(gs, wv.Groups)
		for j := range gs {
			if gs[j].Enemy == enemy || enemy == "all" {
				gs[j].HPMul *= k
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

// hivePlan builds eight turrets for Hive Heart, then takes the railguns and
// flamers up to their super upgrades before anything else.
func hivePlan(supers bool) []step {
	p := []step{B(2, Flamer), B(5, Blaster), S(6, Railgun), B(3, Flamer), B(0, Skyguard), B(4, Cryo), S(7, Railgun), B(1, Skyguard)}
	for i := 0; i < 3; i++ {
		p = append(p, U(6), U(7), U(2), U(3))
	}
	if supers {
		p = append(p, U(6), U(2), U(7), U(3))
	}
	return p
}

type tunePlan struct {
	name  string
	steps []step
}

var tunePlans = map[string][]tunePlan{
	"1-1": {
		{"blasters", []step{B(1, Blaster), B(3, Blaster), B(2, Blaster), B(4, Blaster)}},
	},
	"1-2": {
		{"cryoTop", []step{B(3, Blaster), B(0, Cryo), B(4, Blaster), B(5, Blaster), B(2, Blaster)}},
		{"blHair", []step{B(0, Blaster), B(3, Blaster), B(4, Blaster), B(5, Blaster), B(2, Blaster)}},
		{"cryoLow", []step{B(0, Blaster), B(3, Cryo), B(4, Blaster), B(5, Blaster), B(2, Blaster)}},
	},
	"1-3": {
		{"flame", []step{B(0, Blaster), B(4, Flamer), B(1, Flamer), B(5, Cryo), B(3, Blaster)}},
		{"noflame", []step{B(0, Blaster), B(4, Blaster), B(1, Blaster), B(5, Cryo), B(3, Blaster)}},
	},
	"1-5": {
		{"strong", []step{B(3, Blaster), S(4, Railgun), B(5, Flamer), S(6, Railgun), B(2, Cryo), B(7, Blaster), B(1, Blaster), B(0, Blaster)}},
		{"first", []step{B(3, Blaster), B(4, Railgun), B(5, Flamer), B(6, Railgun), B(2, Cryo), B(7, Blaster), B(1, Blaster), B(0, Blaster)}},
		{"strongE", []step{B(4, Blaster), S(3, Railgun), B(6, Flamer), S(5, Railgun), B(2, Cryo), B(7, Blaster), B(1, Blaster), B(0, Blaster)}},
		{"firstE", []step{B(4, Blaster), B(3, Railgun), B(6, Flamer), B(5, Railgun), B(2, Cryo), B(7, Blaster), B(1, Blaster), B(0, Blaster)}},
		{"late", []step{B(3, Blaster), B(1, Railgun), B(5, Flamer), B(7, Railgun), B(4, Cryo), B(6, Blaster), B(2, Blaster), B(0, Blaster)}},
		{"norail", []step{B(3, Blaster), B(4, Blaster), B(5, Flamer), B(6, Blaster), B(2, Cryo), B(7, Blaster), B(1, Blaster), B(0, Blaster)}},
	},
	"1-6": {
		{"sky", []step{B(7, Blaster), B(2, Skyguard), B(9, Flamer), S(1, Railgun), B(3, Skyguard), B(6, Flamer)}},
		{"nosky", []step{B(7, Blaster), B(2, Blaster), B(9, Flamer), S(1, Railgun), B(3, Blaster), B(6, Flamer)}},
		{"sky0", []step{B(2, Blaster), B(5, Skyguard), B(7, Blaster), S(8, Railgun), B(1, Cryo), B(9, Blaster), B(3, Skyguard), B(4, Flamer)}},
	},
	"1-7": {
		{"even2", []step{B(2, Flamer), B(5, Blaster), S(6, Railgun), B(3, Flamer), B(0, Skyguard), B(4, Cryo), S(7, Railgun), B(1, Skyguard)}},
		{"nosuper", []step{B(2, Flamer), B(5, Blaster), S(6, Railgun), B(3, Flamer), B(0, Skyguard), B(4, Cryo), S(7, Railgun), B(1, Skyguard)}},
		{"super", hivePlan(true)},
		{"even", []step{B(2, Flamer), B(5, Blaster), S(6, Railgun), B(3, Flamer), B(0, Skyguard), B(4, Cryo), S(7, Railgun), B(1, Skyguard)}},
	},
	"1-4": {
		{"rail", []step{B(1, Blaster), B(3, Railgun), B(4, Blaster), B(5, Cryo), B(6, Railgun)}},
		{"norail", []step{B(1, Blaster), B(3, Blaster), B(4, Blaster), B(5, Cryo), B(6, Blaster)}},
		{"railfl", []step{B(1, Blaster), B(3, Railgun), B(4, Flamer), B(5, Cryo), B(6, Railgun)}},
	},
}

// TestTune: TUNE=1-2 ENEMY=runner K=1,1.2,1.4 go test -run TestTune -v
func TestTune(t *testing.T) {
	id := os.Getenv("TUNE")
	if id == "" {
		t.Skip()
	}
	enemy := os.Getenv("ENEMY")
	ks := []float64{1}
	if s := os.Getenv("K"); s != "" {
		ks = nil
		for _, f := range strings.Split(s, ",") {
			v, _ := strconv.ParseFloat(f, 64)
			ks = append(ks, v)
		}
	}
	var tc tutorialCase
	for _, c := range tutorialCases {
		if c.spec.ID == id {
			tc = c
		}
	}
	for _, k := range ks {
		l := variantOf(tc.spec, enemy, k)
		line := fmt.Sprintf("k=%.2f", k)
		for _, p := range tunePlans[id] {
			w := NewWorld(l, 1)
			runToEnd(w, planUp(l, p.steps, false))
			line += fmt.Sprintf(" | %s %s", p.name, summary(w))
		}
		t.Log(line)
		t.Logf("        casual %s | old %s", casualStats(l, tc.casual, 24), casualStats(l, tc.old, 24))
	}
}

// TestCryoVariants: CRYO=1 go test -run TestCryoVariants -v
func TestCryoVariants(t *testing.T) {
	if os.Getenv("CRYO") == "" {
		t.Skip()
	}
	orig := make([]TowerLevel, len(Cryo.Levels))
	copy(orig, Cryo.Levels)
	defer copy(Cryo.Levels, orig)
	type mod struct {
		name string
		f    func(lv []TowerLevel)
	}
	mods := []mod{
		{"cur", func(lv []TowerLevel) {}},
		{"slow+10 r+.3", func(lv []TowerLevel) {
			for i := range lv {
				lv[i].Slow += 0.1
				lv[i].Range += 0.3
			}
		}},
		{"r+.3", func(lv []TowerLevel) {
			for i := range lv {
				lv[i].Range += 0.3
			}
		}},
		{"r+.3 slow+5", func(lv []TowerLevel) {
			for i := range lv {
				lv[i].Range += 0.3
				lv[i].Slow += 0.05
			}
		}},
		{"r+.2 slow+5", func(lv []TowerLevel) {
			for i := range lv {
				lv[i].Range += 0.2
				lv[i].Slow += 0.05
			}
		}},
	}
	k, _ := strconv.ParseFloat(os.Getenv("K"), 64)
	if k == 0 {
		k = 1.25
	}
	for _, m := range mods {
		copy(Cryo.Levels, orig)
		m.f(Cryo.Levels)
		l := variantOf(Canyon, "runner", k)
		line := m.name
		for _, p := range tunePlans["1-2"] {
			line += fmt.Sprintf(" | %s %s", p.name, planStats(l, p.steps, 8))
		}
		t.Log(line)
		t.Logf("        casual %s | old %s", casualStats(l, []*TowerDef{Blaster, Blaster, Cryo}, 24), casualStats(l, []*TowerDef{Blaster}, 24))
		kl, _ := ParseLevel(Level1)
		wins := 0
		for s := int64(1); s <= 16; s++ {
			w := NewWorld(kl, s)
			runToEnd(w, casualBot(kl, s))
			if w.Phase == PhaseWon {
				wins++
			}
		}
		w := NewWorld(kl, 1)
		runToEnd(w, planBot(planCryo))
		t.Logf("        kepler casual %d/16, planCryo %s", wins, summary(w))
	}
}

// planStats runs a build plan with several seeds.
func planStats(l *Level, steps []step, seeds int) string {
	wins, hp := 0, 0
	stars := [4]int{}
	for s := int64(1); s <= int64(seeds); s++ {
		w := NewWorld(l, s)
		runToEnd(w, planUp(l, steps, false))
		if w.Phase == PhaseWon {
			wins++
			hp += w.BaseHP
		}
		stars[w.Stars()]++
	}
	return fmt.Sprintf("%d/%d hp%3d [%d %d %d %d]", wins, seeds, hp/max(1, wins), stars[0], stars[1], stars[2], stars[3])
}
