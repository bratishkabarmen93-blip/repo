package game

import "testing"

func fullRecord() *Record {
	r := &Record{Stars: map[string]int{}, Kills: map[string]int{}, Colony: map[string]int{}, Wins: 30, Flawless: true, Close: true}
	for _, c := range Chapters {
		for _, l := range c.Levels {
			r.Stars[l.ID] = 3
		}
	}
	r.Stars[Skirmish.ID] = 3
	for _, d := range TowerOrder {
		r.Supers = append(r.Supers, d.ID)
	}
	for _, e := range EnemyOrder {
		r.Kills[e.ID] = 1000
	}
	for _, b := range Mars.Buildings {
		r.Colony[b.ID] = len(b.Stages)
	}
	return r
}

func TestTrophyList(t *testing.T) {
	seen := map[string]bool{}
	total := 0
	for _, tr := range AllTrophies {
		if seen[tr.ID] {
			t.Errorf("duplicate trophy %s", tr.ID)
		}
		seen[tr.ID] = true
		if tr.Reward <= 0 || tr.Name == "" || tr.Desc == "" || tr.Model == "" {
			t.Errorf("trophy %s is incomplete", tr.ID)
		}
		if TrophyByID(tr.ID) != tr {
			t.Errorf("TrophyByID(%s)", tr.ID)
		}
		total += tr.Reward
	}
	if len(Trophies) != 12 || len(Specimens) != 8 {
		t.Errorf("%d trophies, %d specimens", len(Trophies), len(Specimens))
	}
	t.Logf("all trophies pay %d crystals", total)
}

func TestTrophyProgress(t *testing.T) {
	empty := &Record{}
	full := fullRecord()
	for _, tr := range AllTrophies {
		if tr.Done(empty) {
			t.Errorf("%s is won by an empty record", tr.ID)
		}
		if !tr.Done(full) {
			h, g := tr.Progress(full)
			t.Errorf("%s is not won by a full record (%d/%d)", tr.ID, h, g)
		}
	}
	// A single missing star keeps the chapter trophy.
	full.Stars[Chapter1.Levels[3].ID] = 2
	if tr := TrophyByID("ch1"); tr.Done(full) {
		t.Error("ch1 won with 2 stars on 1-4")
	} else if h, g := tr.Progress(full); h != g-1 {
		t.Errorf("ch1 progress %d/%d", h, g)
	}
	// Kills add up over all types.
	r := &Record{Kills: map[string]int{"crawler": 600, "runner": 500}}
	if !TrophyByID("kills1").Done(r) || TrophyByID("kills2").Done(r) {
		t.Error("kill trophies")
	}
	if h, _ := TrophyByID("jar.crawler").Progress(r); h != 300 {
		t.Errorf("crawler jar progress %d", h)
	}
}

func TestLevelKills(t *testing.T) {
	m := LevelKills(LevelByID("2-4"))
	if m["splitter"] == 0 || m["shard"] != 4*m["splitter"] {
		t.Errorf("2-4 kills %v", m)
	}
	n := 0
	for _, c := range Chapters {
		for _, l := range c.Levels {
			n += TotalKills(LevelKills(l))
		}
	}
	// A whole campaign reaches the first kill trophy but not the second.
	if n < 1000 || n >= 5000 {
		t.Errorf("campaign kills %d", n)
	}
	t.Logf("one campaign destroys %d enemies", n)
}
