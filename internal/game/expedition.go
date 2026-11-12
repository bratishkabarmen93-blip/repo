package game

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strings"
)

// ---- Expeditions ----
//
// An expedition is a level generated from a seed out of what the player has
// met in the campaign: its turrets and abilities, its enemies, its worlds and
// its kinds of maps. Presets (map layouts, attack themes, a boss or none)
// combine at random; the threat tier sets both the pressure of the waves and
// the crystals a star pays. The waves are measured against the last levels
// the player has won, so an expedition stays fair at every point of the
// campaign.

// ExpeditionID is the level ID every expedition plays under.
const ExpeditionID = "exp"

// ExpeditionPay is what one star of an expedition pays: more than a replay of
// a campaign level of the same threat, less than its first win.
func ExpeditionPay(t Tier) int {
	switch t {
	case TierMedium:
		return 9
	case TierHigh:
		return 12
	}
	return 6
}

// expTune holds the balance knobs of expeditions, tuned with the test bots
// (EXPBAL=1 go test -run TestExpeditionBalance -v).
var expTune = struct {
	Scale      float64            // all pressure (the calibration test moves it)
	Tier       [4]float64         // pressure per threat tier
	Layout     map[string]float64 // pressure per kind of map
	Theme      map[string]float64 // pressure per kind of attack
	LenExp     float64            // how much a longer road raises the pressure
	Rise       float64            // the last wave of 6 against the first
	Boss       float64            // boss hit points, in last waves
	BossChance float64            // how often a known boss leads the last wave
	AirCap     float64            // a flock that cannot grow any more: its bugs at most this much above the usual toughness
}{
	Scale:      1,
	Tier:       [4]float64{1, 0.55, 1.0, 1.38},
	Layout:     map[string]float64{"single": 1, "twin": 1, "fronts": 0.8, "triple": 1, "siege": 1},
	Theme:      map[string]float64{"Mixed": 1, "Swarm": 1, "Armored": 1, "Rush": 0.9, "Air raid": 1, "Ambush": 1},
	LenExp:     0.7,
	Rise:       9,
	Boss:       0.35,
	BossChance: 0.3,
	AirCap:     1.5,
}

// expFactor is a tuning factor, 1 when the table has none.
func expFactor(m map[string]float64, k string) float64 {
	if v, ok := m[k]; ok {
		return v
	}
	return 1
}

// expPressure is the threat of the last wave of a Medium expedition on an
// average map, after each campaign level (calibrated with the test bots:
// EXPCAL=1 go test -run TestExpeditionCalibrate -v).
var expPressure = []float64{3700, 2900, 3300, 3200, 3100, 2600, 2050, 2150, 3500, 2650, 2700, 1950, 1600, 4000,
	3500, 3050, 3250, 3300, 3350, 3550, 3250}

func expPressureAt(stage int) float64 {
	return expPressure[min(max(stage, 1), len(expPressure))-1]
}

// ---- What the player knows ----

// Knowledge is what the player has met in the campaign so far.
type Knowledge struct {
	Turrets   []*TowerDef // in build menu order
	Enemies   []*EnemyDef // regular enemies, in order of appearance
	Bosses    []*EnemyDef
	Target    bool         // target priority is known
	Super     bool         // super upgrades are known
	Entrances int          // the most burrows of a won map
	Fronts    bool         // a won map attacked the base from both sides
	Europa    bool         // the second world has been reached
	Titan     bool         // the third world's fog has been met (its first level won)
	Recent    []*LevelSpec // the last campaign levels won, newest first (up to 3)
	Stage     int          // campaign position of the newest (1 is the first level)
}

// Learn gathers what the player knows from the levels won so far.
func Learn(won func(*LevelSpec) bool) Knowledge {
	k := Knowledge{Entrances: 1}
	turrets := map[*TowerDef]bool{}
	enemies := map[*EnemyDef]bool{}
	pos := 0
	for _, s := range AllLevels() {
		if s != Skirmish {
			pos++
		}
		if !won(s) {
			continue
		}
		for _, d := range s.Arsenal() {
			turrets[d] = true
		}
		k.Target = k.Target || !s.NoTarget
		k.Super = k.Super || !s.NoSuper
		for _, d := range s.Enemies() {
			enemies[d] = true
		}
		if l, err := ParseLevel(s); err == nil {
			k.Entrances = max(k.Entrances, l.Entrances())
			k.Fronts = k.Fronts || l.EastGate()
		}
		if s != Skirmish {
			k.Recent = append([]*LevelSpec{s}, k.Recent...)[:min(3, len(k.Recent)+1)]
			k.Stage = pos
		}
	}
	for _, d := range TowerOrder {
		if turrets[d] {
			k.Turrets = append(k.Turrets, d)
		}
	}
	for _, d := range EnemyOrder {
		switch {
		case !enemies[d]:
		case d.Boss:
			k.Bosses = append(k.Bosses, d)
		default:
			k.Enemies = append(k.Enemies, d)
		}
	}
	ch1 := Chapters[0].Levels
	k.Europa = won(ch1[len(ch1)-1])
	k.Titan = won(HuygensLanding)
	return k
}

// Key identifies the knowledge: the expedition of a seed only changes with it.
func (k Knowledge) Key() string {
	var b strings.Builder
	for _, d := range k.Turrets {
		b.WriteString(d.ID + ",")
	}
	for _, d := range k.Enemies {
		b.WriteString(d.ID + ",")
	}
	for _, d := range k.Bosses {
		b.WriteString(d.ID + ",")
	}
	fmt.Fprintf(&b, "%v,%v,%d,%v,%v,%v,%d", k.Target, k.Super, k.Entrances, k.Fronts, k.Europa, k.Titan, k.Stage)
	return b.String()
}

func (k Knowledge) knows(d *EnemyDef) bool {
	return slices.Contains(k.Enemies, d) || slices.Contains(k.Bosses, d)
}

func (k Knowledge) hasTurret(d *TowerDef) bool { return slices.Contains(k.Turrets, d) }

// ---- Presets ----

// expWorld is a world an expedition can land on.
type expWorld struct {
	Name, Biome string
	W           int
	Boss        *EnemyDef
	Native      []*EnemyDef // nil: every known enemy
	Adj, Noun   []string    // the parts of site names
	Fog         bool        // banks of methane fog drift over its maps
}

func expWorlds(k Knowledge) []*expWorld {
	ws := []*expWorld{{
		Name: "Mars", W: 24, Boss: Queen,
		Native: []*EnemyDef{Crawler, Runner, Swarmling, Walker, Drone},
		Adj:    []string{"Rust", "Crimson", "Dust", "Iron", "Ochre", "Ember", "Red", "Scorched"},
		Noun:   []string{"Ridge", "Basin", "Gorge", "Mesa", "Flats", "Rift", "Crater", "Dunes"},
	}}
	if k.Europa {
		ws = append(ws, &expWorld{
			Name: "Europa", Biome: "ice", W: 25, Boss: Wyrm,
			// The bugs of Mars and Europa: Titan's own stay on Titan.
			Native: []*EnemyDef{Crawler, Runner, Swarmling, Walker, Drone, Beetle, Splitter, Stinger},
			Adj:    []string{"Frost", "Glass", "Pale", "Rime", "Silent", "Blue", "Frozen", "Hollow"},
			Noun:   []string{"Shelf", "Drift", "Crevasse", "Lake", "Fields", "Floe", "Ridge", "Basin"},
		})
	}
	if k.Titan {
		ws = append(ws, &expWorld{
			Name: "Titan", Biome: "titan", W: 24, Boss: Kraken, Fog: true,
			Adj:  []string{"Amber", "Hazy", "Tholin", "Murky", "Dusky", "Misty", "Methane", "Umber"},
			Noun: []string{"Mare", "Dunes", "Shore", "Patera", "Lacus", "Crater", "Plain", "Ridge"},
		})
	}
	return ws
}

// expLayout is a kind of map.
type expLayout struct {
	ID, Name, Desc string
	West, East     int // roads west of the base (they merge) and east of it
}

var expLayouts = []expLayout{
	{"single", "Long road", "One long road winds to the outpost.", 1, 0},
	{"twin", "Twin burrows", "Two burrows: their roads merge on the way to the outpost.", 2, 0},
	{"fronts", "Two fronts", "The outpost is caught in the middle: they come from both sides.", 1, 1},
	{"triple", "Triple breach", "Three burrows feed one road to the outpost.", 3, 0},
	{"siege", "Siege", "Two burrows on one side of the outpost, a third on the other.", 2, 1},
}

func (k Knowledge) layouts() []expLayout {
	var out []expLayout
	for _, l := range expLayouts {
		if l.West+l.East > k.Entrances || (l.East > 0 && !k.Fronts) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// expTheme is a kind of attack: it weighs the enemies the waves pick.
type expTheme struct {
	Name, Desc string
	W          map[string]float64 // weight per enemy ID (missing: 1)
	Key        []string           // it needs one of these enemies
}

func expThemes() []expTheme {
	return []expTheme{
		{"Mixed", "Expect a bit of everything.", nil, nil},
		{"Swarm", "Swarms: masses of small, quick bugs.",
			map[string]float64{"swarmling": 5, "runner": 2.5, "stinger": 1.5, "splitter": 2, "walker": 0.4, "drone": 0.5, "beetle": 0.6, "bloater": 0.5},
			[]string{"swarmling", "stinger"}},
		{"Armored", "Heavy columns that soak up fire.",
			map[string]float64{"walker": 4, "beetle": 3.5, "splitter": 2, "bloater": 2.5, "runner": 0.5, "swarmling": 0.5, "stinger": 0.5},
			[]string{"walker", "beetle"}},
		{"Rush", "Fast bugs rush the road.",
			map[string]float64{"runner": 5, "stinger": 1.5, "swarmling": 2, "stalker": 2, "walker": 0.4, "drone": 0.5, "beetle": 0.6, "bloater": 0.4},
			[]string{"runner"}},
		{"Air raid", "Much of the attack comes by air.",
			map[string]float64{"drone": 4, "stinger": 4, "walker": 0.7, "beetle": 0.7, "bloater": 0.7},
			[]string{"drone", "stinger"}},
		{"Ambush", "Stalkers hunt from the fog.",
			map[string]float64{"stalker": 5, "bloater": 2, "runner": 1.5, "swarmling": 0.7, "walker": 0.6, "drone": 0.6, "beetle": 0.6},
			[]string{"stalker"}},
	}
}

func (th expTheme) fits(pool []*EnemyDef) bool {
	if th.Key == nil {
		return true
	}
	for _, d := range pool {
		if slices.Contains(th.Key, d.ID) {
			return true
		}
	}
	return false
}

// expKind is how an enemy type comes in waves.
type expKind struct {
	Cnt   [2]int     // group size, early and late waves
	Gap   [2]float64 // seconds between two of them (packs as tight as in the campaign: splash hits several)
	From  int        // the first wave it may come in (0-based)
	W     float64    // how hard a hit point of it is to stop
	Tough float64    // its hit point multiplier against the others'
	// Floor: its hit points at least this share of its usual toughness; a thin
	// budget sends fewer of them rather than weaker ones (0: the common floor).
	Floor float64
}

var expKinds = map[string]expKind{
	"crawler":   {[2]int{6, 14}, [2]float64{0.6, 0.9}, 0, 1, 1, 0},
	"runner":    {[2]int{6, 14}, [2]float64{0.35, 0.5}, 0, 1.25, 1, 0},
	"swarmling": {[2]int{16, 36}, [2]float64{0.12, 0.19}, 0, 0.9, 1, 0},
	"walker":    {[2]int{2, 6}, [2]float64{2.2, 3.2}, 1, 1.5, 0.9, 0.85},
	"drone":     {[2]int{1, 4}, [2]float64{3, 4.5}, 2, 1.6, 0.75, 0},
	"beetle":    {[2]int{4, 10}, [2]float64{1.1, 1.5}, 1, 1.2, 0.75, 0},
	"splitter":  {[2]int{3, 8}, [2]float64{1.3, 1.8}, 1, 1.5, 0.9, 0},
	"stinger":   {[2]int{6, 14}, [2]float64{0.25, 0.3}, 2, 1.4, 1, 0},
	"bloater":   {[2]int{3, 8}, [2]float64{1.8, 3}, 1, 1.2, 0.9, 0},
	"stalker":   {[2]int{4, 12}, [2]float64{0.7, 1.4}, 1, 1.5, 1, 0},
}

func expWeight(d *EnemyDef) float64 {
	if k, ok := expKinds[d.ID]; ok {
		return k.W
	}
	return 1
}

// expBudget is the threat of the regular enemies of a wave: their hit points
// weighed by how hard they are to stop.
func expBudget(wv WaveSpec) float64 {
	b := 0.0
	for _, g := range wv.Groups {
		if d := EnemyDefs[g.Enemy]; d != nil && !d.Boss {
			b += float64(g.Count) * float64(d.HP) * g.HPMul * expWeight(d)
		}
	}
	return b
}

// expTough is the typical hit point multiplier of a wave's regular enemies.
func expTough(wv WaveSpec) float64 {
	sum, w := 0.0, 0.0
	for _, g := range wv.Groups {
		if d := EnemyDefs[g.Enemy]; d != nil && !d.Boss {
			b := float64(g.Count) * float64(d.HP) * expWeight(d)
			sum, w = sum+g.HPMul*b, w+b
		}
	}
	if w == 0 {
		return 1
	}
	return sum / w
}

// expAlong reads a measure of a level's waves at t in [0, 1].
func expAlong(s *LevelSpec, t float64, f func(WaveSpec) float64) float64 {
	n := len(s.Waves)
	if n == 1 {
		return f(s.Waves[0])
	}
	p := t * float64(n-1)
	i := min(int(p), n-2)
	x := p - float64(i)
	return f(s.Waves[i])*(1-x) + f(s.Waves[i+1])*x
}

// expEconomy takes the credits of an expedition from the last levels won: the
// median of what they gave per entrance, times its entrances.
func expEconomy(refs []*LevelSpec, ents int, split bool) (credits, bonus int, first, prep float64, baseHP int) {
	var cs, bs []float64
	for _, s := range refs {
		_, n := expRoadLen(s)
		f := 1 + 0.12*float64(n-1)
		cs = append(cs, float64(s.StartCredits)/f)
		bs = append(bs, float64(s.WaveBonus)/f)
		first += s.FirstPrep / float64(len(refs))
		prep += s.Prep / float64(len(refs))
		baseHP = max(baseHP, s.BaseHP)
	}
	f := 1 + 0.12*float64(ents-1)
	if split {
		// The roads never meet: the defence is split, it needs more to build.
		f *= 1.3
	}
	credits = int(math.Round(expMedian(cs)*f/10)) * 10
	bonus = int(math.Round(expMedian(bs)*f/5)) * 5
	return
}

func expMedian(v []float64) float64 {
	s := slices.Clone(v)
	slices.Sort(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

// expRoadLen is the mean road length of a map and its number of entrances.
func expRoadLen(s *LevelSpec) (float64, int) {
	l, err := ParseLevel(s)
	if err != nil || len(l.Routes) == 0 {
		return 40, 1
	}
	sum := 0
	for _, r := range l.Routes {
		sum += len(r.Cells)
	}
	return float64(sum) / float64(len(l.Routes)), len(l.Routes)
}

// ---- The expedition ----

// Expedition is a generated level and what it was made of.
type Expedition struct {
	Seed   int64
	World  string    // the world it lands on
	Layout string    // the kind of map
	Theme  string    // the kind of attack
	Boss   *EnemyDef // nil: no boss this time
	Spec   *LevelSpec
}

// NewExpedition builds the expedition of a seed from what the player knows.
// The same seed and knowledge always give the same expedition.
func NewExpedition(seed int64, k Knowledge) *Expedition { return newExpedition(seed, k, 0) }

func expRollTier(r *rand.Rand) Tier {
	switch x := r.Float64(); {
	case x < 0.35:
		return TierLow
	case x < 0.75:
		return TierMedium
	}
	return TierHigh
}

// expPool lists the enemies an expedition on a world may send.
func expPool(k Knowledge, w *expWorld) []*EnemyDef {
	var out []*EnemyDef
	for _, d := range k.Enemies {
		if _, ok := expKinds[d.ID]; !ok || (w.Native != nil && !slices.Contains(w.Native, d)) {
			continue
		}
		if d.Flying && !k.hasTurret(Skyguard) {
			continue
		}
		if d.Stealth && !k.hasTurret(Beacon) { // in the fog only a Beacon's light shows them
			continue
		}
		out = append(out, d)
	}
	if len(out) == 0 {
		out = []*EnemyDef{Crawler}
	}
	return out
}

// newExpedition builds an expedition; a tier of 0 rolls one.
func newExpedition(seed int64, k Knowledge, tier Tier) *Expedition {
	r := rand.New(rand.NewSource(seed))
	worlds := expWorlds(k)
	w := worlds[r.Intn(len(worlds))]
	if tier == 0 {
		tier = expRollTier(r)
	}
	e := &Expedition{Seed: seed, World: w.Name}
	if w.Boss != nil && k.knows(w.Boss) && r.Float64() < expTune.BossChance {
		e.Boss = w.Boss
	}
	pool := expPool(k, w)
	var themes []expTheme
	for _, th := range expThemes() {
		if th.fits(pool) {
			themes = append(themes, th)
		}
	}
	th := themes[r.Intn(len(themes))]
	layouts := k.layouts()
	lay := layouts[r.Intn(len(layouts))]
	site := w.Adj[r.Intn(len(w.Adj))] + " " + w.Noun[r.Intn(len(w.Noun))]

	// The map: generated and checked like a handmade one.
	var rows []string
	roadLen, ents := 0.0, 0
	for try := 0; try < 8 && rows == nil; try++ {
		if try == 6 { // never expected: the plainest map, without the Wyrm
			lay = expLayouts[0]
			if e.Boss == Wyrm {
				e.Boss = nil
			}
		}
		m := expGenMap(r, w.W, 13, lay, e.Boss == Wyrm)
		if m == nil {
			continue
		}
		if _, err := ParseLevel(&LevelSpec{ID: ExpeditionID, Map: m.rows()}); err != nil {
			continue
		}
		rows = m.rows()
		for _, rt := range m.routes {
			roadLen += float64(len(rt.cells))
		}
		ents = len(m.routes)
		roadLen /= float64(ents)
	}
	if rows == nil {
		lay, e.Boss = expLayouts[0], nil
		rows = slices.Clone(IronPass.Map)
		roadLen, ents = expRoadLen(IronPass)
	}
	e.Layout, e.Theme = lay.Name, th.Name
	// Titan: banks of methane fog over the pads that watch the roads.
	var fog []FogBank
	if w.Fog {
		fog = expFog(r, rows, 2+(ents+r.Intn(2))/2)
	}

	// The pressure is calibrated by campaign stage and scaled by threat and
	// map; the economy and the toughness of single bugs follow the last levels
	// won.
	refs := k.Recent
	if len(refs) == 0 {
		refs = []*LevelSpec{Chapters[0].Levels[0]}
	}
	credits, bonus, first, prep, baseHP := expEconomy(refs, ents, lay.East > 0)
	tough := func(t float64) float64 {
		var v []float64
		for _, s := range refs {
			v = append(v, expAlong(s, t, expTough))
		}
		return expMedian(v)
	}
	n := 5 + int(tier) // Low 6 waves, Medium 7, High 8
	rise := expTune.Rise * (1 + 0.15*float64(n-6))
	final := expPressureAt(k.Stage) * expTune.Scale * expTune.Tier[tier] *
		expFactor(expTune.Layout, lay.ID) * expFactor(expTune.Theme, th.Name) *
		expClamp(math.Pow(roadLen/40, expTune.LenExp), 0.75, 1.3)
	budget := func(t float64) float64 { return final * math.Pow(rise, t-1) }
	waves := expWaves(r, pool, th, e.Boss, n, ents, tough, budget)

	turrets := []string{}
	for _, d := range k.Turrets {
		turrets = append(turrets, d.ID)
	}
	if len(turrets) == 0 {
		turrets = []string{Blaster.ID}
	}
	desc := lay.Desc + " " + th.Desc
	if len(fog) > 0 {
		desc = lay.Desc + " Banks of methane fog drift over the roads. " + th.Desc
	}
	switch e.Boss {
	case Queen:
		desc += " The Hive Queen leads the last wave."
	case Wyrm:
		desc += " The Ice Wyrm hunts under the lakes in the last wave."
	case Kraken:
		desc += " The Kraken crawls out of the methane sea with the last wave."
	}
	e.Spec = &LevelSpec{
		ID: ExpeditionID, Name: site, Desc: desc, Tip: expTip(th, k, e.Boss, len(fog) > 0),
		Map:          rows,
		Fog:          fog,
		StartCredits: credits,
		BaseHP:       max(baseHP, 100),
		FirstPrep:    math.Round(first),
		Prep:         math.Round(prep),
		WaveBonus:    bonus,
		Waves:        waves,
		Turrets:      turrets,
		NoTarget:     !k.Target,
		NoSuper:      !k.Super,
		Biome:        w.Biome,
		Tier:         tier,
	}
	return e
}

func expTip(th expTheme, k Knowledge, boss *EnemyDef, fog bool) string {
	const forecast = "Before every wave, dashed outlines show where the fog will drift. Build where it won't settle"
	switch {
	case boss == Kraken:
		return "Spread your turrets along the Kraken's road, so that it can never seize all of them, and let the Railguns fire from further away."
	case boss == Wyrm:
		return "Hit the Wyrm where the road runs on solid ground: under the lakes it is out of reach."
	case boss == Queen:
		return "Keep firepower for the Hive Queen: she spawns swarmlings as she walks."
	case th.Name == "Ambush" && k.hasTurret(Beacon):
		return "Stalkers in the fog can only be targeted from a Beacon's light. Flames and frost waves hit them anyway."
	case fog && th.Name == "Armored" && k.knows(Bloater) && k.hasTurret(Flamer):
		return "Finish Bloaters with Flamers: burnt down, they explode and hurt the bugs around them instead of leaving fog behind."
	case fog && !k.hasTurret(Beacon):
		return forecast + "."
	case th.Name == "Swarm" && k.hasTurret(Flamer):
		return "Flamers burn whole swarms at once."
	case th.Name == "Armored" && k.hasTurret(Railgun):
		return "Railguns pierce armor."
	case th.Name == "Rush" && k.hasTurret(Cryo):
		return "Cryo slows the runners down for your other turrets."
	case th.Name == "Air raid" && k.hasTurret(Skyguard):
		return "Skyguards bring down what flies."
	case fog:
		return forecast + ", or clear it with a Beacon's light."
	}
	return "Spend early on the busiest stretch of road, then upgrade."
}

// expFog lays n banks of methane fog over a map (its rows): each one rests
// over a pad that watches the road and drifts to another one, 4 to 9 cells
// away, before the next wave, and back. The base stays clear.
func expFog(r *rand.Rand, rows []string, n int) []FogBank {
	var pads, road []Vec
	var base Vec
	nb := 0
	for y, row := range rows {
		for x, ch := range row {
			c := Vec{float64(x) + 0.5, float64(y) + 0.5}
			switch ch {
			case 'o':
				pads = append(pads, c)
			case 'B':
				base, nb = base.Add(c), nb+1
			case '.':
			default: // the road and its burrows
				road = append(road, c)
			}
		}
	}
	if nb > 0 {
		base = base.Mul(1 / float64(nb))
	}
	// A pad weighs as much as the road it watches: the fog matters where
	// the turrets are busy.
	var spots []Vec
	var ws []float64
	for _, p := range pads {
		if p.Dist(base) < 4 {
			continue
		}
		w := 0.0
		for _, c := range road {
			if c.Dist2(p) <= 2.5*2.5 {
				w++
			}
		}
		if w > 0 {
			spots, ws = append(spots, p), append(ws, w)
		}
	}
	pick := func(ok func(Vec) bool) (Vec, bool) {
		tot := 0.0
		for i, p := range spots {
			if ok(p) {
				tot += ws[i]
			}
		}
		if tot == 0 {
			return Vec{}, false
		}
		x := r.Float64() * tot
		var last Vec
		for i, p := range spots {
			if !ok(p) {
				continue
			}
			if last = p; x < ws[i] {
				break
			}
			x -= ws[i]
		}
		return last, true
	}
	jitter := func(p Vec) Vec { return Vec{p.X + 0.8*(r.Float64()-0.5), p.Y + 0.8*(r.Float64()-0.5)} }
	var banks []FogBank
	for try := 0; len(banks) < n && try < 12; try++ {
		a, ok := pick(func(p Vec) bool {
			for _, b := range banks {
				if b.Stops[0].Dist(p) < 4 {
					return false
				}
			}
			return true
		})
		if !ok {
			break
		}
		b, ok := pick(func(p Vec) bool {
			if d := p.Dist(a); d < 4 || d > 9 {
				return false
			}
			for _, b := range banks {
				if b.Stops[1].Dist(p) < 4 {
					return false
				}
			}
			return true
		})
		if !ok {
			continue
		}
		banks = append(banks, FogBank{R: 2 + 0.1*float64(r.Intn(5)), Stops: []Vec{jitter(a), jitter(b)}})
	}
	return banks
}

// expPick picks the enemy type of a group.
func expPick(r *rand.Rand, pool []*EnemyDef, th expTheme, wave int) *EnemyDef {
	ws := make([]float64, len(pool))
	tot := 0.0
	for i, d := range pool {
		w := 1.0
		if v, ok := th.W[d.ID]; ok {
			w = v
		}
		if wave < expKinds[d.ID].From {
			w = 0
		}
		ws[i] = w
		tot += w
	}
	if tot == 0 {
		return pool[0]
	}
	x := r.Float64() * tot
	for i, w := range ws {
		if x < w {
			return pool[i]
		}
		x -= w
	}
	return pool[len(pool)-1]
}

func expLerp(a, b, t float64) float64 { return a + (b-a)*t }

func expClamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func expAbs(v int) int { return max(v, -v) }

func expRound(v, k float64) float64 { return math.Round(v*k) / k }

// expWaves fills n waves, each worth the threat budget(t) at t in [0, 1], with
// bugs about as tough as tough(t) says.
func expWaves(r *rand.Rand, pool []*EnemyDef, th expTheme, boss *EnemyDef, n, ents int, tough, budget func(float64) float64) []WaveSpec {
	waves := make([]WaveSpec, 0, n)
	used := make([]bool, ents)
	for j := 0; j < n; j++ {
		t := float64(j) / float64(n-1)
		b := budget(t)
		last := j == n-1
		if last && boss != nil {
			b *= 0.6
		}
		ng := 1 + int(t*2.4+r.Float64()*0.9)
		kinds := make([]*EnemyDef, ng)
		shares := make([]float64, ng)
		sum := 0.0
		for i := range kinds {
			kinds[i] = expPick(r, pool, th, j)
			shares[i] = 0.6 + 0.8*r.Float64()
			sum += shares[i]
		}
		// The roads wake up one by one, every other wave, and scouts come
		// first down each: the first wave is crawlers on the main road.
		if j == 0 {
			kinds, shares, sum = kinds[:1], shares[:1], shares[0]
		}
		open := min(ents, 1+j/2)
		start := r.Intn(open)
		from := make([]int, len(kinds))
		for i := range kinds {
			from[i] = (start + i) % open
			if !used[from[i]] && slices.Contains(pool, Crawler) {
				kinds[i] = Crawler
			}
			used[from[i]] = true
		}
		// Only Skyguards stop flyers: they carry a part of a wave, not all.
		air, capAir := 0.0, 0.35
		if th.Name == "Air raid" {
			capAir = 0.5
		}
		for i, d := range kinds {
			if d.Flying {
				air += shares[i]
			}
		}
		if air > capAir*sum {
			if air == sum {
				b *= capAir
			} else {
				f := capAir * (sum - air) / ((1 - capAir) * air)
				for i, d := range kinds {
					if d.Flying {
						shares[i] *= f
					}
				}
				sum = sum - air + air*f
			}
		}
		var wv WaveSpec
		delay := 0.0
		for i, d := range kinds {
			gb := b * shares[i] / sum
			kd := expKinds[d.ID]
			unit := float64(d.HP) * kd.W
			// A usual group size, its bugs as tough as the budget makes them,
			// up to a bit above the campaign's at this point: past that the
			// group grows, within reason.
			cnt := expLerp(float64(kd.Cnt[0]), float64(kd.Cnt[1]), 0.25+0.75*t) * (0.8 + 0.4*r.Float64())
			hmax := math.Max(1.2, 1.25*tough(t)*kd.Tough)
			hmin, minCnt := 0.6, math.Max(1, float64(kd.Cnt[0])/2)
			if kd.Floor > 0 {
				hmin, minCnt = math.Max(hmin, kd.Floor*tough(t)*kd.Tough), 1
			}
			switch hm := gb / (cnt * unit); {
			case hm > hmax:
				cnt = math.Min(gb/(unit*hmax), float64(kd.Cnt[1])*1.6)
			case hm < hmin:
				cnt = math.Max(gb/(unit*hmin), minCnt)
			}
			c := max(1, int(math.Round(cnt)))
			hm := math.Max(0.5, gb/(float64(c)*unit))
			if d.Flying {
				// Only Skyguards answer a flock: one that cannot grow any more
				// keeps its bugs within reason, the rest of its share is dropped.
				hm = math.Min(hm, expTune.AirCap*hmax)
			}
			gap := expLerp(kd.Gap[0], kd.Gap[1], r.Float64())
			wv.Groups = append(wv.Groups, Group{
				Enemy: d.ID, Count: c, Interval: expRound(gap, 100), Delay: expRound(delay, 10),
				HPMul: expRound(hm, 100), From: from[i],
			})
			delay += float64(c)*gap*(0.35+0.4*r.Float64()) + 2 + 3*r.Float64()
		}
		if last && boss != nil {
			bh := math.Max(0.5, budget(1)*expTune.Boss/float64(boss.HP))
			wv.Groups = append(wv.Groups, Group{
				Enemy: boss.ID, Count: 1, Interval: 1, Delay: expRound(6+6*r.Float64(), 10),
				HPMul: expRound(bh, 100), From: r.Intn(ents),
			})
		}
		waves = append(waves, wv)
	}
	return waves
}

// ---- Maps ----
//
// Roads run on an uneven lattice: its lines are at least 3 cells apart, so two
// stretches of road never touch unless they follow each other, and merging
// roads meet only at lattice nodes. A road grows from the base gate out to an
// edge of the map; more roads either branch off it (they merge on the way
// in) or leave the base by its east gate.

type expNode struct{ i, j int }

type expLattice struct {
	xs, ys []int
	west   bool // west of the base: its roads leave by the west edge, not the east
}

func (lt *expLattice) at(n expNode) Cell { return Cell{lt.xs[n.i], lt.ys[n.j]} }

func (lt *expLattice) node(c Cell) (expNode, bool) {
	i, j := slices.Index(lt.xs, c.X), slices.Index(lt.ys, c.Y)
	return expNode{i, j}, i >= 0 && j >= 0
}

func (lt *expLattice) neighbors(n expNode) []expNode {
	var out []expNode
	if n.i > 0 {
		out = append(out, expNode{n.i - 1, n.j})
	}
	if n.i < len(lt.xs)-1 {
		out = append(out, expNode{n.i + 1, n.j})
	}
	if n.j > 0 {
		out = append(out, expNode{n.i, n.j - 1})
	}
	if n.j < len(lt.ys)-1 {
		out = append(out, expNode{n.i, n.j + 1})
	}
	return out
}

// exits lists the straight roads from a node out to the map edge.
func (lt *expLattice) exits(n expNode, w, h int) [][]Cell {
	c := lt.at(n)
	var out [][]Cell
	if lt.west && n.i == 0 {
		out = append(out, expLine(c, Cell{0, c.Y}))
	}
	if !lt.west && n.i == len(lt.xs)-1 {
		out = append(out, expLine(c, Cell{w - 1, c.Y}))
	}
	if n.j == 0 {
		out = append(out, expLine(c, Cell{c.X, 0}))
	}
	if n.j == len(lt.ys)-1 {
		out = append(out, expLine(c, Cell{c.X, h - 1}))
	}
	return out
}

func expSign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// expLine lists the cells after a up to b on a straight line.
func expLine(a, b Cell) []Cell {
	dx, dy := expSign(b.X-a.X), expSign(b.Y-a.Y)
	var out []Cell
	for c := a; c != b; {
		c = Cell{c.X + dx, c.Y + dy}
		out = append(out, c)
	}
	return out
}

var expDirs = []Cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// expRoute is a generated road, from the base gate out to its entrance.
type expRoute struct {
	cells  []Cell
	parent int // the road it merges into (-1: none)
	join   int // index in the parent's cells of the merge node
}

type expMap struct {
	r      *rand.Rand
	w, h   int
	bx, by int // centre of the base
	road   map[Cell]bool
	holes  map[Cell]bool
	routes []expRoute
	pads   []Cell
}

// expGenMap generates a map of a layout, or nil if it keeps failing.
func expGenMap(r *rand.Rand, w, h int, lay expLayout, wyrm bool) *expMap {
	for try := 0; try < 80; try++ {
		m := &expMap{r: r, w: w, h: h, road: map[Cell]bool{}, holes: map[Cell]bool{}}
		m.bx = w - 3
		if lay.East > 0 {
			m.bx = w/2 + r.Intn(3) - 1
		}
		m.by = 3 + r.Intn(h-6)
		if !m.build(lay) || (wyrm && !m.placeHoles()) {
			continue
		}
		m.placePads(6 + 2*len(m.routes) + r.Intn(2))
		return m
	}
	return nil
}

func (m *expMap) isBase(c Cell) bool {
	return c.X >= m.bx-1 && c.X <= m.bx+1 && c.Y >= m.by-1 && c.Y <= m.by+1
}

// lattices lays out the lattice west of the base and, for a second front,
// the one east of it.
func (m *expMap) lattices(east bool) (*expLattice, *expLattice) {
	r := m.r
	ys := []int{m.by}
	for y := m.by - 3 - r.Intn(2); y >= 1; y -= 3 + r.Intn(2) {
		ys = append([]int{y}, ys...)
	}
	for y := m.by + 3 + r.Intn(2); y <= m.h-2; y += 3 + r.Intn(2) {
		ys = append(ys, y)
	}
	west := &expLattice{ys: ys, west: true}
	for x := m.bx - 3 - r.Intn(2); x >= 1; x -= 3 + r.Intn(3) {
		west.xs = append([]int{x}, west.xs...)
	}
	if !east {
		return west, nil
	}
	e := &expLattice{ys: ys}
	for x := m.bx + 3 + r.Intn(2); x <= m.w-2; x += 3 + r.Intn(3) {
		e.xs = append(e.xs, x)
	}
	return west, e
}

// clear reports whether new road cells leaving from cell from stay off the
// base and the other roads.
func (m *expMap) clear(cells []Cell, from Cell) bool {
	mine := map[Cell]bool{from: true}
	for _, c := range cells {
		mine[c] = true
	}
	for _, c := range cells {
		if c.X < 0 || c.Y < 0 || c.X >= m.w || c.Y >= m.h || m.road[c] || m.isBase(c) {
			return false
		}
		for _, d := range expDirs {
			nb := Cell{c.X + d.X, c.Y + d.Y}
			if !mine[nb] && (m.road[nb] || m.isBase(nb)) {
				return false
			}
		}
	}
	return true
}

func (m *expMap) commit(cells []Cell) {
	for _, c := range cells {
		m.road[c] = true
	}
}

// grow finds a road over unused lattice nodes from node from out to the map
// edge. The road so far has n cells; the finished one must have lo..hi.
func (m *expMap) grow(lt *expLattice, used map[expNode]bool, from expNode, n, lo, hi int) ([]Cell, bool) {
	var cells []Cell
	budget := 4000
	var dfs func(cur expNode, dir Cell, n int) bool
	dfs = func(cur expNode, dir Cell, n int) bool {
		if budget--; budget < 0 {
			return false
		}
		exits := lt.exits(cur, m.w, m.h)
		m.r.Shuffle(len(exits), func(i, j int) { exits[i], exits[j] = exits[j], exits[i] })
		exit := func() bool {
			for _, ex := range exits {
				if l := n + len(ex); l >= lo && l <= hi && m.clear(ex, lt.at(cur)) {
					cells = append(cells, ex...)
					return true
				}
			}
			return false
		}
		early := m.r.Float64() < 0.3
		if early && exit() {
			return true
		}
		if n < hi {
			nbs := lt.neighbors(cur)
			keys := make([]float64, len(nbs))
			for i, nb := range nbs {
				keys[i] = m.r.Float64()
				if d := expLine(lt.at(cur), lt.at(nb))[0]; (Cell{d.X - lt.at(cur).X, d.Y - lt.at(cur).Y}) == dir {
					keys[i] -= 0.25 // roads like to run straight on
				}
			}
			for i := 1; i < len(nbs); i++ {
				for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
					keys[j], keys[j-1] = keys[j-1], keys[j]
					nbs[j], nbs[j-1] = nbs[j-1], nbs[j]
				}
			}
			for _, nb := range nbs {
				if used[nb] {
					continue
				}
				seg := expLine(lt.at(cur), lt.at(nb))
				d := Cell{seg[0].X - lt.at(cur).X, seg[0].Y - lt.at(cur).Y}
				mark := len(cells)
				used[nb] = true
				cells = append(cells, seg...)
				if dfs(nb, d, n+len(seg)) {
					return true
				}
				used[nb] = false
				cells = cells[:mark]
			}
		}
		return !early && exit()
	}
	if !dfs(from, Cell{}, n) {
		return nil, false
	}
	return cells, true
}

// build lays the roads of a layout.
func (m *expMap) build(lay expLayout) bool {
	west, east := m.lattices(lay.East > 0)
	if len(west.xs) < 2 || (east != nil && len(east.xs) < 2) {
		return false
	}
	jb := slices.Index(west.ys, m.by)
	lo, hi := 34, 48
	if lay.East > 0 {
		lo, hi = 26, 38
	}
	used := map[expNode]bool{}
	gate := expNode{len(west.xs) - 1, jb}
	used[gate] = true
	seg := expLine(Cell{m.bx - 1, m.by}, west.at(gate))
	cells, ok := m.grow(west, used, gate, len(seg), lo, hi)
	if !ok {
		return false
	}
	trunk := append(seg, cells...)
	m.commit(trunk)
	m.routes = append(m.routes, expRoute{cells: trunk, parent: -1})
	joined := map[int]bool{}
	for b := 1; b < lay.West; b++ {
		var cands []int
		for i, c := range trunk {
			nd, isNode := west.node(c)
			if !isNode || joined[i] || i+1 < 8 || len(trunk)-i < 10 {
				continue
			}
			for _, nb := range west.neighbors(nd) {
				if !used[nb] {
					cands = append(cands, i)
					break
				}
			}
		}
		if len(cands) == 0 {
			return false
		}
		ji := cands[m.r.Intn(len(cands))]
		nd, _ := west.node(trunk[ji])
		cells, ok := m.grow(west, used, nd, ji+1, lo-4, hi-4)
		if !ok {
			return false
		}
		m.commit(cells)
		joined[ji] = true
		m.routes = append(m.routes, expRoute{cells: append(slices.Clone(trunk[:ji+1]), cells...), parent: 0, join: ji})
	}
	if east != nil {
		usedE := map[expNode]bool{}
		g := expNode{0, jb}
		usedE[g] = true
		seg := expLine(Cell{m.bx + 1, m.by}, east.at(g))
		cells, ok := m.grow(east, usedE, g, len(seg), 24, 36)
		if !ok {
			return false
		}
		route := append(seg, cells...)
		m.commit(route)
		m.routes = append(m.routes, expRoute{cells: route, parent: -1})
	}
	return true
}

func expStraight(cells []Cell, i int) bool {
	if i <= 0 || i >= len(cells)-1 {
		return false
	}
	a, b := cells[i-1], cells[i+1]
	return a.X == b.X || a.Y == b.Y
}

// pair digs two ice holes 5 cells apart, on straight road, in cells[lo..hi].
func (m *expMap) pair(cells []Cell, lo, hi int) bool {
	var cands []int
	for i := max(lo, 1); i+5 <= hi; i++ {
		if expStraight(cells, i) && expStraight(cells, i+5) && !m.holes[cells[i]] && !m.holes[cells[i+5]] {
			cands = append(cands, i)
		}
	}
	if len(cands) == 0 {
		return false
	}
	i := cands[m.r.Intn(len(cands))]
	m.holes[cells[i]] = true
	m.holes[cells[i+5]] = true
	return true
}

// placeHoles digs the lakes of the Ice Wyrm: every road must pass its holes
// in pairs, so a pair goes on the stretch merged roads share or on each road.
func (m *expMap) placeHoles() bool {
	for ri, rt := range m.routes {
		if rt.parent >= 0 {
			continue
		}
		shared, last := len(rt.cells)-3, 0
		var branches []expRoute
		for _, b := range m.routes {
			if b.parent == ri {
				branches = append(branches, b)
				shared, last = min(shared, b.join-1), max(last, b.join)
			}
		}
		if m.pair(rt.cells, 3, shared) {
			continue
		}
		if len(branches) == 0 || !m.pair(rt.cells, last+2, len(rt.cells)-3) {
			return false
		}
		for _, b := range branches {
			if !m.pair(b.cells, b.join+2, len(b.cells)-3) {
				return false
			}
		}
	}
	return true
}

// placePads puts n turret pads where they cover the most road, each new one
// preferring road the others cover little.
func (m *expMap) placePads(n int) {
	const reach = 2.6
	type spot struct {
		c    Cell
		hits int
	}
	var road []*spot
	spawns := map[Cell]bool{}
	for _, rt := range m.routes {
		for _, c := range rt.cells {
			road = append(road, &spot{c: c})
		}
		spawns[rt.cells[len(rt.cells)-1]] = true
	}
	near := func(a, b Cell, d int) bool { return max(expAbs(a.X-b.X), expAbs(a.Y-b.Y)) <= d }
	var cands []Cell
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			c := Cell{x, y}
			ok := !m.road[c] && !m.isBase(c)
			for s := range spawns {
				ok = ok && !near(c, s, 1)
			}
			if ok {
				cands = append(cands, c)
			}
		}
	}
	in := func(p, c Cell) bool {
		dx, dy := float64(p.X-c.X), float64(p.Y-c.Y)
		return dx*dx+dy*dy <= reach*reach
	}
	for len(m.pads) < n {
		best, bestS := -1, 0.0
		for i, c := range cands {
			free := true
			for _, p := range m.pads {
				free = free && !near(c, p, 1)
			}
			if !free {
				continue
			}
			s := 0.0
			for _, sp := range road {
				if in(c, sp.c) {
					s += 1 / float64(1+2*sp.hits)
				}
			}
			if s += m.r.Float64() * 0.8; s > bestS {
				best, bestS = i, s
			}
		}
		if best < 0 {
			return
		}
		p := cands[best]
		m.pads = append(m.pads, p)
		for _, sp := range road {
			if in(p, sp.c) {
				sp.hits++
			}
		}
	}
}

// rows draws the map in the level legend.
func (m *expMap) rows() []string {
	g := make([][]byte, m.h)
	for y := range g {
		g[y] = []byte(strings.Repeat(".", m.w))
	}
	for c := range m.road {
		g[c.Y][c.X] = '#'
	}
	for c := range m.holes {
		g[c.Y][c.X] = 'H'
	}
	for i, rt := range m.routes {
		c := rt.cells[len(rt.cells)-1]
		g[c.Y][c.X] = "S23"[i]
	}
	for _, c := range m.pads {
		g[c.Y][c.X] = 'o'
	}
	for y := m.by - 1; y <= m.by+1; y++ {
		for x := m.bx - 1; x <= m.bx+1; x++ {
			g[y][x] = 'B'
		}
	}
	out := make([]string, m.h)
	for y := range g {
		out[y] = string(g[y])
	}
	return out
}
