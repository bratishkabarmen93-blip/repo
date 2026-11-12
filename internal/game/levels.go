package game

// Chapter is a run of levels unlocked one after another.
type Chapter struct {
	Num    int
	Name   string
	Desc   string
	Levels []*LevelSpec
	Soon   int // levels still to come, shown locked
}

// Chapter1 is the tutorial: every level unlocks one new turret or ability.
var Chapter1 = &Chapter{
	Num:    1,
	Name:   "First Contact",
	Desc:   "Learn to hold the line, one turret at a time.",
	Levels: []*LevelSpec{Landing, Canyon, Hatchery, IronPass, Convoy, Valley, HiveHeart},
}

// Chapters lists the campaign in order.
var Chapters = []*Chapter{Chapter1, Chapter2, Chapter3}

// Skirmish is the full game on one map: every turret and every enemy.
var Skirmish = Level1

// AllLevels lists every playable level: the campaign, then the skirmish.
func AllLevels() []*LevelSpec {
	var out []*LevelSpec
	for _, c := range Chapters {
		out = append(out, c.Levels...)
	}
	return append(out, Skirmish)
}

// LevelByID finds a level by its ID.
func LevelByID(id string) *LevelSpec {
	for _, l := range AllLevels() {
		if l.ID == id {
			return l
		}
	}
	return nil
}

// Map legend: '.' ground, '#' road, 'S' spawn (the first burrow; '2' and '3'
// are more, on the left or top edge), 'B' base (3x3, the road enters its middle
// row from the west), 'o' turret pad. Roads may merge but must not fork.

var Landing = &LevelSpec{
	ID: "1-1", Name: "Landing Zone",
	Desc: "The first bugs crawl out of the burrow. Learn to build and upgrade turrets.",
	Map: []string{
		"......................",
		"......................",
		"S######.......o.......",
		"......#...........BBB.",
		"....o.#.....######BBB.",
		"......#.o..o#.....BBB.",
		"......#.....#..o......",
		"......#######.........",
		"......................",
		".........o............",
		"......................",
	},
	Turrets:      []string{"blaster"},
	NoTarget:     true,
	NoSuper:      true,
	StartCredits: 150,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    30,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 6, 1.8, 0, 1)}},
		{Groups: []Group{g("crawler", 9, 1.4, 0, 1.15)}},
		{Groups: []Group{g("crawler", 12, 1.1, 0, 1.54)}},
		{Groups: []Group{g("crawler", 10, 1.0, 0, 2.12), g("crawler", 8, 0.8, 14, 2.38)}},
	},
	Lesson: &Lesson{
		Turret:   "blaster",
		Problem:  "Hive crawlers are marching on the outpost. Every bug that reaches it damages the base.",
		Solution: "Click a pad next to the road and build a Blaster. Kills and cleared waves pay credits: spend them on upgrades.",
	},
}

var Canyon = &LevelSpec{
	ID: "1-2", Name: "Dry Canyon",
	Desc: "Fast runners sprint through the canyon. Slow them down.",
	Map: []string{
		"........................",
		"...####.................",
		"...#o.#.................",
		".o.#..#....######.......",
		"...#..#....#....#.o.....",
		"...#.o#.o..#.o..#...BBB.",
		"S###..#....#....####BBB.",
		"......#....#........BBB.",
		"......#....#.o..........",
		"......######............",
		"........................",
		"........o...............",
	},
	Turrets:      []string{"blaster", "cryo"},
	NoTarget:     true,
	NoSuper:      true,
	StartCredits: 160,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    35,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 8, 1.4, 0, 1)}},
		{Groups: []Group{g("runner", 8, 0.5, 0, 1.22), g("crawler", 6, 1.2, 8, 0.94)}},
		{Groups: []Group{g("runner", 12, 0.35, 0, 1.62), g("crawler", 8, 1.1, 8, 1.14)}},
		{Groups: []Group{g("runner", 14, 0.3, 0, 2.07), g("crawler", 10, 1.0, 7, 1.42), g("runner", 10, 0.3, 18, 2.07)}},
		{Groups: []Group{g("runner", 18, 0.28, 0, 2.42), g("crawler", 12, 0.9, 8, 1.8), g("runner", 18, 0.28, 20, 2.59)}},
	},
	Lesson: &Lesson{
		Turret:   "cryo",
		Problem:  "Runners are so fast that they slip past your Blasters before they can fire twice.",
		Solution: "Cryo sends out frost waves that slow down every enemy around it. Build it inside the hairpin bend and let your Blasters finish off the slowed runners.",
	},
}

var Hatchery = &LevelSpec{
	ID: "1-3", Name: "Hatchery",
	Desc: "The hive sends its larvae by the dozen.",
	Map: []string{
		"........................",
		"........................",
		"S##############.........",
		"..............#.........",
		"........o...o.#.o.......",
		"..............#.........",
		"....###########.........",
		"....#...................",
		"..o.#.o....o....o.......",
		"....#...............BBB.",
		"....################BBB.",
		"....................BBB.",
		"...........o............",
	},
	Turrets:      []string{"blaster", "cryo", "flamer"},
	NoTarget:     true,
	NoSuper:      true,
	StartCredits: 180,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    40,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 8, 1.2, 0, 1.1), g("swarmling", 16, 0.22, 8, 1.92)}},
		{Groups: []Group{g("swarmling", 26, 0.2, 0, 2.24), g("runner", 6, 0.5, 10, 1.2)}},
		{Groups: []Group{g("crawler", 10, 1.0, 0, 1.4), g("swarmling", 36, 0.18, 6, 2.56)}},
		{Groups: []Group{g("swarmling", 30, 0.17, 0, 3.04), g("runner", 10, 0.4, 8, 1.5), g("swarmling", 30, 0.17, 16, 3.04)}},
		{Groups: []Group{g("crawler", 14, 0.8, 0, 1.9), g("swarmling", 40, 0.15, 6, 3.52), g("runner", 12, 0.35, 14, 1.8), g("swarmling", 40, 0.15, 22, 3.68)}},
	},
	Lesson: &Lesson{
		Turret:   "flamer",
		Problem:  "Swarmlings come in huge crowds. A Blaster kills just one bug per shot, so the rest of the swarm walks past.",
		Solution: "The Flamer sets whole groups alight. Put it where the swarm bunches up, ideally inside a bend of the road.",
	},
}

var IronPass = &LevelSpec{
	ID: "1-4", Name: "Iron Pass",
	Desc: "War robots march through the pass. Their armor shrugs off light fire.",
	Map: []string{
		"........................",
		"........o...............",
		"........................",
		".....#######............",
		".....#.o...#.o......BBB.",
		".....#.....#.....###BBB.",
		".....#..o..#.....#..BBB.",
		".....#.....#.o...#o.....",
		"...o.#.....#.....#......",
		".....#.....#######......",
		"S#####..................",
		"..............o.........",
		"........................",
	},
	Turrets:      []string{"blaster", "cryo", "flamer", "railgun"},
	NoTarget:     true,
	NoSuper:      true,
	StartCredits: 200,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    40,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 8, 1.2, 0, 1.02), g("walker", 1, 1, 10, 0.98)}},
		{Groups: []Group{g("walker", 2, 5, 0, 1.15), g("runner", 8, 0.5, 8, 1.3)}},
		{Groups: []Group{g("walker", 3, 4.5, 0, 1.26), g("swarmling", 20, 0.2, 6, 1.4)}},
		{Groups: []Group{g("crawler", 12, 0.9, 0, 1.78), g("walker", 3, 4, 4, 1.45), g("runner", 10, 0.4, 16, 1.68)}},
		{Groups: []Group{g("walker", 4, 3.5, 0, 1.64), g("swarmling", 26, 0.18, 5, 1.98), g("crawler", 10, 0.8, 14, 2.2)}},
		{Groups: []Group{g("walker", 5, 3, 0, 1.77), g("runner", 12, 0.35, 8, 2.2), g("crawler", 12, 0.8, 14, 2.53), g("walker", 2, 3, 24, 1.89)}},
	},
	Lesson: &Lesson{
		Turret:   "railgun",
		Problem:  "Walkers have 6 armor: every hit deals 6 less damage. Blaster bolts barely scratch them and flames do almost nothing.",
		Solution: "The Railgun fires slow, heavy shots that ignore armor. Give it a long view of the road.",
	},
}

var Convoy = &LevelSpec{
	ID: "1-5", Name: "Convoy Road",
	Desc: "Heavy walkers march in a convoy while bugs rush past them. Choose what your turrets shoot at.",
	Map: []string{
		"......o.................",
		"........................",
		"S####...#####...........",
		"....#...#...#.o.........",
		"....#.o.#.o.#...........",
		"..o.#...#...#...........",
		"....#...#...#...........",
		"....#...#...#...........",
		"....#.o.#.o.#....o......",
		"....#...#...#.......BBB.",
		"....#####...########BBB.",
		"....................BBB.",
		"...............o........",
	},
	Turrets:      []string{"blaster", "cryo", "flamer", "railgun"},
	NoSuper:      true,
	StartCredits: 220,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    40,
	Waves: []WaveSpec{
		{Groups: []Group{g("walker", 1, 1, 0, 2.8), g("crawler", 8, 1.6, 4, 1.2), g("swarmling", 12, 1.0, 8, 1.3)}},
		{Groups: []Group{g("walker", 2, 6, 0, 3.3), g("swarmling", 24, 0.7, 5, 1.5), g("runner", 6, 2, 10, 1.3)}},
		{Groups: []Group{g("walker", 3, 5, 0, 4.2), g("crawler", 12, 1.4, 4, 1.6), g("swarmling", 26, 0.6, 8, 1.8)}},
		{Groups: []Group{g("walker", 4, 4.5, 0, 5.2), g("swarmling", 32, 0.6, 4, 2), g("runner", 10, 1.5, 8, 1.8), g("crawler", 8, 2, 12, 1.9)}},
		{Groups: []Group{g("walker", 5, 4, 0, 6.1), g("crawler", 14, 1.4, 4, 2.1), g("swarmling", 36, 0.55, 8, 2.2)}},
		{Groups: []Group{g("walker", 6, 4, 0, 7), g("swarmling", 40, 0.5, 4, 2.4), g("runner", 12, 1.4, 8, 2.2), g("crawler", 12, 1.6, 12, 2.4)}},
	},
	Lesson: &Lesson{
		Ability:  "Target priority",
		Icon:     "target",
		Problem:  "Bugs keep rushing past the slow walkers. Railguns shoot whatever is closest to the base, so their heavy shots are wasted on small fry.",
		Solution: "Click a Railgun and press T until it says Strong: it will aim at the toughest enemy in range. Leave the bugs to Blasters and Flamers.",
	},
}

var Valley = &LevelSpec{
	ID: "1-6", Name: "Drone Valley",
	Desc: "Repair drones fly with the robots and keep them in one piece.",
	Map: []string{
		"........................",
		"..####.o................",
		"..#..#..................",
		"..#..###.o..........BBB.",
		"..#.o..#.........o##BBB.",
		"..#....###........#.BBB.",
		"S##...o..#..o...###.....",
		".........###....#..o....",
		"...........#.o###.......",
		".........o.#..#.........",
		"...........####.o.......",
		"........................",
		"........................",
	},
	NoSuper:      true,
	StartCredits: 230,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    40,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1.1, 0, 0.85), g("drone", 2, 5, 6, 0.85)}},
		{Groups: []Group{g("walker", 2, 5, 0, 1.04), g("drone", 2, 5, 3, 0.84), g("runner", 10, 0.45, 10, 0.8)}},
		{Groups: []Group{g("swarmling", 28, 0.18, 0, 1.08), g("drone", 3, 4, 4, 0.99), g("crawler", 10, 0.9, 10, 0.99)}},
		{Groups: []Group{g("walker", 3, 4, 0, 1.35), g("drone", 3, 4, 2, 1.08), g("crawler", 14, 0.7, 8, 1.22), g("runner", 10, 0.4, 18, 1.12)}},
		{Groups: []Group{g("crawler", 16, 0.6, 0, 1.5), g("walker", 4, 3.5, 4, 1.7), g("drone", 4, 3.5, 6, 1.3), g("swarmling", 30, 0.15, 16, 1.6)}},
		{Groups: []Group{g("walker", 5, 3, 0, 1.9), g("drone", 5, 3, 2, 1.4), g("runner", 12, 0.35, 10, 1.55), g("crawler", 16, 0.6, 16, 1.7), g("drone", 2, 4, 24, 1.4)}},
	},
	Lesson: &Lesson{
		Turret:   "skyguard",
		Problem:  "Repair drones fly over the road and heal the enemies below them. Ground turrets cannot hit anything in the air.",
		Solution: "Skyguard launches homing missiles at flying enemies only. Build one where the drones fly in, and a second one later.",
	},
}

var HiveHeart = &LevelSpec{
	ID: "1-7", Name: "Hive Heart",
	Desc: "The Hive Queen leads the swarm herself. Bring everything you have.",
	Map: []string{
		".........................",
		"............o............",
		"S#################.......",
		".................#.......",
		".....####..####..#.o.....",
		".....#o.#..#o.#..#.......",
		"...###..####..####.......",
		"...#.....................",
		".o.#.....o.....o.........",
		"...#.................BBB.",
		"...##################BBB.",
		".....................BBB.",
		".............o...........",
	},
	StartCredits: 500,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    120,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 14, 0.8, 0, 2.4), g("runner", 10, 0.4, 10, 2.25)}},
		{Groups: []Group{g("swarmling", 36, 0.16, 0, 3), g("walker", 3, 4, 4, 2.4)}},
		{Groups: []Group{g("crawler", 16, 0.7, 0, 3.15), g("drone", 2, 5, 6, 1.95), g("walker", 4, 3.5, 10, 2.85)}},
		{Groups: []Group{g("runner", 16, 0.3, 0, 3.6), g("swarmling", 40, 0.14, 8, 3.9), g("walker", 4, 3.5, 14, 3.3), g("drone", 3, 4, 16, 2.1)}},
		{Groups: []Group{g("walker", 6, 3, 0, 3.9), g("crawler", 20, 0.6, 4, 4.2), g("drone", 3, 4, 10, 2.25), g("swarmling", 40, 0.13, 22, 4.5)}},
		{Groups: []Group{g("crawler", 24, 0.5, 0, 4.95), g("runner", 18, 0.3, 10, 4.65), g("walker", 7, 2.6, 14, 4.35), g("drone", 4, 3.5, 16, 2.4)}},
		{Groups: []Group{g("walker", 8, 2.4, 0, 5.1), g("swarmling", 60, 0.11, 6, 5.4), g("drone", 5, 3, 10, 2.7), g("crawler", 26, 0.45, 20, 5.7), g("runner", 18, 0.25, 32, 5.4)}},
		{Groups: []Group{g("crawler", 30, 0.4, 0, 8.62), g("walker", 10, 2, 4, 7.76), g("drone", 5, 3, 8, 4.14), g("queen", 1, 1, 14, 2.59), g("swarmling", 60, 0.15, 16, 8.28), g("drone", 5, 3, 22, 4.14), g("runner", 24, 0.25, 26, 8.28), g("crawler", 24, 0.3, 34, 8.62)}},
	},
	Lesson: &Lesson{
		Ability:  "Super upgrades",
		Icon:     "super",
		Problem:  "The Hive Queen leads the whole swarm in the final wave. There are only eight pads, and level 3 turrets alone will not hold.",
		Solution: "A level 3 turret can now take a fourth, super upgrade with a special ability. Spend your spare credits on supers before the last waves.",
	},
}

// Level1 is the original map, now the skirmish: everything is unlocked.
var Level1 = &LevelSpec{
	ID: "kepler", Name: "Kepler Outpost",
	Desc: "The full assault: every turret and every enemy, eight waves and the Hive Queen.",
	Map: []string{
		".........................",
		"..............o..........",
		"S######..................",
		"......#....#######.......",
		"...o..#.o..#.....#.......",
		"......#....#.o...#.......",
		"..#####....#.....#.o.....",
		"..#........#.....#.......",
		"..#.o...o..#...o.#...BBB.",
		"..#........#.....####BBB.",
		"..##########.........BBB.",
		"...................o.....",
		"......o..................",
	},
	StartCredits: 150,
	BaseHP:       100,
	FirstPrep:    30,
	Prep:         30,
	WaveBonus:    40,
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 8, 1.4, 0, 1)}},
		{Groups: []Group{g("crawler", 10, 1.2, 0, 1.15), g("runner", 6, 0.5, 13, 0.89)}},
		{Groups: []Group{g("crawler", 12, 1.0, 0, 1.45), g("swarmling", 20, 0.22, 14, 1.21)}},
		{Groups: []Group{g("walker", 2, 5, 0, 1.51), g("crawler", 12, 0.9, 3, 1.94), g("runner", 8, 0.45, 16, 1.45)}},
		{Groups: []Group{g("crawler", 14, 0.85, 0, 2.64), g("walker", 3, 4.5, 6, 2.1), g("drone", 2, 5, 9, 1), g("swarmling", 16, 0.22, 20, 2.1)}},
		{Groups: []Group{g("runner", 10, 0.4, 0, 2.24), g("swarmling", 24, 0.2, 8, 2.52), g("walker", 4, 3.5, 12, 2.64), g("drone", 1, 1, 14, 1.15), g("runner", 10, 0.4, 26, 2.24)}},
		{Groups: []Group{g("crawler", 20, 0.7, 0, 3.48), g("walker", 5, 3, 6, 2.87), g("drone", 3, 4, 10, 1.2), g("swarmling", 30, 0.18, 18, 2.77), g("runner", 12, 0.35, 30, 2.44)}},
		{Groups: []Group{g("crawler", 16, 0.8, 0, 4.36), g("runner", 10, 0.4, 6, 2.93), g("walker", 4, 3, 10, 3.45), g("drone", 2, 5, 12, 1.45), g("queen", 1, 1, 18, 1), g("drone", 2, 5, 24, 1.45), g("swarmling", 20, 0.2, 28, 3.45)}},
	},
}
