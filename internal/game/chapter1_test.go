package game

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

var sk = Skyguard

// noSuper copies a level with super upgrades locked.
func noSuper(spec *LevelSpec) *Level {
	s := *spec
	s.NoSuper = true
	l, err := ParseLevel(&s)
	if err != nil {
		panic(err)
	}
	return l
}

// TestChapterEnd: CH=1 go test -run TestChapterEnd -v
func TestChapterEnd(t *testing.T) {
	only := os.Getenv("CH")
	if only == "" {
		t.Skip()
	}
	for _, id := range []string{"1-5", "1-6", "1-7"} {
		if only != "1" && only != id {
			continue
		}
		spec := LevelByID(id)
		l, _ := ParseLevel(spec)
		for _, p := range tunePlans[id] {
			lv := l
			if p.name == "nosuper" {
				lv = noSuper(spec)
			}
			w := NewWorld(lv, 1)
			runToEnd(w, planUp(lv, p.steps, false))
			t.Logf("%s %-9s %s | %s leaks %v rail %v", id, p.name, planStats(lv, p.steps, 6), summary(w), leakLog, railLog)
		}
		for _, c := range tutorialCases {
			if c.spec == spec {
				old := ""
				if c.old != nil {
					old = casualStats(l, c.old, 24)
				}
				t.Logf("%s casual %s | old %s", id, casualStats(l, c.casual, 24), old)
			}
		}
		w := NewWorld(l, 1)
		runToEnd(w, nil)
		t.Logf("%s none %s", id, summary(w))
	}
}

// TestChapterTune: CT=1-7 ENEMY=all K=1,1.5 QK=1,2 go test -run TestChapterTune -v
func TestChapterTune(t *testing.T) {
	id := os.Getenv("CT")
	if id == "" {
		t.Skip()
	}
	spec := LevelByID(id)
	ks := floats(os.Getenv("K"), 1)
	qks := floats(os.Getenv("QK"), 1)
	for _, k := range ks {
		for _, qk := range qks {
			l := variantOf(spec, os.Getenv("ENEMY"), k)
			ls := *l.Spec
			ls.Waves = append([]WaveSpec(nil), l.Spec.Waves...)
			for i, wv := range ls.Waves {
				gs := append([]Group(nil), wv.Groups...)
				for j := range gs {
					if gs[j].Enemy == "queen" {
						gs[j].HPMul *= qk
					}
				}
				ls.Waves[i] = WaveSpec{Groups: gs}
			}
			l, _ = ParseLevel(&ls)
			line := ""
			for _, p := range tunePlans[id] {
				lv := l
				if p.name == "nosuper" {
					lv = noSuper(&ls)
				}
				line += " | " + p.name + " " + planStats(lv, p.steps, 4)
			}
			cs := ""
			for _, c := range tutorialCases {
				if c.spec == spec {
					cs = casualStats(l, c.casual, 16)
					if c.old != nil {
						cs += " | old " + casualStats(l, c.old, 16)
					}
				}
			}
			t.Logf("k=%.2f qk=%.2f%s | casual %s", k, qk, line, cs)
		}
	}
}

func floats(s string, def float64) []float64 {
	if s == "" {
		return []float64{def}
	}
	var out []float64
	for _, f := range strings.Split(s, ",") {
		v, _ := strconv.ParseFloat(f, 64)
		out = append(out, v)
	}
	return out
}

// mixOf copies a level, scaling the health ("walker:2,swarmling:1.5") and the
// numbers ("swarmling:2", spawned proportionally faster) of enemy types.
func mixOf(spec *LevelSpec, hp, cnt string) *LevelSpec {
	parse := func(s string) map[string]float64 {
		m := map[string]float64{}
		for _, kv := range strings.Split(s, ",") {
			if a, b, ok := strings.Cut(kv, ":"); ok {
				v, _ := strconv.ParseFloat(b, 64)
				m[a] = v
			}
		}
		return m
	}
	hm, cm := parse(hp), parse(cnt)
	wk := parse(os.Getenv("WK")) // per wave: "7:2,8:2.5"
	s := *spec
	s.Waves = make([]WaveSpec, len(spec.Waves))
	for i, wv := range spec.Waves {
		gs := append([]Group(nil), wv.Groups...)
		for j := range gs {
			if k, ok := wk[strconv.Itoa(i+1)]; ok {
				gs[j].HPMul *= k
			}
			if k, ok := wk[strconv.Itoa(i+1)+"n"]; ok {
				n := int(float64(gs[j].Count)*k + 0.5)
				gs[j].Interval *= float64(gs[j].Count) / float64(max(1, n))
				gs[j].Count = n
			}
			if k, ok := hm[gs[j].Enemy]; ok {
				gs[j].HPMul *= k
			} else if k, ok := hm["all"]; ok {
				gs[j].HPMul *= k
			}
			if k, ok := cm[gs[j].Enemy]; ok {
				n := int(float64(gs[j].Count)*k + 0.5)
				gs[j].Interval *= float64(gs[j].Count) / float64(max(1, n))
				gs[j].Count = n
			}
		}
		s.Waves[i] = WaveSpec{Groups: gs}
	}
	return &s
}

// TestMix: MIX=1-5 HP="walker:2;walker:3" CNT="swarmling:2" go test -run TestMix -v
func TestMix(t *testing.T) {
	id := os.Getenv("MIX")
	if id == "" {
		t.Skip()
	}
	spec := LevelByID(id)
	for _, hp := range strings.Split(os.Getenv("HP"), ";") {
		for _, cnt := range strings.Split(os.Getenv("CNT"), ";") {
			ls := mixOf(spec, hp, cnt)
			l, err := ParseLevel(ls)
			if err != nil {
				t.Fatal(err)
			}
			line := ""
			for _, p := range tunePlans[id] {
				if pl := os.Getenv("PL"); pl != "" && !strings.Contains(","+pl+",", ","+p.name+",") {
					continue
				}
				lv := l
				if p.name == "nosuper" {
					lv = noSuper(ls)
				}
				w := NewWorld(lv, 1)
				runToEnd(w, planUp(lv, p.steps, false))
				lk := ""
				if os.Getenv("LEAKS") != "" {
					lk = fmt.Sprint(" ", leakLog)
				}
				line += fmt.Sprintf(" | %s %s r%v%s", p.name, planStats(lv, p.steps, 4), railLog["walker"]*100/max(1, sumMap(railLog)), lk)
			}
			cs := ""
			n, _ := strconv.Atoi(os.Getenv("CASN"))
			for _, c := range tutorialCases {
				if c.spec == spec {
					cs = casualStats(l, c.casual, max(n, 16))
				}
			}
			t.Logf("hp=%s cnt=%s%s | casual %s", hp, cnt, line, cs)
		}
	}
}

func sumMap(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
