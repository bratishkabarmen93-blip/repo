package game

// Chapter3 takes the war to Titan, the hazy moon of Saturn: banks of methane
// fog drift over the dunes and cut the range of the turrets inside them.
var Chapter3 = &Chapter{
	Num:    3,
	Name:   "Methane Mist",
	Desc:   "Saturn's hazy moon, where drifting methane fog hides the hive.",
	Levels: []*LevelSpec{HuygensLanding, ShangriLa, LigeiaShore, XanaduRidge, SotraPatera, SelkCrater, KrakenMare},
}

// The Beacon arrives on Titan: the levels before it keep the classic five.
func init() {
	for _, c := range []*Chapter{Chapter1, Chapter2} {
		for _, s := range c.Levels {
			if s.Turrets == nil {
				s.Turrets = ClassicTurrets
			}
		}
	}
	if Skirmish.Turrets == nil {
		Skirmish.Turrets = ClassicTurrets
	}
}

// Huygens Landing pads: 0 (5,1), 1 (2,3), 2 (8,4) the first bend, 3 (12,4),
// 4 (16,4), 5 (6,8), 6 (11,8) between the roads, 7 (16,8), 8 (19,8) at the
// gate, 9 (9,12).
var HuygensLanding = &LevelSpec{
	ID: "3-1", Name: "Huygens Landing", Biome: "titan",
	Desc: "The first steps on Titan. Banks of methane fog drift across the dunes.",
	Map: []string{
		"........................",
		".....o..................",
		"S##########.............",
		"..o.......#.............",
		"........o.#.o...o.......",
		"..........#.........BBB.",
		"..........##########BBB.",
		".............#......BBB.",
		"......o....o.#..o..o....",
		".............#..........",
		"2#############..........",
		"........................",
		".........o..............",
	},
	StartCredits: 420,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    65,
	Turrets:      ClassicTurrets,
	Fog: []FogBank{
		Bank(2.3, 12.5, 4.5, 6.5, 8.0, 16.5, 7.0),
		Bank(2.2, 3.5, 3.0, 11.5, 9.0),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1.1, 0, 1.2), g("runner", 8, 0.5, 6, 1.2).Via(EntB)}},
		{Groups: []Group{g("walker", 2, 5, 0, 1.3), g("swarmling", 24, 0.16, 4, 1.4).Via(EntB), g("stinger", 5, 0.3, 12, 1.1)}},
		{Groups: []Group{g("beetle", 4, 2, 0, 1.3), g("crawler", 12, 0.8, 2, 1.6).Via(EntB), g("drone", 2, 5, 6, 1.2)}},
		{Groups: []Group{g("splitter", 5, 1.8, 0, 1.5), g("runner", 12, 0.4, 4, 1.7).Via(EntB), g("stinger", 8, 0.28, 10, 1.3)}},
		{Groups: []Group{g("walker", 3, 4, 0, 1.8), g("beetle", 5, 1.8, 2, 1.5).Via(EntB), g("swarmling", 28, 0.14, 8, 2), g("drone", 2, 5, 10, 1.4).Via(EntB)}},
		{Groups: []Group{g("crawler", 16, 0.6, 0, 2.1), g("splitter", 6, 1.6, 2, 1.8).Via(EntB), g("stinger", 10, 0.26, 8, 1.5), g("walker", 3, 4, 12, 2).Via(EntB)}},
		{Groups: []Group{g("runner", 14, 0.35, 0, 2.1), g("walker", 4, 3.5, 4, 2.2).Via(EntB), g("beetle", 7, 1.5, 6, 1.7), g("drone", 3, 4, 10, 1.7).Via(EntB), g("stinger", 10, 0.25, 14, 1.6)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.3), g("splitter", 6, 1.5, 4, 2.1).Via(EntB), g("drone", 4, 3.5, 6, 1.8), g("swarmling", 34, 0.13, 10, 2.4).Via(EntB),
			g("runner", 14, 0.32, 18, 2.2), g("stinger", 12, 0.25, 22, 1.7).Via(EntB)}},
	},
	Lesson: &Lesson{
		Threat:   "Methane fog",
		Icon:     "fog",
		Problem:  "Banks of methane fog drift over Titan. A turret standing in the fog loses a third of its range.",
		Solution: "Before every wave, dashed outlines show where the fog will drift. Build where it won't settle, or give the turrets it covers more range.",
	},
}

// Shangri-La pads: 0 (4,3), 1 (9,3), 2 (13,3), 3 (15,5) by the loop, 4 (4,6),
// 5 (9,7), 6 (11,7), 7 (15,7) inside the loop, 8 (19,8), 9 (4,9), 10 (10,9),
// 11 (15,11), 12 (8,12). Pads 5, 6 and 10 make a triangle: a Beacon on one of
// them lights the other two.
var ShangriLa = &LevelSpec{
	ID: "3-2", Name: "Shangri-La Dunes", Biome: "titan",
	Desc: "The dark dune sea of Shangri-La. The fog never leaves the best pads alone.",
	Tip:  "Build a Beacon among your turrets: its light clears the fog and stretches their range.",
	Map: []string{
		"........................",
		"S######.................",
		"......#.................",
		"....o.#..o...o..........",
		"......#.................",
		"......########.o....BBB.",
		"....o........#...###BBB.",
		".........o.o.#.o.#..BBB.",
		".............#...#.o....",
		"....o.....o..#####......",
		"2#############..........",
		"...............o........",
		"........o...............",
	},
	StartCredits: 430,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    65,
	Fog: []FogBank{
		Bank(2.4, 10.5, 8.0, 11.0, 3.5),
		Bank(2.3, 15.5, 6.5, 18.5, 9.0),
		Bank(2.2, 5.0, 4.5, 6.0, 10.5),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("runner", 10, 0.5, 0, 1.3), g("crawler", 10, 1, 4, 1.3).Via(EntB)}},
		{Groups: []Group{g("beetle", 4, 2, 0, 1.4), g("stinger", 6, 0.3, 6, 1.2).Via(EntB), g("swarmling", 24, 0.15, 10, 1.5)}},
		{Groups: []Group{g("walker", 3, 4.5, 0, 1.5), g("splitter", 4, 2, 2, 1.5).Via(EntB), g("drone", 2, 5, 8, 1.3)}},
		{Groups: []Group{g("crawler", 14, 0.7, 0, 1.8), g("beetle", 5, 1.8, 2, 1.6).Via(EntB), g("stinger", 8, 0.28, 8, 1.4), g("walker", 2, 5, 12, 1.8).Via(EntB)}},
		{Groups: []Group{g("splitter", 6, 1.7, 0, 1.8), g("runner", 14, 0.38, 4, 1.9).Via(EntB), g("drone", 3, 4, 8, 1.5), g("swarmling", 28, 0.14, 12, 2.1).Via(EntB)}},
		{Groups: []Group{g("walker", 4, 3.5, 0, 2.1), g("beetle", 6, 1.6, 4, 1.8).Via(EntB), g("stinger", 10, 0.26, 6, 1.6), g("crawler", 16, 0.6, 12, 2.2).Via(EntB)}},
		{Groups: []Group{g("runner", 16, 0.33, 0, 2.2), g("splitter", 6, 1.5, 2, 2.1).Via(EntB), g("walker", 4, 3.2, 8, 2.3), g("drone", 3, 4, 10, 1.8).Via(EntB), g("stinger", 12, 0.25, 16, 1.7)}},
		{Groups: []Group{g("walker", 6, 2.8, 0, 2.4), g("beetle", 8, 1.4, 4, 2), g("drone", 4, 3.5, 6, 1.9).Via(EntB), g("swarmling", 36, 0.12, 10, 2.5).Via(EntB),
			g("splitter", 6, 1.5, 18, 2.2), g("stinger", 12, 0.24, 22, 1.8).Via(EntB)}},
	},
	Lesson: &Lesson{
		Turret:   "beacon",
		Problem:  "The fog keeps settling over the best pads, and the turrets standing in it barely reach the road.",
		Solution: "The Beacon never shoots, but its light clears the fog and gives the turrets around it extra range. Build it in the middle of a group of turrets.",
	},
}

// Ligeia Shore pads: 0 (4,1), 1 (9,2), 2 (17,2) by road B, 3 (13,4), 4 (2,5),
// 5 (9,6), 6 (13,6), 7 (9,8), 8 (13,8), 9 (19,8) at the gate, 10 (13,11).
// Pads 3, 6 and 8 stand between the two roads and cover both.
var LigeiaShore = &LevelSpec{
	ID: "3-3", Name: "Ligeia Shore", Biome: "titan",
	Desc: "The shore of a methane sea. Bloaters waddle in from the haze.",
	Tip:  "Finish Bloaters with Flamers: burnt down, they explode and hurt the bugs around them instead of leaving fog behind.",
	Map: []string{
		"...............2........",
		"....o..........#........",
		".........o.....#.o......",
		"S###########...#........",
		"...........#.o.#........",
		"..o........#...#....BBB.",
		".........o.#.o.#####BBB.",
		"...........#...#....BBB.",
		".........o.#.o.#...o....",
		"...........#####........",
		"........................",
		".............o..........",
		"........................",
	},
	StartCredits: 440,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    70,
	Fog: []FogBank{
		Bank(2.3, 13.5, 5.0, 9.5, 7.5),
		Bank(2.2, 17.0, 3.5, 13.5, 9.5),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1.1, 0, 1.3), g("bloater", 2, 4, 4, 1.2).Via(EntB)}},
		{Groups: []Group{g("bloater", 4, 3, 0, 1.3), g("runner", 10, 0.45, 4, 1.4).Via(EntB), g("stinger", 6, 0.3, 10, 1.3)}},
		{Groups: []Group{g("walker", 2, 5, 0, 1.5), g("bloater", 4, 2.8, 2, 1.4).Via(EntB), g("swarmling", 26, 0.15, 6, 1.7)}},
		{Groups: []Group{g("beetle", 5, 1.8, 0, 1.6), g("bloater", 5, 2.6, 2, 1.6).Via(EntB), g("drone", 2, 5, 6, 1.4), g("crawler", 12, 0.7, 10, 1.9).Via(EntB)}},
		{Groups: []Group{g("splitter", 5, 1.8, 0, 1.8), g("bloater", 6, 2.4, 2, 1.7).Via(EntB), g("stinger", 8, 0.28, 8, 1.5), g("runner", 12, 0.38, 12, 2)}},
		{Groups: []Group{g("walker", 3, 4, 0, 2.1).Via(EntB), g("bloater", 6, 2.2, 0, 1.9), g("runner", 12, 0.36, 6, 2.1), g("drone", 3, 4, 8, 1.7).Via(EntB), g("swarmling", 30, 0.13, 14, 2.2)}},
		{Groups: []Group{g("bloater", 8, 2, 0, 2.1), g("beetle", 6, 1.5, 2, 1.9).Via(EntB), g("stinger", 10, 0.25, 8, 1.7), g("walker", 4, 3.5, 12, 2.3).Via(EntB), g("crawler", 16, 0.55, 16, 2.3)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.4), g("bloater", 8, 1.8, 2, 2.2).Via(EntB), g("splitter", 6, 1.5, 6, 2.2), g("drone", 4, 3.5, 8, 1.9).Via(EntB),
			g("runner", 14, 0.32, 16, 2.3), g("stinger", 12, 0.24, 20, 1.8).Via(EntB)}},
	},
}

// Xanadu Ridge pads (mirrored pairs, A above / B below): 0 (10,2) / 10
// (10,10), 1 (13,2) / 11 (13,10), 2 (4,3) / 9 (4,9), 3 (17,4) / 7 (17,8),
// 4 (19,4) / 8 (19,8); 5 (9,6) and 6 (12,6) on the ridge cover both roads.
var XanaduRidge = &LevelSpec{
	ID: "3-4", Name: "Xanadu Ridge", Biome: "titan",
	Desc: "A ridge between two valleys, and the fog drifts over both of them.",
	Tip:  "Give each front its own Beacon, and keep the ridge pads in the middle lit: they cover both roads.",
	Map: []string{
		"........................",
		"S#######................",
		".......#..o..o..........",
		"....o..#................",
		".......#########.o.o....",
		"...............#....BBB.",
		".........o..o..#####BBB.",
		"...............#....BBB.",
		".......#########.o.o....",
		"....o..#................",
		".......#..o..o..........",
		"2#######................",
		"........................",
	},
	StartCredits: 450,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    70,
	Fog: []FogBank{
		Bank(2.2, 11.5, 2.5, 5.0, 2.5),
		Bank(2.2, 18.0, 4.0, 10.5, 5.5),
		Bank(2.2, 5.0, 10.5, 11.5, 10.5),
		Bank(2.2, 12.5, 7.0, 18.0, 9.0),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1, 0, 1.4), g("crawler", 10, 1, 0, 1.4).Via(EntB)}},
		{Groups: []Group{g("runner", 10, 0.45, 0, 1.5), g("bloater", 3, 3, 2, 1.4).Via(EntB), g("stinger", 6, 0.3, 10, 1.3)}},
		{Groups: []Group{g("beetle", 4, 2, 0, 1.6).Via(EntB), g("walker", 2, 5, 0, 1.6), g("swarmling", 24, 0.15, 6, 1.8).Via(EntB)}},
		{Groups: []Group{g("bloater", 5, 2.5, 0, 1.7), g("splitter", 5, 1.8, 2, 1.7).Via(EntB), g("drone", 2, 5, 8, 1.5), g("stinger", 8, 0.28, 10, 1.5).Via(EntB)}},
		{Groups: []Group{g("walker", 3, 4, 0, 1.9).Via(EntB), g("runner", 14, 0.36, 2, 2), g("beetle", 5, 1.7, 6, 1.8), g("drone", 2, 5, 10, 1.6).Via(EntB)}},
		{Groups: []Group{g("bloater", 6, 2.2, 0, 2).Via(EntB), g("crawler", 16, 0.55, 0, 2.2), g("splitter", 6, 1.6, 8, 2), g("stinger", 10, 0.26, 12, 1.7).Via(EntB)}},
		{Groups: []Group{g("walker", 4, 3.5, 0, 2.2), g("walker", 3, 3.5, 6, 2.2).Via(EntB), g("swarmling", 32, 0.13, 4, 2.3).Via(EntB), g("drone", 3, 4, 8, 1.8), g("bloater", 6, 2, 12, 2.1)}},
		{Groups: []Group{g("beetle", 8, 1.4, 0, 2.1).Via(EntB), g("runner", 16, 0.32, 0, 2.3), g("splitter", 6, 1.5, 6, 2.2).Via(EntB), g("stinger", 12, 0.24, 10, 1.9), g("drone", 3, 4, 14, 1.9).Via(EntB)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.5), g("bloater", 8, 1.8, 2, 2.3).Via(EntB), g("drone", 4, 3.5, 6, 2), g("swarmling", 36, 0.12, 10, 2.5).Via(EntB),
			g("splitter", 6, 1.4, 16, 2.3), g("stinger", 12, 0.24, 20, 1.9).Via(EntB), g("runner", 14, 0.3, 24, 2.4)}},
	},
	Lesson: &Lesson{
		Threat:   "Fog on two fronts",
		Icon:     "fog",
		Problem:  "Two roads in two valleys, and the fog drifts over both of them. A Beacon on one side can't help the other.",
		Solution: "Read the fog forecast of each valley before the wave. Give each front a Beacon, and hold the ridge in the middle: its pads cover both roads.",
	},
}

// Sotra Patera pads: 0 (3,1), 1 (10,2), 2 (14,2) by road B, 3 (15,4), 4 (3,5),
// 5 (8,5), 6 (7,7) inside the loop, 7 (15,8), 8 (18,8), 9 (7,11).
var SotraPatera = &LevelSpec{
	ID: "3-5", Name: "Sotra Patera", Biome: "titan",
	Desc: "The slopes of an ice volcano, where Stalkers hunt from the fog.",
	Tip:  "Stalkers in the fog can only be targeted from a Beacon's light. Flames and frost waves hit them anyway.",
	Map: []string{
		"............2...........",
		"...o........#...........",
		"..........o.#.o.........",
		"S#####......#...........",
		".....#......#..o........",
		"...o.#..o...#.......BBB.",
		".....#....##########BBB.",
		".....#.o..#.........BBB.",
		".....#....#....o..o.....",
		".....######.............",
		"........................",
		".......o................",
		"........................",
	},
	StartCredits: 450,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    70,
	Fog: []FogBank{
		Bank(2.4, 7.5, 7.5, 5.5, 3.5),
		Bank(2.4, 15.5, 6.5, 12.5, 3.0),
		Bank(2.0, 18.5, 7.5, 9.5, 5.0),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1, 0, 1.4), g("stalker", 4, 1.5, 6, 1.3).Via(EntB)}},
		{Groups: []Group{g("stalker", 6, 1.2, 0, 1.4), g("runner", 10, 0.45, 4, 1.5).Via(EntB), g("stinger", 6, 0.3, 10, 1.4)}},
		{Groups: []Group{g("walker", 3, 4.5, 0, 1.6), g("stalker", 6, 1.1, 2, 1.5).Via(EntB), g("bloater", 3, 3, 8, 1.5)}},
		{Groups: []Group{g("beetle", 5, 1.8, 0, 1.7).Via(EntB), g("stalker", 8, 1, 2, 1.6), g("drone", 2, 5, 6, 1.5), g("swarmling", 26, 0.14, 10, 1.9).Via(EntB)}},
		{Groups: []Group{g("splitter", 5, 1.8, 0, 1.9), g("stalker", 8, 0.9, 4, 1.8).Via(EntB), g("bloater", 4, 2.6, 6, 1.8), g("stinger", 8, 0.27, 12, 1.6).Via(EntB)}},
		{Groups: []Group{g("walker", 4, 3.5, 0, 2.1).Via(EntB), g("stalker", 10, 0.9, 0, 1.9), g("runner", 12, 0.36, 8, 2.2), g("drone", 3, 4, 10, 1.8).Via(EntB)}},
		{Groups: []Group{g("bloater", 6, 2.2, 0, 2.1), g("stalker", 10, 0.8, 4, 2).Via(EntB), g("beetle", 6, 1.5, 6, 2), g("stinger", 10, 0.25, 12, 1.8).Via(EntB), g("crawler", 16, 0.55, 16, 2.3)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.3), g("stalker", 12, 0.75, 2, 2.1).Via(EntB), g("splitter", 6, 1.5, 6, 2.2), g("drone", 4, 3.5, 8, 1.9).Via(EntB), g("swarmling", 32, 0.12, 14, 2.4)}},
		{Groups: []Group{g("stalker", 14, 0.7, 0, 2.3), g("walker", 5, 3, 2, 2.5).Via(EntB), g("bloater", 8, 1.8, 6, 2.3), g("drone", 4, 3.5, 8, 2).Via(EntB),
			g("runner", 14, 0.3, 16, 2.4), g("stinger", 12, 0.24, 20, 2).Via(EntB)}},
	},
}

// Selk Crater pads: 0 (3,2), 1 (8,2) by road B, 2 (16,4), 3 (8,5), 4 (10,5),
// 5 (3,7), 6 (9,7) between roads A and C, 7 (15,7) where the roads meet,
// 8 (19,8) at the gate, 9 (7,10), 10 (13,10), 11 (17,11).
var SelkCrater = &LevelSpec{
	ID: "3-6", Name: "Selk Crater", Biome: "titan",
	Desc: "A great impact crater: three burrows, drifting fog and every Titan bug at once.",
	Tip:  "Light up the killzones with Beacons, burn the Bloaters, and keep your strongest turrets where the roads meet before the gate.",
	Map: []string{
		"..........2.............",
		"..........#.............",
		"...o....o.#.............",
		"..........####..........",
		"S#####.......#..o.......",
		".....#..o.o..#......BBB.",
		".....###############BBB.",
		"...o.....o.....o#...BBB.",
		"................#..o....",
		".........########.......",
		".......o.#...o..........",
		".........#.......o......",
		".........3..............",
	},
	StartCredits: 470,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    75,
	Fog: []FogBank{
		Bank(2.3, 9.0, 5.5, 14.5, 8.5),
		Bank(2.2, 4.5, 4.0, 8.5, 9.5),
		Bank(2.2, 16.5, 5.0, 11.5, 3.5),
		Bank(2.0, 18.5, 8.5, 5.0, 7.0),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1, 0, 1.5), g("runner", 8, 0.5, 4, 1.5).Via(EntB), g("stalker", 4, 1.4, 8, 1.4).Via(EntC)}},
		{Groups: []Group{g("bloater", 4, 2.8, 0, 1.5).Via(EntB), g("swarmling", 24, 0.15, 2, 1.7).Via(EntC), g("stinger", 6, 0.3, 8, 1.4)}},
		{Groups: []Group{g("walker", 3, 4.5, 0, 1.7), g("beetle", 4, 2, 2, 1.6).Via(EntC), g("stalker", 6, 1.1, 6, 1.6).Via(EntB)}},
		{Groups: []Group{g("splitter", 5, 1.8, 0, 1.8).Via(EntB), g("bloater", 5, 2.5, 2, 1.7).Via(EntC), g("drone", 2, 5, 6, 1.6), g("runner", 12, 0.4, 10, 1.9)}},
		{Groups: []Group{g("walker", 3, 4, 0, 2).Via(EntC), g("stalker", 8, 0.9, 2, 1.8), g("beetle", 5, 1.7, 4, 1.9).Via(EntB), g("stinger", 8, 0.27, 10, 1.7).Via(EntC)}},
		{Groups: []Group{g("bloater", 6, 2.2, 0, 2), g("crawler", 16, 0.55, 2, 2.2).Via(EntB), g("splitter", 6, 1.6, 6, 2.1).Via(EntC), g("drone", 3, 4, 10, 1.8)}},
		{Groups: []Group{g("walker", 4, 3.5, 0, 2.3).Via(EntB), g("stalker", 10, 0.8, 2, 2).Via(EntC), g("swarmling", 32, 0.13, 6, 2.4), g("stinger", 10, 0.25, 12, 1.9).Via(EntB)}},
		{Groups: []Group{g("beetle", 8, 1.4, 0, 2.2).Via(EntC), g("bloater", 6, 2, 2, 2.2).Via(EntB), g("runner", 16, 0.32, 4, 2.4), g("drone", 4, 3.5, 10, 2), g("walker", 3, 3.5, 14, 2.4).Via(EntC)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.6), g("stalker", 12, 0.7, 2, 2.3).Via(EntB), g("bloater", 8, 1.8, 4, 2.4).Via(EntC), g("drone", 4, 3.5, 8, 2.1).Via(EntB),
			g("splitter", 6, 1.4, 14, 2.4), g("swarmling", 34, 0.12, 18, 2.6).Via(EntC), g("stinger", 12, 0.24, 22, 2.1)}},
	},
}

// Kraken Mare pads: 0 (4,0), 1 (15,3), 2 (19,3) by road B, 3 (9,4), 4 (5,5),
// 5 (12,5), 6 (9,7), 7 (12,7) inside the bay, 8 (16,8), 9 (19,8) at the
// gate, 10 (5,9), 11 (10,11).
var KrakenMare = &LevelSpec{
	ID: "3-7", Name: "Kraken Mare", Biome: "titan",
	Desc: "The Kraken rises from the methane sea to crush the colony.",
	Tip:  "Spread your turrets along the Kraken's road, so that it can never seize all of them, and let the Railguns fire from further away.",
	Map: []string{
		"....o............2......",
		".................#......",
		"S#######.........#......",
		".......#.......o.#.o....",
		".......#.o.......#......",
		".....o.#....o....#..BBB.",
		".......#......######BBB.",
		".......#.o..o.#.....BBB.",
		".......#......#.o..o....",
		".....o.########.........",
		"........................",
		"..........o.............",
		"........................",
	},
	StartCredits: 480,
	BaseHP:       100,
	FirstPrep:    35,
	Prep:         30,
	WaveBonus:    75,
	Fog: []FogBank{
		Bank(2.4, 10.5, 6.0, 16.5, 7.5),
		Bank(2.3, 5.5, 3.5, 11.0, 9.5),
		Bank(2.2, 18.0, 3.5, 5.5, 8.5),
	},
	Waves: []WaveSpec{
		{Groups: []Group{g("crawler", 10, 1, 0, 1.5), g("stalker", 6, 1.2, 4, 1.5).Via(EntB)}},
		{Groups: []Group{g("bloater", 5, 2.6, 0, 1.6), g("runner", 12, 0.42, 4, 1.7).Via(EntB), g("stinger", 6, 0.3, 10, 1.5)}},
		{Groups: []Group{g("walker", 3, 4.5, 0, 1.8), g("stalker", 8, 1, 2, 1.7).Via(EntB), g("swarmling", 26, 0.14, 8, 2)}},
		{Groups: []Group{g("beetle", 6, 1.7, 0, 1.9).Via(EntB), g("bloater", 6, 2.3, 2, 1.8), g("drone", 3, 4, 6, 1.7), g("stinger", 8, 0.27, 12, 1.6).Via(EntB)}},
		{Groups: []Group{g("splitter", 6, 1.7, 0, 2), g("stalker", 10, 0.9, 4, 1.9).Via(EntB), g("walker", 3, 4, 8, 2.2), g("crawler", 14, 0.6, 12, 2.3).Via(EntB)}},
		{Groups: []Group{g("bloater", 8, 2, 0, 2.1).Via(EntB), g("runner", 14, 0.34, 2, 2.3), g("drone", 3, 4, 8, 1.9), g("beetle", 6, 1.5, 10, 2.1)}},
		{Groups: []Group{g("walker", 5, 3, 0, 2.4), g("stalker", 12, 0.75, 4, 2.1).Via(EntB), g("splitter", 6, 1.5, 6, 2.3), g("stinger", 12, 0.25, 12, 2).Via(EntB)}},
		{Groups: []Group{g("beetle", 8, 1.4, 0, 2.3), g("bloater", 8, 1.8, 2, 2.3).Via(EntB), g("swarmling", 34, 0.12, 6, 2.6), g("drone", 4, 3.5, 10, 2.1).Via(EntB), g("walker", 4, 3.2, 14, 2.5)}},
		{Groups: []Group{g("crawler", 16, 0.55, 0, 2.4), g("kraken", 1, 1, 8, 1.6), g("stalker", 12, 0.7, 24, 2.3), g("drone", 3, 4, 20, 2.1),
			g("bloater", 6, 1.8, 4, 2.4).Via(EntB), g("walker", 4, 3, 16, 2.5).Via(EntB), g("stinger", 12, 0.25, 28, 2.1).Via(EntB), g("runner", 14, 0.3, 40, 2.4).Via(EntB)}},
	},
	Lesson: &Lesson{
		Threat:   "Kraken",
		Icon:     "kraken",
		Problem:  "The Kraken of the methane sea leads the last wave. Its tentacles seize the turrets nearest to it and shut them down for a few seconds.",
		Solution: "Spread your firepower along its road, so that it can never seize all of it at once. Railguns with a long view can keep firing from out of its reach.",
	},
}
