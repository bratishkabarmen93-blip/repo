package game

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// variant copies Level1 and lets a function tweak its waves.
func variant(mod func(ws []WaveSpec)) *Level {
	spec := *Level1
	spec.Waves = make([]WaveSpec, len(Level1.Waves))
	for i, wv := range Level1.Waves {
		gs := make([]Group, len(wv.Groups))
		copy(gs, wv.Groups)
		spec.Waves[i] = WaveSpec{Groups: gs}
	}
	if mod != nil {
		mod(spec.Waves)
	}
	l, err := ParseLevel(&spec)
	if err != nil {
		panic(err)
	}
	return l
}

// scaleHP multiplies the health of every non-drone, non-queen group of wave i by k[i].
func scaleHP(k ...float64) func(ws []WaveSpec) {
	return func(ws []WaveSpec) {
		for i := range ws {
			for j := range ws[i].Groups {
				g := &ws[i].Groups[j]
				if g.Enemy != "drone" && g.Enemy != "queen" && i < len(k) {
					g.HPMul *= k[i]
				}
			}
		}
	}
}

func evalLevel(t *testing.T, name string, l *Level) {
	res := func(w *World) string {
		lv := ""
		for _, t := range w.Towers {
			lv += fmt.Sprintf("%c%d", t.Def.ID[0], t.Level+1)
		}
		return fmt.Sprintf("%d/%d:%3d cr%d %s", w.Phase, w.Wave, w.BaseHP, w.Credits, lv)
	}
	leaks := ""
	runPlan := func(p []step) *World {
		w := NewWorld(l, 1)
		runToEnd(w, planBot(p))
		leaks += fmt.Sprint(leakLog) + " "
		return w
	}
	mixed, bl, bla := runPlan(planMixed), runPlan(planBlasters), runPlan(planBlastersAA)
	rails, cryo := runPlan(planRails), runPlan(planCryo)
	lw := NewWorld(l, 1)
	runToEnd(lw, lazyBot(l))
	wins, hpSum := 0, 0
	var cs []string
	for s := int64(1); s <= 16; s++ {
		w := NewWorld(l, s)
		runToEnd(w, casualBot(l, s))
		if w.Phase == PhaseWon {
			wins++
			hpSum += w.BaseHP
		}
		cs = append(cs, fmt.Sprintf("%d:%d", w.Wave, w.BaseHP))
	}
	t.Logf("%-8s rails %s | cryo %s", name, res(rails), res(cryo))
	t.Logf("%-8s mixed %s | blasters %s | bl+AA %s | lazy wave %d | casual %2d/16 avg %2d [%s]",
		name, res(mixed), res(bl), res(bla), lw.Wave, wins, hpSum/max(1, wins), strings.Join(cs, " "))
	t.Logf("         leaks mixed/blasters/bl+AA/rails/cryo: %s", leaks)
}

// TestScan compares wave variants: SCAN=1 go test -run TestScan -v
func TestScan(t *testing.T) {
	if os.Getenv("SCAN") == "" {
		t.Skip()
	}
	vs := map[string]func(ws []WaveSpec){
		"cur":  nil,
		"h108": scaleHP(1, 1, 1, 1.08, 1.08, 1.08, 1.08, 1.08),
		"h115": scaleHP(1, 1, 1, 1.15, 1.15, 1.15, 1.15, 1.15),
		"h125": scaleHP(1, 1, 1.1, 1.25, 1.25, 1.25, 1.25, 1.25),
		"h135": scaleHP(1, 1.05, 1.15, 1.35, 1.35, 1.35, 1.35, 1.35),
	}
	names := []string{"cur", "h115", "h125", "h135"}
	if s := os.Getenv("VARIANTS"); s != "" {
		names = strings.Split(s, ",")
	}
	for _, n := range names {
		evalLevel(t, n, variant(vs[n]))
	}
}
