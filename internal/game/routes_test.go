package game

import (
	"math"
	"strings"
	"testing"
)

func testSpec(rows ...string) *LevelSpec {
	return &LevelSpec{ID: "t", Map: rows, StartCredits: 1000, BaseHP: 100, FirstPrep: 1, Prep: 1,
		Waves: []WaveSpec{{Groups: []Group{g("crawler", 1, 1, 0, 1)}}}}
}

var mergeMap = []string{
	"....2.......",
	"....#.......",
	"S####....BBB",
	"....#....BBB",
	"....#####BBB",
}

func TestRoutesMerge(t *testing.T) {
	l, err := ParseLevel(testSpec(mergeMap...))
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Routes) != 2 {
		t.Fatalf("%d routes, want 2", len(l.Routes))
	}
	a, b := l.Routes[0], l.Routes[1]
	if a.Spawn != (Cell{0, 2}) || b.Spawn != (Cell{4, 0}) {
		t.Fatalf("spawns %v %v", a.Spawn, b.Spawn)
	}
	if a.Join != len(a.Cells) {
		t.Errorf("the first route never joins another (join %d of %d)", a.Join, len(a.Cells))
	}
	// B walks down to the junction (4,2) and then on A's road.
	if b.Cells[b.Join] != (Cell{4, 2}) {
		t.Errorf("route B joins at %v, want 4,2", b.Cells[b.Join])
	}
	ta, tb := a.Cells[len(a.Cells)-5:], b.Cells[len(b.Cells)-5:]
	for i := range ta {
		if ta[i] != tb[i] {
			t.Fatalf("the routes must share the road after the junction: %v vs %v", ta, tb)
		}
	}
	if a.EndDir != (Cell{1, 0}) || b.EndDir != (Cell{1, 0}) {
		t.Errorf("both routes enter the gate eastwards")
	}
	if math.Abs(a.Path.Len-b.Path.Len-2) > 0.3 {
		t.Errorf("route lengths %.2f %.2f", a.Path.Len, b.Path.Len)
	}
	if l.Path != a.Path || l.Spawn != a.Spawn {
		t.Error("the level's main path is the first route")
	}
}

func TestRoutesErrors(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want string
	}{
		{"fork", []string{
			".####BBB",
			"S#...BBB",
			".####BBB",
		}, "forks"},
		{"dead end", []string{
			"S####BBB",
			"..#..BBB",
			".....BBB",
		}, "not on any route"},
		{"gap", []string{
			".....BBB",
			"S#.##BBB",
			".....BBB",
		}, "does not reach"},
		{"numbering", []string{
			"S...#BBB",
			"....#BBB",
			"3####BBB",
		}, "numbered"},
	}
	for _, c := range cases {
		_, err := ParseLevel(testSpec(c.rows...))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.want)
		}
	}
}

func TestRoutesSpawnAndTarget(t *testing.T) {
	spec := testSpec(mergeMap...)
	spec.Waves = []WaveSpec{{Groups: []Group{g("crawler", 2, 1, 0, 1), g("runner", 2, 1, 0, 1).Via(EntB)}}}
	l, err := ParseLevel(spec)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorld(l, 1)
	w.SkipTimer()
	w.Update(0.05)
	var onB int
	for _, e := range w.Enemies {
		if e.Route == EntB {
			onB++
			if e.Def != Runner {
				t.Errorf("%s came out of burrow B", e.Def.ID)
			}
			if e.Pos.Dist(l.Routes[EntB].Spawn.Center()) > 0.5 {
				t.Errorf("enemy on B starts at %v", e.Pos)
			}
		}
	}
	if onB != 1 {
		t.Fatalf("%d enemies on B, want 1", onB)
	}
	// Brood inherits the route of its mother.
	q := w.spawnOn(EntB, Queen, 1, 3, false)
	w.layBrood(q, Queen.Brood)
	for _, e := range w.Enemies {
		if e.Def == Swarmling && e.Route != EntB {
			t.Error("brood must walk the queen's route")
		}
	}
	// "First" means closest to the base, whatever the route.
	w2 := NewWorld(l, 1)
	near := w2.spawnOn(EntB, Crawler, 1, 6, false) // B is 2 shorter: 6 along B is 8 along A
	far := w2.spawnOn(EntA, Crawler, 1, 7, false)
	tw := w2.PlaceTower(0, Blaster, 0)
	_ = tw
	tower := &Tower{Def: Blaster, Mode: TargetFirst, Pos: l.BaseCenter()}
	if tower.score(near) <= tower.score(far) {
		t.Errorf("first: %.2f left on B must beat %.2f left on A", near.Left(), far.Left())
	}
	tower.Mode = TargetLast
	if tower.score(far) <= tower.score(near) {
		t.Error("last picks the enemy furthest from the base")
	}
	// Enemies of both routes reach the base.
	w3 := NewWorld(l, 1)
	w3.SkipTimer()
	for i := 0; i < 4000 && w3.Phase == PhaseWave; i++ {
		w3.Update(0.05)
	}
	if w3.Leaks != 4 {
		t.Errorf("%d leaks, want 4", w3.Leaks)
	}
}

func TestBeetle(t *testing.T) {
	l, _ := ParseLevel(Level1)
	w := NewWorld(l, 1)
	b := w.spawn(Beetle, 1, 5, false)
	c := w.spawn(Crawler, 1, 5, false)
	b.chill(0.5, 3)
	c.chill(0.5, 3)
	if b.SpeedMul() != 1 || c.SpeedMul() == 1 {
		t.Fatal("frost slows crawlers, not beetles")
	}
	// A deep-freeze cryo next to both.
	pos := b.Pos
	tw := &Tower{Def: Cryo, Level: 3, Pos: pos, Pulses: Cryo.Super.FreezeEvery - 1}
	w.updatePulse(tw, tw.Stats().Range, 1)
	if b.FreezeT > 0 || b.SlowT > 0 || c.FreezeT <= 0 {
		t.Fatal("deep freeze freezes crawlers, not beetles")
	}
	// Fire deals double damage: flames and burning.
	hp := b.HP
	ft := &Tower{Def: Flamer, Pos: pos.Add(Vec{-0.5, 0}), Angle: 0}
	w.flameTick(ft, b, Flamer.Levels[0], Flamer.Levels[0].Range)
	if got, want := hp-b.HP, 2*Flamer.Levels[0].Damage; math.Abs(got-want) > 1e-9 {
		t.Fatalf("flame hit %.2f, want %.2f", got, want)
	}
	hp = b.HP
	b.BurnT, b.BurnDPS, b.burnCD = 1, 4, 0.001
	w.updateEnemies(0.002)
	if got, want := hp-b.HP, 2*4*burnTick; math.Abs(got-want) > 1e-9 {
		t.Fatalf("burn tick %.2f, want %.2f", got, want)
	}
}
