package game

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
)

// TestHP50: HP50=1 [SEEDS=24] — what a base of 50 HP instead of 100 would do
// to every level: casual players (wins and stars) and the tuned best plan.
func TestHP50(t *testing.T) {
	if os.Getenv("HP50") == "" {
		t.Skip()
	}
	n := int(floats(os.Getenv("SEEDS"), 24)[0])
	withHP := func(spec *LevelSpec, hp int) *Level {
		s := *spec
		s.BaseHP = hp
		l, err := ParseLevel(&s)
		if err != nil {
			panic(err)
		}
		return l
	}
	type res struct {
		won   bool
		hp    int
		stars int
	}
	run := func(l *Level, seed int64, b bot) res {
		w := NewWorld(l, seed)
		for i := 0; i < 400000 && w.Active(); i++ {
			if i%30 == 0 {
				b(w)
			}
			w.Update(1.0 / 60)
			w.Events = w.Events[:0]
		}
		return res{w.Phase == PhaseWon, w.BaseHP, w.Stars()}
	}
	stat := func(rs []res) string {
		wins := 0
		st := [4]int{}
		for _, r := range rs {
			if r.won {
				wins++
			}
			st[r.stars]++
		}
		return fmt.Sprintf("%2d/%d won, 3*%2d 2*%2d 1*%2d", wins, len(rs), st[3], st[2], st[1])
	}
	for _, c := range tutorialCases {
		var line [2]string
		var best [2]string
		for hi, hp := range []int{100, 50} {
			l := withHP(c.spec, hp)
			rs := make([]res, n)
			var wg sync.WaitGroup
			ch := make(chan int)
			for k := 0; k < runtime.NumCPU(); k++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for s := range ch {
						rs[s] = run(l, int64(s+1), casualFor(l, int64(s+1), c.casual))
					}
				}()
			}
			for s := 0; s < n; s++ {
				ch <- s
			}
			close(ch)
			wg.Wait()
			line[hi] = stat(rs)
			var b bot
			if plans := tunePlans[c.spec.ID]; len(plans) > 0 {
				b = planUp(l, plans[0].steps, false)
			} else {
				b = goodBot(l, c.good)
			}
			r := run(l, 1, b)
			best[hi] = fmt.Sprintf("%3d hp %d*", r.hp, r.stars)
			if !r.won {
				best[hi] = "LOST    "
			}
		}
		t.Logf("%s %-13s | casual 100: %s | casual 50: %s | best 100: %s | best 50: %s", c.spec.ID, c.spec.Name, line[0], line[1], best[0], best[1])
	}
}
