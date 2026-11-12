package game

import (
	"errors"
	"math"
	"math/rand"
	"sort"
)

type Phase int

const (
	PhasePrep Phase = iota
	PhaseWave
	PhaseWon
	PhaseLost
)

type EventKind int

const (
	EvSpawn EventKind = iota // Value is the entrance
	EvShot
	EvHit
	EvKill
	EvLeak
	EvFizzle
	EvWaveStart
	EvWaveClear
	EvWon
	EvLost
	EvBuild
	EvUpgrade
	EvHeal         // a repair pulse; ID is the healer, Value the number of patients
	EvHealed       // an enemy got repaired; Value is the amount
	EvBrood        // the queen laid eggs; Value is the count
	EvBuilt        // a turret finished building
	EvUpgradeStart // an upgrade job started; Level is the level being built
	EvCancel       // a construction job was cancelled; Value is the refund, Flag: the turret was removed
	EvSell         // a turret was sold; Value is the refund
	EvPulse        // a frost wave; R is its radius, Flag: it freezes
	EvRail         // a railgun beam from Pos to To; Flag: it pierces
	EvExplode      // a missile blew up; R is the blast radius, Flag: it hit its target
	EvNapalm       // a napalm pool was laid at Pos
	EvFreeze       // an enemy froze solid
	EvBounce       // a ricochet left Pos (Enemy is the type it bounced off)
	EvSplit        // a splitter burst; Value is the number of parts
	EvDive         // a diver went under the ice (Pos is the hole)
	EvSurface      // a diver came up through a hole; Value is the number of enemies it spat out
	EvGrab         // tentacles seized a turret: Pos is the seizer, To and ID the turret, Value the seizer, R the hold time
	EvRelease      // a seized turret is free again (ID is the turret, Value the seizer)
	EvPuff         // a Bloater left a cloud of fog (ID is the cloud, R its radius)
	EvBlast        // a Bloater exploded: R is the radius, Value the number of enemies caught
	EvHide         // a Stalker vanished in the fog
	EvReveal       // a hidden Stalker can be seen again
)

// Event is emitted by the simulation so the presentation layer can react.
type Event struct {
	Kind  EventKind
	Pos   Vec
	Angle float64
	ID    int
	Value int
	Level int
	Enemy *EnemyDef // the enemy type involved, if any
	Tower *TowerDef // the turret type involved, if any
	Flag  bool      // EvHit: armour absorbed part of the hit; EvSpawn: laid by the queen; EvKill: fire dealt the blow
	R     float64   // a radius or length in cells
	To    Vec
	Route int     // EvKill, EvDive, EvSurface: the route of the enemy
	Dist  float64 // EvKill, EvDive, EvSurface: how far along its route it was
}

type Enemy struct {
	ID      int
	Def     *EnemyDef
	Route   int // index of the route (entrance) it walks
	HP      float64
	MaxHP   float64
	Dist    float64 // distance travelled along the path
	Under   bool    // swimming under the ice: no turret can reach it
	Hidden  bool    // a Stalker in the fog: only turrets in the light of a Beacon can aim at it
	Offset  float64 // sideways offset from the path centre
	Pos     Vec
	Dir     Vec
	Age     float64
	HitT    float64 // seconds since the last hit
	HealT   float64 // seconds since the last repair
	PulseT  float64 // seconds since the last repair pulse (healers)
	SlowT   float64 // seconds of frost slow left
	SlowF   float64 // share of speed the frost takes away
	FreezeT float64 // seconds frozen solid left
	BurnT   float64 // seconds of burning left
	BurnDPS float64
	pending float64 // damage already in flight
	healCD  float64
	broodCD float64
	burnCD  float64
	grabCD  float64
	path    *Path
	alive   bool
}

func (e *Enemy) Alive() bool { return e.alive }

// Targetable reports whether turrets can aim at, hit or affect the enemy.
func (e *Enemy) Targetable() bool { return e.alive && !e.Under }

// PathAt is the position and direction on the enemy's route at distance d.
func (e *Enemy) PathAt(d float64) (Vec, Vec) { return e.path.At(d) }

// PathLen is the length of the enemy's route.
func (e *Enemy) PathLen() float64 { return e.path.Len }

// Left is the distance the enemy still has to walk to the base.
func (e *Enemy) Left() float64 { return e.path.Len - e.Dist }

// SpeedMul is the share of its normal speed the enemy moves at right now.
func (e *Enemy) SpeedMul() float64 {
	if e.FreezeT > 0 {
		return 0
	}
	if e.SlowT > 0 {
		return 1 - e.SlowF
	}
	return 1
}

func (e *Enemy) chill(slow, dur float64) {
	if e.Def.NoChill {
		return
	}
	if e.SlowT <= 0 {
		e.SlowF = slow
	} else {
		e.SlowF = math.Max(e.SlowF, slow)
	}
	e.SlowT = math.Max(e.SlowT, dur)
}

// burnTick is how often burning enemies take their burn damage.
const burnTick = 0.5

func (e *Enemy) ignite(dps, dur float64) {
	if e.BurnT <= 0 {
		e.BurnDPS = dps
		e.burnCD = burnTick
	} else {
		e.BurnDPS = math.Max(e.BurnDPS, dps)
	}
	e.BurnT = math.Max(e.BurnT, dur)
}

type spawnOrder struct {
	at    float64
	def   *EnemyDef
	route int
	hpMul float64
}

var (
	ErrNoCredits = errors.New("not enough credits")
	ErrInactive  = errors.New("the game is over")
	ErrMaxLevel  = errors.New("already at max level")
	ErrSlot      = errors.New("slot is not available")
)

// World is the complete, deterministic game state of one level run.
type World struct {
	Lvl         *Level
	Time        float64
	Phase       Phase
	Timer       float64
	TimerMax    float64
	Wave        int // number of waves started so far
	Credits     int
	BaseHP      int
	BaseMaxHP   int
	Enemies     []*Enemy
	Towers      []*Tower
	SlotTower   []*Tower
	Projectiles []*Projectile
	Patches     []*Patch
	Clouds      []*Cloud // fog: the drifting banks of the level and the puffs of Bloaters
	Events      []Event
	Kills       int
	Leaks       int
	Hold        bool // the first wave timer waits (while the player reads a briefing)

	pending   []spawnOrder
	waveClock float64
	nextID    int
	rng       *rand.Rand
}

func NewWorld(l *Level, seed int64) *World {
	s := l.Spec
	w := &World{
		Lvl: l, Phase: PhasePrep, Timer: s.FirstPrep, TimerMax: s.FirstPrep,
		Credits: s.StartCredits, BaseHP: s.BaseHP, BaseMaxHP: s.BaseHP,
		SlotTower: make([]*Tower, len(l.Slots)),
		rng:       rand.New(rand.NewSource(seed)),
	}
	w.initFog()
	return w
}

func (w *World) TotalWaves() int { return len(w.Lvl.Spec.Waves) }

// Stars rates a finished game: one for the win, two with at least half of the
// base left and three with at least 90%.
func (w *World) Stars() int {
	if w.Phase != PhaseWon {
		return 0
	}
	switch {
	case w.BaseHP*10 >= w.BaseMaxHP*9:
		return 3
	case w.BaseHP*2 >= w.BaseMaxHP:
		return 2
	}
	return 1
}
func (w *World) Active() bool { return w.Phase == PhasePrep || w.Phase == PhaseWave }

// EnemiesLeft counts alive enemies plus the ones still waiting to spawn.
func (w *World) EnemiesLeft() int {
	n := len(w.pending)
	for _, e := range w.Enemies {
		if e.alive {
			n++
		}
	}
	return n
}

// Spawning reports whether the current wave still has enemies to send
// through an entrance.
func (w *World) Spawning(route int) bool {
	for _, o := range w.pending {
		if o.route == route {
			return true
		}
	}
	return false
}

func (w *World) id() int { w.nextID++; return w.nextID }

func (w *World) emit(e Event) { w.Events = append(w.Events, e) }

// Update advances the simulation using small fixed-size substeps.
func (w *World) Update(dt float64) {
	const maxStep = 1.0 / 120
	for dt > 1e-9 && w.Active() {
		s := math.Min(dt, maxStep)
		w.step(s)
		dt -= s
	}
}

func (w *World) step(dt float64) {
	w.Time += dt
	switch w.Phase {
	case PhasePrep:
		if !w.Hold || w.Wave > 0 {
			w.Timer -= dt
		}
		if w.Timer <= 0 {
			w.startWave()
		}
	case PhaseWave:
		w.waveClock += dt
		for len(w.pending) > 0 && w.pending[0].at <= w.waveClock {
			o := w.pending[0]
			w.pending = w.pending[1:]
			w.spawnOn(o.route, o.def, o.hpMul, 0, false)
		}
	}
	w.updateFog(dt)
	w.updateEnemies(dt)
	if w.Phase == PhaseLost {
		return
	}
	w.updateTowers(dt)
	w.updateProjectiles(dt)
	w.updatePatches(dt)
	w.cleanup()
	if w.Phase == PhaseWave && len(w.pending) == 0 && len(w.Enemies) == 0 {
		w.waveCleared()
	}
}

// SkipTimer starts the next wave right away during the preparation phase.
func (w *World) SkipTimer() bool {
	if w.Phase != PhasePrep {
		return false
	}
	w.startWave()
	return true
}

func (w *World) startWave() {
	if w.Wave >= w.TotalWaves() {
		return
	}
	spec := w.Lvl.Spec.Waves[w.Wave]
	w.Wave++
	w.Phase = PhaseWave
	w.waveClock = 0
	w.Timer = 0
	w.pending = w.pending[:0]
	for _, g := range spec.Groups {
		def := EnemyDefs[g.Enemy]
		mul := g.HPMul
		if mul == 0 {
			mul = 1
		}
		route := g.entrance(len(w.Lvl.Routes))
		for i := 0; i < g.Count; i++ {
			w.pending = append(w.pending, spawnOrder{at: g.Delay + float64(i)*g.Interval, def: def, route: route, hpMul: mul})
		}
	}
	sort.SliceStable(w.pending, func(i, j int) bool { return w.pending[i].at < w.pending[j].at })
	w.driftFog()
	w.emit(Event{Kind: EvWaveStart, Value: w.Wave})
}

func (w *World) waveCleared() {
	if w.Wave >= w.TotalWaves() {
		w.Phase = PhaseWon
		w.emit(Event{Kind: EvWaveClear, Level: w.Wave})
		w.emit(Event{Kind: EvWon})
		return
	}
	bonus := w.Lvl.Spec.WaveBonus
	w.Credits += bonus
	w.emit(Event{Kind: EvWaveClear, Value: bonus, Level: w.Wave})
	w.Phase = PhasePrep
	w.Timer = w.Lvl.Spec.Prep
	w.TimerMax = w.Timer
}

func (w *World) spawn(def *EnemyDef, hpMul, dist float64, laid bool) *Enemy {
	return w.spawnOn(0, def, hpMul, dist, laid)
}

// spawnOn puts a new enemy on a route at the given distance from its entrance.
func (w *World) spawnOn(route int, def *EnemyDef, hpMul, dist float64, laid bool) *Enemy {
	hp := def.HP * hpMul
	e := &Enemy{
		ID: w.id(), Def: def, Route: route, HP: hp, MaxHP: hp, Dist: dist, Offset: (w.rng.Float64()*2 - 1) * def.Spread,
		HitT: 99, HealT: 99, PulseT: 99, path: w.Lvl.Routes[route].Path, alive: true,
	}
	if def.Heal != nil {
		e.healCD = def.Heal.Period * 0.5
	}
	if def.Brood != nil {
		e.broodCD = def.Brood.Period
	}
	if def.Grab != nil {
		e.grabCD = def.Grab.Period * 0.5
	}
	w.place(e)
	if def.Stealth {
		e.Hidden = w.Foggy(e.Pos)
	}
	w.Enemies = append(w.Enemies, e)
	w.emit(Event{Kind: EvSpawn, Pos: e.Pos, ID: e.ID, Enemy: def, Flag: laid, Value: route})
	return e
}

// place updates the position of an enemy from its distance along the road.
func (w *World) place(e *Enemy) {
	p := e.path
	pos, dir := p.At(e.Dist)
	off := e.Offset * Smoothstep(0, 1.2, e.Dist) * Smoothstep(0, 1.2, p.Len-e.Dist)
	e.Pos = pos.Add(dir.Perp().Mul(off))
	e.Dir = dir
}

func (w *World) updateEnemies(dt float64) {
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		p := e.path
		e.Age += dt
		e.HitT += dt
		e.HealT += dt
		e.PulseT += dt
		speed := e.Def.Speed * e.SpeedMul()
		if e.Under {
			speed *= e.Def.Dive
		}
		e.Dist += speed * dt
		w.place(e)
		if e.Def.Dive > 0 {
			w.updateDive(e)
		}
		if e.Def.Stealth {
			w.updateHidden(e)
		}
		if g := e.Def.Grab; g != nil && !e.Under && e.Dist < p.Len-0.5 {
			w.updateGrab(e, g, dt)
		}
		e.FreezeT = math.Max(0, e.FreezeT-dt)
		if e.SlowT > 0 {
			if e.SlowT -= dt; e.SlowT <= 0 {
				e.SlowT, e.SlowF = 0, 0
			}
		}
		if e.BurnT > 0 {
			e.BurnT -= dt
			if e.burnCD -= dt; e.burnCD <= 0 {
				e.burnCD += burnTick
				w.hurtBy(e, e.Def.EffDamage(e.BurnDPS*burnTick*e.Def.FireMul()), true)
				if !e.alive {
					continue
				}
			}
			if e.BurnT <= 0 {
				e.BurnT, e.BurnDPS = 0, 0
			}
		}
		if h := e.Def.Heal; h != nil {
			e.healCD -= dt
			if e.healCD <= 0 {
				e.healCD += h.Period
				w.healPulse(e, h)
			}
		}
		if b := e.Def.Brood; b != nil && e.Dist < p.Len-1.5 && !e.Under {
			e.broodCD -= dt
			if e.broodCD <= 0 {
				e.broodCD += b.Period
				w.layBrood(e, b)
			}
		}
		if e.Dist >= p.Len {
			e.alive = false
			w.BaseHP -= e.Def.Damage
			w.Leaks++
			w.emit(Event{Kind: EvLeak, Pos: e.Pos, ID: e.ID, Value: e.Def.Damage, Enemy: e.Def})
			if w.BaseHP <= 0 {
				w.BaseHP = 0
				w.Phase = PhaseLost
				w.emit(Event{Kind: EvLost})
				return
			}
		}
	}
}

// updateDive takes a diver under the ice or back up when it passes a hole.
func (w *World) updateDive(e *Enemy) {
	rt := w.Lvl.Routes[e.Route]
	under := rt.UnderIce(e.Dist)
	if under == e.Under {
		return
	}
	e.Under = under
	if under {
		// The water puts out the fire, and the shots in flight lose it.
		e.BurnT, e.BurnDPS = 0, 0
		e.SlowT, e.SlowF, e.FreezeT = 0, 0, 0
		e.pending = 0
		for _, p := range w.Projectiles {
			if p.TargetID == e.ID {
				p.TargetID = 0
			}
		}
		w.emit(Event{Kind: EvDive, Pos: e.Pos, ID: e.ID, Enemy: e.Def, Angle: e.Dir.Angle(), Route: e.Route, Dist: e.Dist})
		return
	}
	n := 0
	if sp := e.Def.Spit; sp != nil {
		def := EnemyDefs[sp.Enemy]
		for i := 0; i < sp.Count; i++ {
			d := math.Min(e.Dist+1.1+0.45*float64(i), e.path.Len-1)
			s := w.spawnOn(e.Route, def, sp.HPMul, d, true)
			s.Age = 0.15
			n++
		}
	}
	w.emit(Event{Kind: EvSurface, Pos: e.Pos, ID: e.ID, Enemy: e.Def, Angle: e.Dir.Angle(), Value: n, Route: e.Route, Dist: e.Dist})
}

// healPulse repairs the damaged ground enemies around a healer.
func (w *World) healPulse(src *Enemy, h *HealSpec) {
	src.PulseT = 0
	n := 0
	r2 := h.Radius * h.Radius
	for _, e := range w.Enemies {
		if !e.Targetable() || e == src || e.Def.Flying || e.HP >= e.MaxHP || e.Pos.Dist2(src.Pos) > r2 {
			continue
		}
		amt := math.Min(math.Min(e.MaxHP*h.Frac, h.Cap), e.MaxHP-e.HP)
		e.HP += amt
		e.HealT = 0
		n++
		w.emit(Event{Kind: EvHealed, Pos: e.Pos, ID: e.ID, Value: int(math.Ceil(amt)), Enemy: e.Def})
	}
	w.emit(Event{Kind: EvHeal, Pos: src.Pos, ID: src.ID, Value: n, Enemy: src.Def})
}

// layBrood spawns a few small enemies right behind the queen.
func (w *World) layBrood(src *Enemy, b *BroodSpec) {
	def := EnemyDefs[b.Enemy]
	w.emit(Event{Kind: EvBrood, Pos: src.Pos, ID: src.ID, Value: b.Count, Enemy: src.Def, Angle: src.Dir.Angle()})
	for i := 0; i < b.Count; i++ {
		d := math.Max(0, src.Dist-0.45-0.22*float64(i))
		e := w.spawnOn(src.Route, def, b.HPMul, d, true)
		e.Age = 0.2 // skip most of the emerge fade
	}
}

func (w *World) enemyByID(id int) *Enemy {
	for _, e := range w.Enemies {
		if e.ID == id {
			if e.alive {
				return e
			}
			return nil
		}
	}
	return nil
}

// damage hits an enemy with a visible impact (it flashes).
func (w *World) damage(e *Enemy, dmg float64) {
	if e.alive {
		e.HitT = 0
	}
	w.hurt(e, dmg)
}

// hurt takes health without the hit flash (frost).
func (w *World) hurt(e *Enemy, dmg float64) { w.hurtBy(e, dmg, false) }

// hurtBy takes health; fire tells whether flames or burning dealt it (fire
// makes a Bloater explode instead of leaving fog behind).
func (w *World) hurtBy(e *Enemy, dmg float64, fire bool) {
	if !e.alive {
		return
	}
	e.HP -= dmg
	if e.HP <= 0 {
		e.HP = 0
		e.alive = false
		w.Credits += e.Def.Reward
		w.Kills++
		w.emit(Event{Kind: EvKill, Pos: e.Pos, ID: e.ID, Value: e.Def.Reward, Angle: e.Dir.Angle(), Enemy: e.Def, Route: e.Route, Dist: e.Dist, Flag: fire})
		if sp := e.Def.Split; sp != nil {
			w.split(e, sp)
		}
		switch {
		case fire && e.Def.Blast != nil:
			w.blast(e, e.Def.Blast)
		case e.Def.Puff != nil:
			w.puff(e, e.Def.Puff)
		}
	}
}

// split bursts a destroyed enemy into its parts around the spot where it fell.
func (w *World) split(src *Enemy, sp *SplitSpec) {
	def := EnemyDefs[sp.Enemy]
	mul := src.MaxHP / src.Def.HP * sp.HPMul
	w.emit(Event{Kind: EvSplit, Pos: src.Pos, ID: src.ID, Value: sp.Count, Enemy: src.Def, Angle: src.Dir.Angle()})
	for i := 0; i < sp.Count; i++ {
		d := math.Max(0, src.Dist+0.3*(float64(i)-float64(sp.Count-1)/2))
		e := w.spawnOn(src.Route, def, mul, d, true)
		e.Age = 0.15 // a quick pop instead of the burrow fade
	}
}

func (w *World) cleanup() {
	es := w.Enemies[:0]
	for _, e := range w.Enemies {
		if e.alive {
			es = append(es, e)
		}
	}
	for i := len(es); i < len(w.Enemies); i++ {
		w.Enemies[i] = nil
	}
	w.Enemies = es
	ps := w.Projectiles[:0]
	for _, p := range w.Projectiles {
		if p.alive {
			ps = append(ps, p)
		}
	}
	for i := len(ps); i < len(w.Projectiles); i++ {
		w.Projectiles[i] = nil
	}
	w.Projectiles = ps
}

// WaveEntry summarises one enemy type of a wave.
type WaveEntry struct {
	Def   *EnemyDef
	Count int
	HPMul float64 // the highest health multiplier of its groups
	From  int     // entrance (SummaryFrom only)
}

// Summary lists the enemy types of a wave in introduction order.
func (s WaveSpec) Summary() []WaveEntry { return s.summary(-1, 0) }

// SummaryFrom lists what a wave sends through one of the n entrances of a level.
func (s WaveSpec) SummaryFrom(from, n int) []WaveEntry { return s.summary(from, n) }

func (s WaveSpec) summary(from, n int) []WaveEntry {
	var out []WaveEntry
	for _, d := range EnemyOrder {
		en := WaveEntry{Def: d, From: max(from, 0)}
		for _, g := range s.Groups {
			if g.Enemy != d.ID || from >= 0 && g.entrance(n) != from {
				continue
			}
			en.Count += g.Count
			mul := g.HPMul
			if mul == 0 {
				mul = 1
			}
			en.HPMul = math.Max(en.HPMul, mul)
		}
		if en.Count > 0 {
			out = append(out, en)
		}
	}
	return out
}

// Total is the number of enemies a wave sends (not counting brood).
func (s WaveSpec) Total() int {
	n := 0
	for _, g := range s.Groups {
		n += g.Count
	}
	return n
}

// Has reports whether the wave contains the enemy type.
func (s WaveSpec) Has(d *EnemyDef) bool {
	for _, g := range s.Groups {
		if g.Enemy == d.ID {
			return true
		}
	}
	return false
}

// NextWave returns the spec of the wave the preparation timer counts down to.
func (w *World) NextWave() (WaveSpec, bool) {
	if w.Phase != PhasePrep || w.Wave >= w.TotalWaves() {
		return WaveSpec{}, false
	}
	return w.Lvl.Spec.Waves[w.Wave], true
}

// Boss returns the living boss, if any.
func (w *World) Boss() *Enemy {
	for _, e := range w.Enemies {
		if e.alive && e.Def.Boss {
			return e
		}
	}
	return nil
}

// DebugStartAt jumps to the preparation phase before wave n (1-based).
// DebugWin ends the level as won with the base intact (testing).
func (w *World) DebugWin() {
	w.Wave = w.TotalWaves()
	w.Phase = PhaseWon
	w.emit(Event{Kind: EvWon})
}

func (w *World) DebugStartAt(n, credits int) {
	n = max(1, min(n, w.TotalWaves()))
	w.Wave = n - 1
	for _, c := range w.Clouds {
		if !c.Puff() {
			p := w.Lvl.Spec.Fog[c.Bank].StopFor(n)
			c.Pos, c.From, c.To = p, p, p
		}
	}
	if credits > 0 {
		w.Credits = credits
	}
}
