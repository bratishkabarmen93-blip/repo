package game

import "math"

// HealSpec makes an enemy repair the ground enemies around it in pulses.
type HealSpec struct {
	Radius float64 // cells
	Frac   float64 // fraction of the patient's max HP per pulse
	Cap    float64 // max HP restored per patient per pulse
	Period float64 // seconds between pulses
}

// BroodSpec makes an enemy lay other enemies while it walks.
type BroodSpec struct {
	Enemy  string
	Count  int
	Period float64
	HPMul  float64
	Verb   string // "lays" when empty
}

// SplitSpec makes an enemy burst into smaller ones when it is destroyed.
type SplitSpec struct {
	Enemy string
	Count int
	HPMul float64 // health of the parts, relative to the parent's wave multiplier
}

// EnemyDef describes an enemy type.
type EnemyDef struct {
	ID     string
	Name   string
	Plural string
	Desc   string  // one line for the "new enemy" card
	HP     float64 // base health, scaled per wave group
	Speed  float64 // cells per second
	Damage int     // damage dealt to the base on arrival
	Reward int     // credits for a kill
	Armor  float64 // flat damage reduction per hit
	Flying bool    // only anti-air weapons can target it
	Spread float64 // max sideways offset from the road centre (cells)
	Boss   bool
	Heal   *HealSpec
	Brood  *BroodSpec
	Split  *SplitSpec
	// NoChill: frost cannot slow or freeze it.
	NoChill bool
	// Fire scales the damage of flames and burning (0 means normal damage).
	Fire float64
	// Dive is the speed factor under the ice: between paired ice holes the
	// enemy swims below the surface, out of reach of every turret (0: it never dives).
	Dive float64
	// Spit is what it spits out ahead of itself when it comes up through a hole.
	Spit *BroodSpec
}

// FireMul is the factor flames and burning damage are scaled by.
func (d *EnemyDef) FireMul() float64 {
	if d.Fire == 0 {
		return 1
	}
	return d.Fire
}

// MinDamageFrac is the share of a hit that always gets through armour.
const MinDamageFrac = 0.2

// EffDamage is the damage a hit of dmg actually deals to this enemy.
func (d *EnemyDef) EffDamage(dmg float64) float64 {
	return math.Max(dmg-d.Armor, dmg*MinDamageFrac)
}

var (
	Crawler = &EnemyDef{
		ID: "crawler", Name: "Crawler", Plural: "crawlers", HP: 20, Speed: 1.1, Damage: 10, Reward: 5, Spread: 0.16,
		Desc: "Common hive bug. Weak alone, dangerous in numbers.",
	}
	Runner = &EnemyDef{
		ID: "runner", Name: "Runner", Plural: "runners", HP: 14, Speed: 2.2, Damage: 6, Reward: 4, Spread: 0.18,
		Desc: "Fast, fragile bugs that sprint past slow defences in packs.",
	}
	Swarmling = &EnemyDef{
		ID: "swarmling", Name: "Swarmling", Plural: "swarmlings", HP: 6, Speed: 1.5, Damage: 2, Reward: 1, Spread: 0.26,
		Desc: "Tiny, weak larvae. They come by the dozen and slip past busy turrets.",
	}
	Walker = &EnemyDef{
		ID: "walker", Name: "Walker", Plural: "walkers", HP: 90, Speed: 0.58, Damage: 30, Reward: 16, Armor: 6, Spread: 0.08,
		Desc: "Slow, heavy war robot. Weak shots barely scratch it: upgrade, or bring a Railgun.",
	}
	Drone = &EnemyDef{
		ID: "drone", Name: "Repair Drone", Plural: "repair drones", HP: 60, Speed: 0.75, Damage: 10, Reward: 14, Flying: true, Spread: 0.05,
		Heal: &HealSpec{Radius: 1.6, Frac: 0.08, Cap: 16, Period: 2},
		Desc: "A flying repair unit that patches up the enemies below it as it passes.",
	}
	Beetle = &EnemyDef{
		ID: "beetle", Name: "Frost Beetle", Plural: "frost beetles", HP: 70, Speed: 0.9, Damage: 15, Reward: 9, Spread: 0.12,
		NoChill: true, Fire: 2,
		Desc: "An ice-shelled bug of the frozen moon. Frost cannot slow it, but fire melts it fast.",
	}
	Splitter = &EnemyDef{
		ID: "splitter", Name: "Splitter", Plural: "splitters", HP: 55, Speed: 0.85, Damage: 12, Reward: 7, Spread: 0.12,
		Split: &SplitSpec{Enemy: "shard", Count: 4, HPMul: 1},
		Desc:  "A crystal bug that bursts into four quick Shards when destroyed. Kill it far from the base.",
	}
	// Shard is a splinter of a Splitter: it never comes out of a burrow.
	Shard = &EnemyDef{
		ID: "shard", Name: "Shard", Plural: "shards", HP: 10, Speed: 1.45, Damage: 3, Reward: 1, Spread: 0.28,
		Desc: "A splinter of a Splitter: small, quick and fragile.",
	}
	Stinger = &EnemyDef{
		ID: "stinger", Name: "Stinger", Plural: "stingers", HP: 12, Speed: 1.6, Damage: 3, Reward: 3, Flying: true, Spread: 0.2,
		Desc: "A hive wasp that hunts in fast packs. Splash missiles hit several at once.",
	}
	Queen = &EnemyDef{
		ID: "queen", Name: "Hive Queen", Plural: "hive queens", HP: 1500, Speed: 0.33, Damage: 50, Reward: 120, Armor: 4, Boss: true,
		Brood: &BroodSpec{Enemy: "swarmling", Count: 3, Period: 6, HPMul: 1.5},
		Desc:  "The mother of the swarm: huge, slow and very tough. Stop her before she reaches the base!",
	}
	// Wyrm is the boss of the ice moon: a giant serpent that swims under the
	// frozen lake between ice holes.
	Wyrm = &EnemyDef{
		ID: "wyrm", Name: "Ice Wyrm", Plural: "ice wyrms", HP: 2600, Speed: 0.32, Damage: 50, Reward: 150, Boss: true,
		NoChill: true, Fire: 1.5, Dive: 2.5,
		Brood: &BroodSpec{Enemy: "shard", Count: 2, Period: 5, HPMul: 1.5, Verb: "sheds"},
		Spit:  &BroodSpec{Enemy: "splitter", Count: 2, HPMul: 2},
		Desc:  "A giant serpent of the frozen moon. It dives under the ice, where no turret can reach it.",
	}
)

// EnemyOrder lists every enemy type in the order they are introduced.
var EnemyOrder = []*EnemyDef{Crawler, Runner, Swarmling, Walker, Drone, Queen, Beetle, Splitter, Stinger, Wyrm}

// Offspring are enemy types born from others (never sent by a wave).
var Offspring = []*EnemyDef{Shard}

var EnemyDefs = map[string]*EnemyDef{}

func init() {
	for _, d := range append(append([]*EnemyDef(nil), EnemyOrder...), Offspring...) {
		EnemyDefs[d.ID] = d
	}
}

// Group is a batch of identical enemies inside a wave.
type Group struct {
	Enemy    string
	Count    int
	Interval float64 // seconds between spawns
	Delay    float64 // seconds after the wave start
	HPMul    float64
	From     int // entrance index (0 is the first entrance)
}

// Via sends the group through another entrance.
func (g Group) Via(from int) Group { g.From = from; return g }

// entrance is the route the group takes on a level with n entrances.
func (g Group) entrance(n int) int {
	if g.From < 0 || g.From >= n {
		return 0
	}
	return g.From
}

type WaveSpec struct{ Groups []Group }

// Lesson introduces what a tutorial level unlocks as a problem and its solution.
type Lesson struct {
	Turret   string // turret type unlocked by the level ("" when it unlocks an ability)
	Ability  string // name of the unlocked ability, e.g. "Target priority"
	Threat   string // name of a new threat the level brings (a briefing without an unlock)
	Icon     string // HUD icon of the ability or threat
	Problem  string
	Solution string
}

// LevelSpec is the editable description of a level.
type LevelSpec struct {
	ID           string
	Name         string
	Desc         string
	Tip          string // advice after a defeat (the lesson's solution when empty)
	Map          []string
	StartCredits int
	BaseHP       int
	FirstPrep    float64 // seconds before the first wave
	Prep         float64 // seconds between waves
	WaveBonus    int
	Waves        []WaveSpec
	Turrets      []string // turret types that can be built (nil: all of them)
	NoTarget     bool     // target priority is still locked
	NoSuper      bool     // super abilities are still locked
	Lesson       *Lesson
	Biome        string // "" for the red desert of chapter 1, "ice" for the frozen moon
	Tier         Tier   // threat: sets the crystals its stars pay (colony.go)
}

// Allows reports whether turrets of type d can be built on this level.
func (s *LevelSpec) Allows(d *TowerDef) bool {
	if s.Turrets == nil {
		return true
	}
	for _, id := range s.Turrets {
		if id == d.ID {
			return true
		}
	}
	return false
}

// Arsenal lists the turret types of the level in build menu order.
func (s *LevelSpec) Arsenal() []*TowerDef {
	var out []*TowerDef
	for _, d := range TowerOrder {
		if s.Allows(d) {
			out = append(out, d)
		}
	}
	return out
}

// Enemies lists the enemy types of the level in order of appearance.
func (s *LevelSpec) Enemies() []*EnemyDef {
	var out []*EnemyDef
	seen := map[*EnemyDef]bool{}
	for _, wv := range s.Waves {
		for _, en := range wv.Summary() {
			if !seen[en.Def] {
				seen[en.Def] = true
				out = append(out, en.Def)
			}
		}
	}
	return out
}

func g(enemy string, count int, interval, delay, hpMul float64) Group {
	return Group{Enemy: enemy, Count: count, Interval: interval, Delay: delay, HPMul: hpMul}
}
