package game

import "slices"

// ---- Trophies: one-time rewards shown on the bridge of the Vanguard ----

// TrophyKind is what a trophy is won for.
type TrophyKind int

const (
	TrBoss     TrophyKind = iota // beat a boss level
	TrChapter                    // 3 stars on every level of a chapter
	TrPerfect                    // 3 stars on one level
	TrFlawless                   // a win with the base untouched
	TrClose                      // a win with 10% of the base or less
	TrSupers                     // every super upgrade built once
	TrKills                      // enemies destroyed in all
	TrWins                       // battles won, replays included
	TrCity                       // a colony built up to the City rank
	TrSpecimen                   // enemies of one type destroyed
)

// Trophy is won once and pays its crystals once.
type Trophy struct {
	ID      string
	Name    string
	Desc    string
	Reward  int // crystals
	Kind    TrophyKind
	Goal    int        // counters: the number to reach
	Level   *LevelSpec // TrBoss, TrPerfect
	Chapter *Chapter   // TrChapter
	Enemy   *EnemyDef  // TrSpecimen
	Colony  *Location  // TrCity
	Model   string     // the hologram on the bridge
}

// Trophies stand on the pedestals of the bridge.
var Trophies = []*Trophy{
	{ID: "queen", Name: "Queen Slayer", Desc: "Defeat the Hive Queen in 1-7.", Reward: 30, Kind: TrBoss, Level: HiveHeart, Model: "queen"},
	{ID: "wyrm", Name: "Wyrm Hunter", Desc: "Defeat the Ice Wyrm in 2-7.", Reward: 30, Kind: TrBoss, Level: WyrmLake, Model: "wyrm"},
	{ID: "ch1", Name: "Star of Mars", Desc: "Earn 3 stars on every level of chapter 1.", Reward: 40, Kind: TrChapter, Chapter: Chapter1, Model: "star-mars"},
	{ID: "ch2", Name: "Star of Europa", Desc: "Earn 3 stars on every level of chapter 2.", Reward: 50, Kind: TrChapter, Chapter: Chapter2, Model: "star-europa"},
	{ID: "kepler", Name: "Kepler Veteran", Desc: "Earn 3 stars in the Skirmish at Kepler Outpost.", Reward: 30, Kind: TrPerfect, Level: Skirmish, Model: "kepler"},
	{ID: "flawless", Name: "Flawless", Desc: "Win a battle without the base taking any damage.", Reward: 25, Kind: TrFlawless, Model: "shield"},
	{ID: "close", Name: "Close Call", Desc: "Win a battle with 10% of the base or less.", Reward: 25, Kind: TrClose, Model: "flag"},
	{ID: "supers", Name: "Supercharged", Desc: "Build every super upgrade at least once: Ricochet, Deep Freeze, Napalm, Overcharge and Salvo.", Reward: 30, Kind: TrSupers, Goal: len(TowerOrder), Model: "gem"},
	{ID: "kills1", Name: "Exterminator", Desc: "Destroy 1000 enemies.", Reward: 20, Kind: TrKills, Goal: 1000, Model: "skull"},
	{ID: "kills2", Name: "Hive Breaker", Desc: "Destroy 5000 enemies.", Reward: 40, Kind: TrKills, Goal: 5000, Model: "cup"},
	{ID: "wins", Name: "Commander", Desc: "Win 25 battles. Replays count.", Reward: 30, Kind: TrWins, Goal: 25, Model: "chevrons"},
	{ID: "city", Name: "Mars City", Desc: "Build the Mars colony up to the City rank.", Reward: 30, Kind: TrCity, Colony: Mars, Model: "city"},
}

// Specimens stand in jars on the shelf of the bridge, one per enemy type.
var Specimens = []*Trophy{
	specimen(Crawler, 300), specimen(Runner, 200), specimen(Swarmling, 500), specimen(Walker, 100),
	specimen(Drone, 50), specimen(Beetle, 100), specimen(Splitter, 50), specimen(Stinger, 100),
}

// AllTrophies are the trophies and then the specimens.
var AllTrophies = append(slices.Clone(Trophies), Specimens...)

func specimen(e *EnemyDef, goal int) *Trophy {
	return &Trophy{ID: "jar." + e.ID, Name: e.Name, Desc: "Destroy " + itoa(goal) + " " + e.Plural + ".",
		Reward: 10, Kind: TrSpecimen, Goal: goal, Enemy: e, Model: e.ID}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// TrophyByID finds a trophy or a specimen.
func TrophyByID(id string) *Trophy {
	for _, t := range AllTrophies {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// Record is the progress the trophies are judged on.
type Record struct {
	Stars    map[string]int // best stars per level
	Kills    map[string]int // enemies destroyed per type
	Supers   []string       // turret types whose super has been built
	Wins     int
	Flawless bool // won a battle with the base untouched
	Close    bool // won a battle with 10% of the base or less
	Colony   map[string]int
}

// Progress is how far the record is towards the trophy (have of goal).
func (t *Trophy) Progress(r *Record) (have, goal int) {
	flag := func(b bool) (int, int) {
		if b {
			return 1, 1
		}
		return 0, 1
	}
	switch t.Kind {
	case TrBoss:
		return flag(r.Stars[t.Level.ID] > 0)
	case TrPerfect:
		return flag(r.Stars[t.Level.ID] >= 3)
	case TrChapter:
		for _, l := range t.Chapter.Levels {
			have += min(3, r.Stars[l.ID])
		}
		return have, 3 * len(t.Chapter.Levels)
	case TrFlawless:
		return flag(r.Flawless)
	case TrClose:
		return flag(r.Close)
	case TrSupers:
		for _, d := range TowerOrder {
			if slices.Contains(r.Supers, d.ID) {
				have++
			}
		}
		return have, t.Goal
	case TrKills:
		return min(TotalKills(r.Kills), t.Goal), t.Goal
	case TrWins:
		return min(r.Wins, t.Goal), t.Goal
	case TrCity:
		return t.Colony.BuiltIn(r.Colony), t.Colony.Stages()
	case TrSpecimen:
		return min(r.Kills[t.Enemy.ID], t.Goal), t.Goal
	}
	return 0, 1
}

// Done reports whether the record has won the trophy.
func (t *Trophy) Done(r *Record) bool {
	have, goal := t.Progress(r)
	return have >= goal
}

// Counter reports whether the trophy counts up to its goal (a progress bar)
// rather than being won at once.
func (t *Trophy) Counter() bool {
	switch t.Kind {
	case TrChapter, TrSupers, TrKills, TrWins, TrCity, TrSpecimen:
		return true
	}
	return false
}

// TotalKills adds up the enemies destroyed of every type.
func TotalKills(m map[string]int) int {
	n := 0
	for _, k := range m {
		n += k
	}
	return n
}

// LevelKills is what one victory on a level destroys at least: every enemy
// of its waves and the parts the splitters burst into.
func LevelKills(s *LevelSpec) map[string]int {
	m := map[string]int{}
	for _, w := range s.Waves {
		for _, g := range w.Groups {
			m[g.Enemy] += g.Count
			if d := EnemyDefs[g.Enemy]; d != nil && d.Split != nil {
				m[d.Split.Enemy] += g.Count * d.Split.Count
			}
		}
	}
	return m
}
