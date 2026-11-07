package game

import (
	"fmt"
	"os"
	"testing"
)

// TestRailDiag: DIAG=1-5 go test -run TestRailDiag -v
func TestRailDiag(t *testing.T) {
	id := os.Getenv("DIAG")
	if id == "" {
		t.Skip()
	}
	spec := mixOf(LevelByID(id), os.Getenv("HP"), os.Getenv("CNT"))
	l, _ := ParseLevel(spec)
	for _, p := range tunePlans[id] {
		w := NewWorld(l, 1)
		b := planUp(l, p.steps, false)
		shots, wasted, walkerIn := 0, 0, 0
		perWave := map[int][3]int{}
		leaks := map[int]string{}
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			for _, ev := range w.Events {
				if ev.Kind == EvLeak {
					leaks[w.Wave] += fmt.Sprintf("%c%d ", ev.Enemy.ID[0], ev.Value)
				}
				if ev.Kind != EvRail {
					continue
				}
				var tw *Tower
				for _, x := range w.Towers {
					if x.ID == ev.ID {
						tw = x
					}
				}
				r := tw.Stats().Range
				in := false
				for _, e := range w.Enemies {
					if e.alive && e.Def == Walker && e.Pos.Dist2(tw.Pos) <= r*r {
						in = true
					}
				}
				shots++
				pw := perWave[w.Wave]
				pw[0]++
				if in {
					walkerIn++
					pw[1]++
					if ev.Enemy != Walker {
						wasted++
						pw[2]++
					}
				}
				perWave[w.Wave] = pw
			}
			w.Events = w.Events[:0]
		}
		t.Logf("%-7s %s shots %d walkerInRange %d wasted %d perWave %v", p.name, summary(w), shots, walkerIn, wasted, perWave)
		t.Logf("        leaks %v", leaks)
	}
}

// TestCasualDiag: CDIAG=1-5 go test -run TestCasualDiag -v
func TestCasualDiag(t *testing.T) {
	id := os.Getenv("CDIAG")
	if id == "" {
		t.Skip()
	}
	spec := mixOf(LevelByID(id), os.Getenv("HP"), os.Getenv("CNT"))
	l, _ := ParseLevel(spec)
	var types []*TowerDef
	for _, c := range tutorialCases {
		if c.spec.ID == id {
			types = c.casual
		}
	}
	for s := int64(1); s <= 16; s++ {
		w := NewWorld(l, s)
		log := runToEnd(w, casualFor(l, s, types))
		hp := ""
		for _, x := range log {
			hp += fmt.Sprintf("%d ", x.hp)
		}
		t.Logf("seed %2d %s | hp by wave %s| leaks %v", s, summary(w), hp, leakLog)
	}
}

// TestEcon: ECON=1-7 go test -run TestEcon -v
func TestEcon(t *testing.T) {
	id := os.Getenv("ECON")
	if id == "" {
		t.Skip()
	}
	spec := mixOf(LevelByID(id), os.Getenv("HP"), os.Getenv("CNT"))
	l, _ := ParseLevel(spec)
	for _, p := range tunePlans[id] {
		lv := l
		if p.name == "nosuper" {
			lv = noSuper(spec)
		}
		w := NewWorld(lv, 1)
		b := planUp(lv, p.steps, false)
		out := ""
		wave := -1
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			if w.Wave != wave {
				wave = w.Wave
				lv := ""
				for _, t := range w.Towers {
					lv += fmt.Sprintf("%c%d", t.Def.ID[0], t.Level+1)
				}
				out += fmt.Sprintf("\n   w%d hp%d cr%d %s leaks:", wave, w.BaseHP, w.Credits, lv)
			}
			for _, ev := range w.Events {
				if ev.Kind == EvLeak {
					out += fmt.Sprintf(" %c%d", ev.Enemy.ID[0], ev.Value)
				}
			}
			w.Events = w.Events[:0]
		}
		t.Logf("%s %s%s", p.name, summary(w), out)
	}
}
