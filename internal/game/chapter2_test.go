package game

import "testing"

// Balance cases of chapter 2 (see TestTutorialBalance and TestTune).
//
// Twin Passes pads: 0 (13,2) B only, 1 (4,4) A only, 2 (9,4) both, 3 (17,4)
// shared, 4 (5,7) A only, 5 (9,7) both, 6 (13,7) the merge, 7 (17,8) shared,
// 8 (12,11) shared.
// Beetle Drift pads: 0 (4,0) A, 1 (15,1) shared, 2 (6,4) A, 3 (10,4) both,
// 4 (15,5) the loop, 5 (19,5) shared, 6 (6,8) B, 7 (10,8) both, 8 (18,9)
// shared, 9 (4,12) B.
func init() {
	tunePlans["2-1"] = []tunePlan{
		{"sky2", []step{B(6, bl), B(5, fl), S(7, rl), B(2, sk), B(8, bl), B(3, sk), B(0, bl), B(1, cr)}},
		{"good", []step{B(6, bl), B(5, bl), S(7, rl), B(2, fl), B(3, sk), B(8, cr), B(0, bl), S(4, rl)}},
		{"oneroad", []step{B(1, bl), B(4, bl), B(5, bl), B(2, fl), S(6, rl), B(3, sk), B(8, cr), B(0, bl)}},
		{"noair", []step{B(6, bl), B(5, bl), S(7, rl), B(2, fl), B(3, bl), B(8, cr), B(0, bl), S(4, rl)}},
		{"sky6", []step{B(6, bl), B(5, bl), S(7, rl), B(2, fl), B(8, sk), B(3, cr), B(0, bl), S(4, rl)}},
	}
	tunePlans["2-2"] = []tunePlan{
		{"fire4", []step{B(4, fl), B(7, fl), B(3, fl), S(5, rl), B(1, sk), S(8, rl), B(2, sk), B(6, bl)}},
		{"nofire", []step{B(4, bl), B(7, bl), B(3, bl), S(5, rl), B(1, sk), S(8, rl), B(2, sk), B(6, bl)}},
		{"fire", []step{B(4, fl), B(3, bl), B(7, fl), S(5, rl), B(1, sk), B(8, cr), B(2, bl), B(6, bl)}},
		{"cryo", []step{B(4, cr), B(3, bl), B(7, cr), S(5, rl), B(1, sk), B(8, bl), B(2, bl), B(6, bl)}},
		{"fire2", []step{B(4, fl), B(3, fl), S(5, rl), B(7, bl), B(1, sk), B(8, sk), S(2, rl), B(6, cr)}},
		{"fire3", []step{B(3, bl), B(4, fl), B(7, fl), S(5, rl), B(8, sk), B(1, bl), B(2, cr), B(6, fl)}},
		{"firesky2", []step{B(4, fl), B(3, bl), B(7, fl), S(5, rl), B(1, sk), B(8, sk), B(2, bl), B(6, cr)}},
	}
	// Two Fronts pads (mirrored pairs): A best 4 (7,5) / B best 5 (17,7);
	// A 6 (4,8) / B 3 (20,4); A 1 (4,4) / B 8 (20,8); A 7 (8,8) / B 2 (16,4);
	// A 9 (4,12) / B 0 (20,0).
	splitCasual["2-3"] = true
	tunePlans["2-3"] = []tunePlan{
		{"good", []step{B(4, bl), S(5, rl), B(6, fl), B(3, fl), S(8, rl), B(2, sk), B(1, bl), B(0, bl)}},
		{"good2", []step{B(4, bl), S(5, rl), B(6, fl), B(3, bl), B(2, sk), S(8, rl), B(1, fl), B(0, fl)}},
		{"railB", []step{B(4, bl), S(5, rl), B(6, fl), B(3, bl), S(8, rl), B(2, sk), B(1, fl), B(7, bl)}},
		{"fireA", []step{B(4, fl), S(5, rl), B(6, bl), B(3, bl), S(8, rl), B(2, sk), B(1, fl), B(0, sk)}},
		{"split", []step{B(4, bl), B(5, bl), B(6, fl), B(3, fl), S(1, rl), S(8, rl), B(7, sk), B(2, sk)}},
		{"split2", []step{B(4, fl), B(5, fl), B(6, bl), B(3, bl), S(1, rl), S(8, rl), B(7, sk), B(2, sk)}},
		{"noair", []step{B(4, bl), B(5, bl), B(6, fl), B(3, fl), S(1, rl), S(8, rl), B(7, bl), B(2, bl)}},
		{"oneside", []step{B(4, bl), B(6, fl), S(1, rl), B(7, sk), B(9, fl), B(5, bl), B(3, fl), S(8, rl)}},
		{"lateB", []step{B(4, bl), B(6, fl), S(1, rl), B(5, bl), B(3, fl), S(8, rl), B(7, sk), B(2, sk)}},
	}
	// Shatter Field pads: 0 (2,2) A early, 1 (11,4), 2 (14,4) the loop,
	// 3 (18,4) late, 4 (7,6) the merge, 5 (11,8), 6 (14,8), 7 (20,9) at the
	// base, 8 (2,10) B early, 9 (14,11).
	tunePlans["2-4"] = []tunePlan{
		{"best", []step{B(4, fl), B(2, fl), S(1, rl), B(5, cr), B(3, sk), B(6, bl)}},
		{"noflame", []step{B(4, bl), B(2, bl), S(1, rl), B(5, cr), B(3, sk), B(6, bl)}},
		{"mixed", []step{B(2, fl), B(4, bl), B(1, bl), S(5, rl), B(6, cr), B(3, sk), B(0, bl), B(8, bl)}},
		{"front", []step{B(4, fl), B(1, fl), B(2, bl), S(5, rl), B(6, cr), B(3, sk), B(0, bl), B(8, bl)}},
		{"back", []step{B(3, bl), B(7, bl), B(6, fl), S(2, rl), B(9, cr), B(5, sk), B(1, bl), B(4, bl)}},
		{"oneflame", []step{B(4, fl), B(2, bl), S(1, rl), B(5, cr), B(3, sk), B(6, bl)}},
	}
	tunePlans["2-5"] = []tunePlan{
		{"best", []step{B(2, sk), B(5, bl), B(4, bl), B(1, cr), B(3, sk), B(8, fl)}},
		{"onesky", []step{B(2, sk), B(5, bl), B(4, bl), B(1, cr), B(3, bl), B(8, fl)}},
		{"latesky", []step{B(2, bl), B(5, bl), B(4, bl), B(1, cr), B(3, sk), B(8, fl)}},
		{"ground", []step{B(5, bl), B(4, bl), B(1, cr), B(8, fl), B(2, rl), B(3, bl)}},
	}
	tunePlans["2-6"] = []tunePlan{
		{"best", []step{B(7, sk), B(0, sk), B(6, fl), B(8, rl), B(3, fl), B(5, fl), B(9, cr)}},
		{"nosky", []step{B(7, fl), B(4, bl), B(8, fl), S(6, rl), B(0, fl), B(3, bl), B(11, rl)}},
		{"noflame", []step{B(7, bl), B(4, sk), B(8, bl), S(6, rl), B(0, bl), B(3, sk), B(11, rl)}},
		{"norail", []step{B(7, fl), B(4, sk), B(8, fl), B(6, bl), B(0, fl), B(3, sk), B(11, bl)}},
	}
	// Wyrm Lake pads: 0 (7,0), 1 (3,4), 2 (8,4) above the first lake, 3 (14,4),
	// 4 (16,4), 5 (19,4) by the last stretch, 6 (3,8), 7 (7,8), 8 (11,8) the
	// middle, 9 (15,8) the merge, 10 (18,8), 11 (1,10), 12 (9,11) by B.
	tunePlans["2-7"] = []tunePlan{
		{"best", []step{B(5, sk), B(4, sk), B(10, fl), S(6, rl), B(9, fl), B(3, rl)}},
		{"nofire", []step{B(5, sk), B(4, sk), B(10, bl), S(6, rl), B(9, bl), B(3, rl)}},
		{"norail", []step{B(5, sk), B(4, sk), B(10, fl), B(6, bl), B(9, fl), B(3, bl)}},
		{"alt", []step{B(12, sk), B(0, cr), S(3, rl), B(5, sk), B(9, fl), B(10, bl), S(11, rl)}},
		{"middle", []step{B(7, fl), B(9, fl), S(8, rl), B(12, sk), B(2, bl), S(3, rl), B(10, cr), B(4, sk)}},
	}
	tutorialCases = append(tutorialCases,
		tutorialCase{TwinPasses, nil, nil, []*TowerDef{bl, bl, cr, fl, rl, sk}, nil},
		tutorialCase{BeetleDrift, nil, nil, []*TowerDef{bl, bl, cr, fl, fl, rl, sk}, []*TowerDef{bl, bl, cr, rl, sk}},
		tutorialCase{TwoFronts, nil, nil, []*TowerDef{bl, bl, cr, fl, fl, rl, sk}, nil},
		tutorialCase{ShatterField, nil, nil, []*TowerDef{bl, bl, cr, fl, fl, rl, sk}, []*TowerDef{bl, bl, rl, sk}},
		tutorialCase{StingerRidge, nil, nil, []*TowerDef{bl, bl, cr, fl, rl, sk, sk}, []*TowerDef{bl, bl, cr, fl, rl, sk}},
		tutorialCase{TripleBreach, nil, nil, []*TowerDef{bl, bl, cr, fl, fl, rl, sk, sk}, nil},
		tutorialCase{WyrmLake, nil, nil, []*TowerDef{bl, bl, cr, fl, fl, rl, rl, sk, sk}, nil},
	)
	counters = append(counters, struct {
		enemy  string
		turret *TowerDef
		by     int
	}{"beetle", Flamer, 1}, struct {
		enemy  string
		turret *TowerDef
		by     int
	}{"stinger", Skyguard, 1})
}

// TestWyrmDives: between paired ice holes the Wyrm swims under the ice, where
// turrets cannot reach it, and it spits splitters when it comes up.
func TestWyrmDives(t *testing.T) {
	l, err := ParseLevel(WyrmLake)
	if err != nil {
		t.Fatal(err)
	}
	ice := l.Routes[0].Ice
	if len(ice) != 2 || len(l.Routes[1].Ice) != 0 {
		t.Fatalf("2-7 needs two stretches under the ice on road A only, got %v / %v", ice, l.Routes[1].Ice)
	}
	w := NewWorld(l, 1)
	w.SkipTimer()
	w.pending = w.pending[:0]
	e := w.spawnOn(0, Wyrm, 1, ice[0][0]-0.05, false)
	e.ignite(10, 5)
	pad := -1
	for i, s := range l.Slots {
		if s.Center().Dist(e.Pos) < 3 {
			pad = i
		}
	}
	tw := w.PlaceTower(pad, Blaster, 2)
	w.Update(0.5)
	if !e.Under || e.BurnT > 0 {
		t.Fatalf("the wyrm must dive at the first hole and the water must put out its fire (under %v burn %.1f)", e.Under, e.BurnT)
	}
	if w.pickTarget(tw, 99) == e {
		t.Fatal("turrets must not target a wyrm under the ice")
	}
	start := e.Dist
	w.Update(1)
	if got := e.Dist - start; got < Wyrm.Speed*Wyrm.Dive*0.9 {
		t.Fatalf("under the ice the wyrm must swim %.1f× faster, moved %.2f", Wyrm.Dive, got)
	}
	n := len(w.Enemies)
	for i := 0; i < 2000 && e.Under; i++ {
		w.Update(0.05)
	}
	if e.Under || e.Dist < ice[0][1] {
		t.Fatal("the wyrm must come up at the second hole")
	}
	spat := 0
	for _, o := range w.Enemies {
		if o.Def == Splitter && o.Dist > e.Dist {
			spat++
		}
	}
	if spat != Wyrm.Spit.Count || len(w.Enemies) < n+Wyrm.Spit.Count {
		t.Fatalf("the wyrm must spit %d splitters ahead of itself when it surfaces, got %d", Wyrm.Spit.Count, spat)
	}
}
