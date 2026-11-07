package game

// ---- Crystals: the reward of victories, spent on the colonies ----

// Tier is how hard a level is; it sets how many crystals its stars pay.
type Tier int

const (
	TierLow Tier = iota + 1
	TierMedium
	TierHigh
)

func (t Tier) String() string {
	switch t {
	case TierLow:
		return "Low"
	case TierMedium:
		return "Medium"
	case TierHigh:
		return "High"
	}
	return "?"
}

// The threat of every level.
func init() {
	for _, s := range []*LevelSpec{Landing, Canyon, Hatchery} {
		s.Tier = TierLow
	}
	for _, s := range []*LevelSpec{IronPass, Convoy, Valley, TwinPasses, BeetleDrift} {
		s.Tier = TierMedium
	}
	for _, s := range []*LevelSpec{HiveHeart, TwoFronts, ShatterField, StingerRidge, TripleBreach, WyrmLake, Level1} {
		s.Tier = TierHigh
	}
}

// StarPay is what one star of a level pays: first when it is earned for the
// first time, again when it is earned again on a replay.
func StarPay(t Tier) (first, again int) {
	switch t {
	case TierLow:
		return 10, 3
	case TierMedium:
		return 15, 4
	case TierHigh:
		return 20, 5
	}
	return 0, 0
}

// Payout is the crystals of a won level.
type Payout struct {
	New, Again int // stars earned for the first time, stars earned again
	Total      int
}

// Pay works out the crystals for winning s with the given stars when the
// first paid stars of it have been paid before.
func Pay(s *LevelSpec, stars, paid int) Payout {
	first, again := StarPay(s.Tier)
	p := Payout{New: max(0, stars-paid), Again: min(stars, paid)}
	p.Total = p.New*first + p.Again*again
	return p
}

// ---- Colonies: locations freed by beating a chapter boss ----

// Location is a world where the colonists settle once its boss is beaten.
type Location struct {
	ID, Name  string
	Boss      string     // the beaten boss, shown on the tab
	Level     *LevelSpec // the boss level that frees it
	Desc      string
	Biome     string
	W, H      int     // map size in cells
	HQ        Cell    // top-left cell of the 3×3 command center
	Avenue    float64 // row (cells) of the main street through the command center
	Buildings []*Building
	Soon      bool // not built yet: a locked tab
}

// Building is a colony building on a fixed lot; every stage makes it bigger.
type Building struct {
	ID     string // "mars.habitat"
	Name   string
	Desc   string
	Lot    Cell // top-left cell
	W, H   int
	Stages []Stage
}

// Stage is one step of a building.
type Stage struct {
	Cost int
	Text string
}

// Colonies are the locations in story order.
var Colonies = []*Location{Mars, Europa}

// Mars is freed by beating the Hive Queen in 1-7.
var Mars = &Location{
	ID: "mars", Name: "Mars", Boss: "Hive Queen", Level: HiveHeart,
	Desc: "With the Hive Queen gone, the Kepler Outpost can grow into a colony.",
	W:    24, H: 13, HQ: Cell{10, 5}, Avenue: 6.5,
	Buildings: []*Building{
		{ID: "mars.habitat", Name: "Habitat", Lot: Cell{6, 2}, W: 2, H: 2,
			Desc:   "Pressurized homes. Every stage brings more colonists to Mars.",
			Stages: []Stage{{20, "A living dome for 3 colonists"}, {35, "A second dome and a tunnel, 6 colonists"}, {55, "A residential tower, 9 colonists"}}},
		{ID: "mars.solar", Name: "Solar Farm", Lot: Cell{15, 9}, W: 2, H: 2,
			Desc:   "Panels that catch the weak Martian sun and keep the lights on.",
			Stages: []Stage{{20, "Two rows of solar panels"}, {35, "Four rows and a battery bank"}}},
		{ID: "mars.greenhouse", Name: "Greenhouse", Lot: Cell{6, 9}, W: 2, H: 2,
			Desc:   "Fresh food grown under glass in Martian soil.",
			Stages: []Stage{{25, "A glass tunnel full of plants"}, {40, "A second tunnel and a water tank"}}},
		{ID: "mars.workshop", Name: "Workshop", Lot: Cell{17, 2}, W: 2, H: 2,
			Desc:   "Builds and repairs everything the colony needs.",
			Stages: []Stage{{30, "A workshop hangar"}, {45, "A second bay and a loading crane"}}},
		{ID: "mars.comms", Name: "Comms Array", Lot: Cell{14, 2}, W: 2, H: 2,
			Desc:   "Keeps the colony in touch with the fleet in orbit.",
			Stages: []Stage{{25, "A radio mast"}, {40, "A deep space dish"}}},
		{ID: "mars.pad", Name: "Landing Pad", Lot: Cell{20, 5}, W: 3, H: 3,
			Desc:   "Supply shuttles from the fleet land here.",
			Stages: []Stage{{35, "A landing pad with lights"}, {55, "A control tower: shuttles start to arrive"}}},
		{ID: "mars.monument", Name: "Queen Monument", Lot: Cell{1, 5}, W: 3, H: 3,
			Desc:   "A bronze Hive Queen on a pedestal, in memory of the battle for the Hive Heart.",
			Stages: []Stage{{80, "The statue on its pedestal"}}},
	},
}

// Europa is freed when the Ice Wyrm falls in 2-7.
var Europa = &Location{
	ID: "europa", Name: "Europa", Boss: "Ice Wyrm", Level: WyrmLake, Biome: "ice",
	Desc: "Beyond the ice, a new colony rises on the frozen moon.",
	W:    24, H: 13, HQ: Cell{10, 5}, Avenue: 6.5,
	Buildings: []*Building{
		{ID: "europa.habitat", Name: "Polar Habitat", Lot: Cell{6, 2}, W: 2, H: 2,
			Desc:   "Insulated homes shelter settlers from Europa's cold.",
			Stages: []Stage{{25, "A warm dome for 3 colonists"}, {40, "A second dome, 6 colonists"}, {60, "A residential tower, 9 colonists"}}},
		{ID: "europa.drill", Name: "Ice Drill", Lot: Cell{15, 9}, W: 2, H: 2,
			Desc:   "An industrial drill reaches through Europa's thick ice.",
			Stages: []Stage{{30, "Drill platform and tower"}, {45, "A processing tank and extra machinery"}}},
		{ID: "europa.greenhouse", Name: "Hydroponics", Lot: Cell{6, 9}, W: 2, H: 2,
			Desc:   "Warm, lit gardens grow food beneath the ice.",
			Stages: []Stage{{30, "A protected growing tunnel"}, {45, "A second tunnel and water tank"}}},
		{ID: "europa.lab", Name: "Research Lab", Lot: Cell{17, 2}, W: 2, H: 2,
			Desc:   "Scientists study Europa and the creatures beneath its ice.",
			Stages: []Stage{{35, "A research module"}, {50, "An observation dome and scanner"}}},
		{ID: "europa.thermal", Name: "Thermal Plant", Lot: Cell{14, 2}, W: 2, H: 2,
			Desc:   "A geothermal plant warms the colony through the long night.",
			Stages: []Stage{{30, "A heat generator and steam vent"}, {45, "A second generator and storage tank"}}},
		{ID: "europa.pad", Name: "Landing Pad", Lot: Cell{20, 5}, W: 3, H: 3,
			Desc:   "A landing site for supply shuttles from the Vanguard.",
			Stages: []Stage{{35, "A landing pad with lights"}, {55, "A control tower: shuttles start to arrive"}}},
		{ID: "europa.monument", Name: "Wyrm Monument", Lot: Cell{1, 5}, W: 3, H: 3,
			Desc:   "A memorial to the Ice Wyrm and the battle for Europa.",
			Stages: []Stage{{90, "The Ice Wyrm on its pedestal"}}},
	},
}

// BuiltIn counts the stages built of the location in a colony record
// (stages per building ID).
func (l *Location) BuiltIn(m map[string]int) int {
	n := 0
	for _, b := range l.Buildings {
		n += min(m[b.ID], len(b.Stages))
	}
	return n
}

// Building finds a building of the location by ID.
func (l *Location) Building(id string) *Building {
	for _, b := range l.Buildings {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// Stages counts the stages of all buildings of the location.
func (l *Location) Stages() int {
	n := 0
	for _, b := range l.Buildings {
		n += len(b.Stages)
	}
	return n
}

// Cost is the price of every stage of the location.
func (l *Location) Cost() int {
	n := 0
	for _, b := range l.Buildings {
		for _, s := range b.Stages {
			n += s.Cost
		}
	}
	return n
}

// HQCenter is the middle of the command center (cells).
func (l *Location) HQCenter() Vec { return Vec{float64(l.HQ.X) + 1.5, float64(l.HQ.Y) + 1.5} }

// Center is the middle of a building's lot (cells).
func (b *Building) Center() Vec {
	return Vec{float64(b.Lot.X) + float64(b.W)/2, float64(b.Lot.Y) + float64(b.H)/2}
}

// Door is where a building meets its street (cells): the side of the lot
// that faces the avenue, or the command center for lots on the avenue.
func (l *Location) Door(b *Building) Vec {
	c := b.Center()
	switch {
	case c.Y < l.Avenue-0.01:
		return Vec{c.X, float64(b.Lot.Y + b.H)}
	case c.Y > l.Avenue+0.01:
		return Vec{c.X, float64(b.Lot.Y)}
	case c.X < l.HQCenter().X:
		return Vec{float64(b.Lot.X + b.W), c.Y}
	}
	return Vec{float64(b.Lot.X), c.Y}
}

// Junction is where the street of a building joins the avenue (cells).
func (l *Location) Junction(b *Building) Vec { return Vec{l.Door(b).X, l.Avenue} }

// Streets are the paved segments of the location (cells): the avenue first,
// then one spur per building off the avenue.
func (l *Location) Streets() [][2]Vec {
	hq := l.HQCenter()
	x0, x1 := hq.X-1.5, hq.X+1.5
	for _, b := range l.Buildings {
		j := l.Junction(b)
		x0, x1 = min(x0, j.X), max(x1, j.X)
	}
	out := [][2]Vec{{{x0, l.Avenue}, {x1, l.Avenue}}}
	for _, b := range l.Buildings {
		if d, j := l.Door(b), l.Junction(b); d != j {
			out = append(out, [2]Vec{j, d})
		}
	}
	return out
}

// Ranks name a colony by the share of its stages that are built.
var Ranks = []struct {
	Name string
	Pct  int
}{{"Outpost", 0}, {"Settlement", 25}, {"Colony", 60}, {"City", 100}}

// Rank is the name of a colony with built of its total stages, and the share
// (in percent) where the next rank starts (-1 at the top).
func Rank(built, total int) (string, int) {
	p := 0
	if total > 0 {
		p = built * 100 / total
	}
	name, next := Ranks[0].Name, -1
	for i, r := range Ranks {
		if p >= r.Pct {
			name, next = r.Name, -1
			if i+1 < len(Ranks) {
				next = Ranks[i+1].Pct
			}
		}
	}
	return name, next
}
