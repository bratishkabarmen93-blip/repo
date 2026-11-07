package game

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestWyrmDiag: WY=1 [P="plan;plan"] [SEEDS=24] [K=1] — where the Ice Wyrm
// dies (path distance) or whether it leaks, for casual players and plans.
func TestWyrmDiag(t *testing.T) {
	if os.Getenv("WY") == "" {
		t.Skip()
	}
	spec := WyrmLake
	l := variantOf(spec, os.Getenv("ENEMY"), floats(os.Getenv("K"), 1)[0])
	var types []*TowerDef
	for _, c := range tutorialCases {
		if c.spec == spec {
			types = c.casual
		}
	}
	run := func(w *World, b bot) string {
		res := "alive"
		hp9 := -1
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			if w.Wave == w.TotalWaves() && hp9 < 0 {
				hp9 = w.BaseHP
			}
			for _, ev := range w.Events {
				if ev.Enemy == Wyrm {
					switch ev.Kind {
					case EvKill:
						res = fmt.Sprintf("dies at %4.1f", ev.Dist)
					case EvLeak:
						res = "LEAKS     "
					}
				}
			}
			w.Events = w.Events[:0]
		}
		return fmt.Sprintf("%s | hp before w9 %3d | %s", res, hp9, summary(w))
	}
	n := int(floats(os.Getenv("SEEDS"), 24)[0])
	if ps := os.Getenv("P"); ps != "" {
		for _, p := range strings.Split(ps, ";") {
			for s := int64(1); s <= int64(n); s++ {
				t.Logf("%-28s seed %2d %s", p, s, run(NewWorld(l, s), planUp(l, parsePlan(p), false)))
			}
		}
		return
	}
	for _, p := range tunePlans["2-7"] {
		t.Logf("%-8s %s", p.name, run(NewWorld(l, 1), planUp(l, p.steps, false)))
	}
	for s := int64(1); s <= int64(n); s++ {
		t.Logf("casual seed %2d %s", s, run(NewWorld(l, s), casualFor(l, s, types)))
	}
}
