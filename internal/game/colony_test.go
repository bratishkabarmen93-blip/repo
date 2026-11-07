package game

import (
	"strings"
	"testing"
)

func TestEveryLevelHasTier(t *testing.T) {
	for _, s := range AllLevels() {
		if s.Tier < TierLow || s.Tier > TierHigh {
			t.Errorf("%s has no threat tier", s.ID)
		}
	}
}

func TestPay(t *testing.T) {
	cases := []struct {
		s           *LevelSpec
		stars, paid int
		want        Payout
	}{
		{Landing, 3, 0, Payout{3, 0, 30}},
		{Landing, 3, 3, Payout{0, 3, 9}},
		{IronPass, 2, 0, Payout{2, 0, 30}},
		{HiveHeart, 2, 1, Payout{1, 1, 25}},
		{HiveHeart, 1, 3, Payout{0, 1, 5}},
		{WyrmLake, 3, 2, Payout{1, 2, 30}},
	}
	for _, c := range cases {
		if got := Pay(c.s, c.stars, c.paid); got != c.want {
			t.Errorf("Pay(%s, %d, %d) = %+v, want %+v", c.s.ID, c.stars, c.paid, got, c.want)
		}
	}
}

func inside(p Vec, c Cell, w, h int) bool {
	const e = 0.01
	return p.X > float64(c.X)+e && p.X < float64(c.X+w)-e && p.Y > float64(c.Y)+e && p.Y < float64(c.Y+h)-e
}

func TestColonyLayout(t *testing.T) {
	for _, l := range Colonies {
		if l.Soon {
			continue
		}
		ids := map[string]bool{}
		type box struct {
			name string
			c    Cell
			w, h int
		}
		boxes := []box{{"HQ", l.HQ, 3, 3}}
		for _, b := range l.Buildings {
			if !strings.HasPrefix(b.ID, l.ID+".") || ids[b.ID] {
				t.Errorf("%s: bad or repeated ID", b.ID)
			}
			ids[b.ID] = true
			if b.Lot.X < 1 || b.Lot.Y < 1 || b.Lot.X+b.W > l.W-1 || b.Lot.Y+b.H > l.H-1 {
				t.Errorf("%s: lot too close to the map edge", b.ID)
			}
			if len(b.Stages) == 0 {
				t.Errorf("%s: no stages", b.ID)
			}
			for _, s := range b.Stages {
				if s.Cost <= 0 || s.Text == "" {
					t.Errorf("%s: bad stage %+v", b.ID, s)
				}
			}
			boxes = append(boxes, box{b.ID, b.Lot, b.W, b.H})
		}
		for i, a := range boxes {
			for _, b := range boxes[i+1:] {
				if a.c.X < b.c.X+b.w && b.c.X < a.c.X+a.w && a.c.Y < b.c.Y+b.h && b.c.Y < a.c.Y+a.h {
					t.Errorf("%s overlaps %s", a.name, b.name)
				}
			}
		}
		// Streets never run through a lot (the avenue may pass under the HQ).
		for si, st := range l.Streets() {
			for k := 0; k <= 40; k++ {
				p := st[0].Add(st[1].Sub(st[0]).Mul(float64(k) / 40))
				for bi, b := range boxes {
					if bi == 0 && si == 0 {
						continue
					}
					if inside(p, b.c, b.w, b.h) {
						t.Errorf("street %d runs through %s at %v", si, b.name, p)
					}
				}
			}
		}
		t.Logf("%s: %d buildings, %d stages, %d crystals in all", l.Name, len(l.Buildings), l.Stages(), l.Cost())
	}
}

func TestRank(t *testing.T) {
	cases := []struct {
		built, total int
		name         string
		next         int
	}{{0, 14, "Outpost", 25}, {4, 14, "Settlement", 60}, {9, 14, "Colony", 100}, {14, 14, "City", -1}}
	for _, c := range cases {
		if n, nx := Rank(c.built, c.total); n != c.name || nx != c.next {
			t.Errorf("Rank(%d, %d) = %s %d, want %s %d", c.built, c.total, n, nx, c.name, c.next)
		}
	}
}

func TestEuropaAvailabilityAndEconomy(t *testing.T) {
	if Europa.Soon || Europa.Level != WyrmLake || Europa.Biome != "ice" {
		t.Fatal("Europa must be an ice colony unlocked by Wyrm Lake")
	}
	if Europa.Stages() != 14 || Europa.Cost() != 615 {
		t.Fatalf("unexpected Europa economy: %d stages, %d crystals", Europa.Stages(), Europa.Cost())
	}
	for _, id := range []string{"europa.habitat", "europa.drill", "europa.greenhouse", "europa.lab", "europa.thermal", "europa.pad", "europa.monument"} {
		if Europa.Building(id) == nil {
			t.Errorf("missing %s", id)
		}
	}
	record := map[string]int{"mars.habitat": 3, "europa.habitat": 2, "europa.drill": 1}
	if Mars.BuiltIn(record) != 3 || Europa.BuiltIn(record) != 3 {
		t.Fatal("colonies must have separate progress")
	}
}
