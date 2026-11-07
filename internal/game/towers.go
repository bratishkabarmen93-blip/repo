package game

import (
	"errors"
	"math"
	"sort"
)

// AttackKind is how a turret type deals its damage.
type AttackKind int

const (
	AtkBolt    AttackKind = iota // plasma bolts at one target
	AtkMissile                   // slow homing missiles (anti-air only)
	AtkPulse                     // frost waves around the turret
	AtkFlame                     // a short cone of fire that sets enemies alight
	AtkRail                      // an instant beam that ignores armour
)

// TargetMode decides which enemy in range a turret shoots at.
type TargetMode int

const (
	TargetFirst  TargetMode = iota // closest to the base
	TargetLast                     // furthest from the base
	TargetStrong                   // toughest enemy type, then closest to the base
	TargetClose                    // nearest to the turret
	NumTargetModes
)

var targetModeNames = [NumTargetModes]string{"First", "Last", "Strong", "Close"}

func (m TargetMode) String() string { return targetModeNames[m] }

// TowerLevel holds the stats of one upgrade level.
type TowerLevel struct {
	Cost      int
	Build     float64 // seconds to build or upgrade to this level
	Damage    float64 // per bolt, missile, frost wave, flame tick or beam
	FireRate  float64 // shots, waves or flame ticks per second
	Range     float64 // cells
	ProjSpeed float64 // cells per second
	Splash    float64 // missile blast radius (cells)
	Slow      float64 // share of speed a frost wave takes away
	SlowTime  float64
	Burn      float64 // burn damage per second
	BurnTime  float64
}

func (l TowerLevel) DPS() float64 { return l.Damage * l.FireRate }

// SuperDef is the last level of a turret: it adds an ability instead of stats.
type SuperDef struct {
	Name        string
	Desc        string
	Cost        int
	Build       float64
	Bounces     int     // blaster: ricochets per bolt
	BounceDmg   float64 // blaster: damage share of each ricochet
	BounceR     float64 // blaster: ricochet search radius (cells)
	Salvo       int     // skyguard: missiles per launch
	FreezeEvery int     // cryo: every n-th wave freezes
	FreezeTime  float64
	PatchEvery  float64 // flamer: seconds between napalm pools while firing
	PatchLife   float64
	PatchR      float64
	Pierce      bool // railgun: the beam hits every enemy in its line
}

// TowerDef describes a turret type.
type TowerDef struct {
	ID, Name    string
	Role        string // a few words for the build menu
	Desc        string
	Attack      AttackKind
	TurnRate    float64 // radians per second
	AntiAir     bool    // targets flying enemies only (ground towers ignore them)
	IgnoreArmor bool
	Cone        float64 // flamer: half angle of the fire cone
	Mode        TargetMode
	Levels      []TowerLevel
	Super       SuperDef
}

// CanTarget reports whether the turret type can shoot at the enemy type.
func (t *TowerDef) CanTarget(e *EnemyDef) bool { return t.AntiAir == e.Flying }

// Aims reports whether the turret picks targets (and so has targeting modes).
func (t *TowerDef) Aims() bool     { return t.Attack != AtkPulse }
func (t *TowerDef) NumLevels() int { return len(t.Levels) + 1 }
func (t *TowerDef) Cost() int      { return t.Levels[0].Cost }

// SellRefund is the share of the invested credits a sale returns.
const SellRefund = 0.7

const (
	buildTime = 3.0
	superTime = 5.0
)

var (
	Blaster = &TowerDef{
		ID: "blaster", Name: "Blaster", Role: "All-round", Attack: AtkBolt, TurnRate: 9,
		Desc: "Reliable plasma turret that handles anything on the ground.",
		Levels: []TowerLevel{
			{Cost: 60, Build: buildTime, Damage: 10, FireRate: 1.0, Range: 2.5, ProjSpeed: 11},
			{Cost: 70, Build: buildTime, Damage: 18, FireRate: 1.15, Range: 2.8, ProjSpeed: 12},
			{Cost: 110, Build: buildTime, Damage: 30, FireRate: 1.3, Range: 3.1, ProjSpeed: 13},
		},
		Super: SuperDef{Name: "Ricochet", Cost: 130, Build: superTime, Bounces: 2, BounceDmg: 0.5, BounceR: 1.8,
			Desc: "Every bolt bounces on to 2 more enemies for half damage."},
	}
	Cryo = &TowerDef{
		ID: "cryo", Name: "Cryo", Role: "Slows", Attack: AtkPulse,
		Desc: "Sends out frost waves that slow every ground enemy around it.",
		Levels: []TowerLevel{
			{Cost: 70, Build: buildTime, Damage: 2, FireRate: 0.5, Range: 2.6, Slow: 0.4, SlowTime: 3.5},
			{Cost: 60, Build: buildTime, Damage: 3, FireRate: 0.55, Range: 2.8, Slow: 0.5, SlowTime: 3.7},
			{Cost: 90, Build: buildTime, Damage: 5, FireRate: 0.6, Range: 3.0, Slow: 0.55, SlowTime: 3.9},
		},
		Super: SuperDef{Name: "Deep Freeze", Cost: 110, Build: superTime, FreezeEvery: 3, FreezeTime: 1.2,
			Desc: "Every third wave freezes enemies solid for a moment. The Queen resists."},
	}
	Flamer = &TowerDef{
		ID: "flamer", Name: "Flamer", Role: "Burns crowds", Attack: AtkFlame, TurnRate: 6, Cone: 0.42,
		Desc: "Short-range flamethrower that sets whole groups alight. Armor soaks most of it.",
		Levels: []TowerLevel{
			{Cost: 80, Build: buildTime, Damage: 2, FireRate: 5, Range: 2.3, Burn: 4, BurnTime: 2.5},
			{Cost: 70, Build: buildTime, Damage: 3, FireRate: 5, Range: 2.45, Burn: 6, BurnTime: 2.5},
			{Cost: 100, Build: buildTime, Damage: 4, FireRate: 5, Range: 2.6, Burn: 9, BurnTime: 2.5},
		},
		Super: SuperDef{Name: "Napalm", Cost: 120, Build: superTime, PatchEvery: 1.5, PatchLife: 4, PatchR: 0.55,
			Desc: "Leaves pools of burning napalm on the road."},
	}
	Railgun = &TowerDef{
		ID: "railgun", Name: "Railgun", Role: "Pierces armor", Attack: AtkRail, TurnRate: 3.2, IgnoreArmor: true,
		Desc: "Slow, long-range magnetic cannon. Its shots ignore armor.",
		Levels: []TowerLevel{
			{Cost: 110, Build: buildTime, Damage: 60, FireRate: 0.4, Range: 4.0},
			{Cost: 90, Build: buildTime, Damage: 95, FireRate: 0.42, Range: 4.4},
			{Cost: 130, Build: buildTime, Damage: 140, FireRate: 0.45, Range: 4.8},
		},
		Super: SuperDef{Name: "Overcharge", Cost: 150, Build: superTime, Pierce: true,
			Desc: "The beam pierces every enemy in its line."},
	}
	Skyguard = &TowerDef{
		ID: "skyguard", Name: "Skyguard", Role: "Anti-air", Attack: AtkMissile, TurnRate: 5, AntiAir: true,
		Desc: "Launches slow homing missiles. Shoots flying enemies only.",
		Levels: []TowerLevel{
			{Cost: 70, Build: buildTime, Damage: 30, FireRate: 0.3, Range: 3.6, ProjSpeed: 6.5, Splash: 0.7},
			{Cost: 60, Build: buildTime, Damage: 45, FireRate: 0.33, Range: 4.0, ProjSpeed: 7, Splash: 0.7},
			{Cost: 90, Build: buildTime, Damage: 65, FireRate: 0.36, Range: 4.4, ProjSpeed: 7.5, Splash: 0.8},
		},
		Super: SuperDef{Name: "Salvo", Cost: 100, Build: superTime, Salvo: 3,
			Desc: "Fires three missiles at once, at different targets when it can."},
	}

	// TowerOrder is the order of the build menu (hotkeys 1-5).
	TowerOrder = []*TowerDef{Blaster, Cryo, Flamer, Railgun, Skyguard}
	TowerDefs  = map[string]*TowerDef{}
)

func init() {
	for _, d := range TowerOrder {
		TowerDefs[d.ID] = d
	}
}

type Tower struct {
	ID       int
	Def      *TowerDef
	Level    int // 0..2 are stat levels, 3 adds the super ability
	Slot     int
	Pos      Vec
	Angle    float64
	Mode     TargetMode
	Cooldown float64
	TargetID int
	ShotT    float64 // seconds since the last shot, wave, flame tick or beam
	BuiltT   float64 // seconds since the last build or upgrade finished
	Spent    int     // credits in finished levels
	Busy     bool    // under construction: the turret does nothing
	Fresh    bool    // the job is the initial build (cancelling removes the turret)
	Work     float64 // seconds of the current job done
	WorkDur  float64
	WorkCost int  // credits paid for the current job, refunded in full on cancel
	Pulses   int  // frost waves sent (deep freeze counts them)
	NoSuper  bool // super abilities are locked on this level
	patchCD  float64
}

func (t *Tower) StatLevel() int    { return min(t.Level, len(t.Def.Levels)-1) }
func (t *Tower) Stats() TowerLevel { return t.Def.Levels[t.StatLevel()] }
func (t *Tower) HasSuper() bool    { return t.Level >= len(t.Def.Levels) }
func (t *Tower) MaxLevel() bool    { return t.Level >= t.TopLevel() }
func (t *Tower) NextIsSuper() bool { return !t.NoSuper && t.Level == len(t.Def.Levels)-1 }

// TopLevel is the highest level the turret can reach (the super, unless it is locked).
func (t *Tower) TopLevel() int {
	if t.NoSuper {
		return len(t.Def.Levels) - 1
	}
	return len(t.Def.Levels)
}

// UpgradeCost is the price of the next level (the super ability after level 3).
func (t *Tower) UpgradeCost() int {
	switch {
	case t.MaxLevel():
		return 0
	case t.NextIsSuper():
		return t.Def.Super.Cost
	}
	return t.Def.Levels[t.Level+1].Cost
}

func (t *Tower) upgradeTime() float64 {
	if t.NextIsSuper() {
		return t.Def.Super.Build
	}
	return t.Def.Levels[t.Level+1].Build
}

// NextStats are the stats after the next upgrade (the super keeps them).
func (t *Tower) NextStats() TowerLevel { return t.Def.Levels[min(t.Level+1, len(t.Def.Levels)-1)] }

// JobLevel is the level the current construction job builds.
func (t *Tower) JobLevel() int {
	if t.Busy && !t.Fresh {
		return t.Level + 1
	}
	return t.Level
}

// Progress of the current construction job, 0..1 (1 when idle).
func (t *Tower) Progress() float64 {
	if !t.Busy || t.WorkDur <= 0 {
		return 1
	}
	return Clamp(t.Work/t.WorkDur, 0, 1)
}

// SellValue is what selling the finished turret returns.
func (t *Tower) SellValue() int { return int(math.Floor(float64(t.Spent)*SellRefund + 1e-9)) }

var (
	ErrBusy   = errors.New("the turret is under construction")
	ErrIdle   = errors.New("nothing to cancel")
	ErrLocked = errors.New("this turret is not available on this level")
)

// CanBuild reports whether turrets of type def are unlocked on this level.
func (w *World) CanBuild(def *TowerDef) bool { return w.Lvl.Spec.Allows(def) }

// CanTarget reports whether target priority can be changed on this level.
func (w *World) CanTarget() bool { return !w.Lvl.Spec.NoTarget }

// Build starts the construction of a turret on an empty pad.
func (w *World) Build(slot int, def *TowerDef) (*Tower, error) {
	if !w.Active() {
		return nil, ErrInactive
	}
	if slot < 0 || slot >= len(w.SlotTower) || w.SlotTower[slot] != nil {
		return nil, ErrSlot
	}
	if !w.CanBuild(def) {
		return nil, ErrLocked
	}
	lv := def.Levels[0]
	if w.Credits < lv.Cost {
		return nil, ErrNoCredits
	}
	w.Credits -= lv.Cost
	pos := w.Lvl.Slots[slot].Center()
	near := w.Lvl.NearestRoad(pos)
	t := &Tower{
		ID: w.id(), Def: def, Slot: slot, Pos: pos, Angle: near.Sub(pos).Angle(), Mode: def.Mode,
		ShotT: 99, BuiltT: 99, Busy: true, Fresh: true, WorkDur: lv.Build, WorkCost: lv.Cost,
		NoSuper: w.Lvl.Spec.NoSuper,
	}
	w.Towers = append(w.Towers, t)
	w.SlotTower[slot] = t
	w.emit(Event{Kind: EvBuild, Pos: pos, ID: t.ID, Tower: def})
	if t.WorkDur <= 0 {
		w.finishJob(t)
	}
	return t, nil
}

// Upgrade starts raising a turret to its next level.
func (w *World) Upgrade(t *Tower) error {
	switch {
	case !w.Active():
		return ErrInactive
	case t.Busy:
		return ErrBusy
	case t.MaxLevel():
		return ErrMaxLevel
	}
	cost := t.UpgradeCost()
	if w.Credits < cost {
		return ErrNoCredits
	}
	w.Credits -= cost
	t.Busy, t.Fresh, t.Work, t.WorkDur, t.WorkCost = true, false, 0, t.upgradeTime(), cost
	t.TargetID = 0
	w.emit(Event{Kind: EvUpgradeStart, Pos: t.Pos, ID: t.ID, Level: t.Level + 1, Tower: t.Def})
	if t.WorkDur <= 0 {
		w.finishJob(t)
	}
	return nil
}

// Cancel stops the current construction job and refunds it in full.
func (w *World) Cancel(t *Tower) error {
	if !w.Active() {
		return ErrInactive
	}
	if !t.Busy {
		return ErrIdle
	}
	w.Credits += t.WorkCost
	w.emit(Event{Kind: EvCancel, Pos: t.Pos, ID: t.ID, Value: t.WorkCost, Level: t.Level, Tower: t.Def, Flag: t.Fresh})
	if t.Fresh {
		w.removeTower(t)
		return nil
	}
	t.Busy, t.Work, t.WorkCost = false, 0, 0
	return nil
}

// Sell removes a finished turret and returns part of its price.
func (w *World) Sell(t *Tower) error {
	if !w.Active() {
		return ErrInactive
	}
	if t.Busy {
		return ErrBusy
	}
	v := t.SellValue()
	w.Credits += v
	w.emit(Event{Kind: EvSell, Pos: t.Pos, ID: t.ID, Value: v, Level: t.Level, Tower: t.Def})
	w.removeTower(t)
	return nil
}

// CycleMode switches a turret to its next targeting mode.
func (w *World) CycleMode(t *Tower) {
	if t.Def.Aims() && w.CanTarget() {
		t.Mode = (t.Mode + 1) % NumTargetModes
	}
}

// PlaceTower puts a finished turret of the given level on a pad for free
// (the menu backdrop uses it).
func (w *World) PlaceTower(slot int, def *TowerDef, level int) *Tower {
	if slot < 0 || slot >= len(w.SlotTower) || w.SlotTower[slot] != nil {
		return nil
	}
	pos := w.Lvl.Slots[slot].Center()
	near := w.Lvl.NearestRoad(pos)
	t := &Tower{
		ID: w.id(), Def: def, Slot: slot, Pos: pos, Angle: near.Sub(pos).Angle(), Mode: def.Mode,
		ShotT: 99, BuiltT: 99, Level: min(level, len(def.Levels)), NoSuper: w.Lvl.Spec.NoSuper,
	}
	for i := 0; i <= min(t.Level, len(def.Levels)-1); i++ {
		t.Spent += def.Levels[i].Cost
	}
	if t.HasSuper() {
		t.Spent += def.Super.Cost
	}
	w.Towers = append(w.Towers, t)
	w.SlotTower[slot] = t
	return t
}

func (w *World) removeTower(t *Tower) {
	w.SlotTower[t.Slot] = nil
	ts := w.Towers[:0]
	for _, o := range w.Towers {
		if o != t {
			ts = append(ts, o)
		}
	}
	w.Towers[len(ts)] = nil
	w.Towers = ts
}

func (w *World) finishJob(t *Tower) {
	t.Busy = false
	t.Spent += t.WorkCost
	t.Work, t.WorkCost, t.BuiltT = 0, 0, 0
	if t.Fresh {
		t.Fresh = false
		t.Cooldown = 0
		w.emit(Event{Kind: EvBuilt, Pos: t.Pos, ID: t.ID, Tower: t.Def})
		return
	}
	t.Level++
	w.emit(Event{Kind: EvUpgrade, Pos: t.Pos, ID: t.ID, Level: t.Level, Tower: t.Def})
}

// --- combat ----------------------------------------------------------------

// effDamage is the damage a hit of a turret type deals to an enemy type.
func effDamage(td *TowerDef, ed *EnemyDef, dmg float64) float64 {
	if td.IgnoreArmor {
		return dmg
	}
	return ed.EffDamage(dmg)
}

func (t *Tower) score(e *Enemy) float64 {
	switch t.Mode {
	case TargetLast:
		return e.Left()
	case TargetStrong:
		return e.MaxHP*1e3 - e.Left()
	case TargetClose:
		return -e.Pos.Dist2(t.Pos) - e.Left()*1e-4
	}
	return -e.Left()
}

// pickTarget chooses an enemy in range by the turret's targeting mode; enemies
// already doomed by shots in flight are only a fallback.
func (w *World) pickTarget(t *Tower, r float64) *Enemy {
	var best, doomed *Enemy
	var bv, dv float64
	r2 := r * r
	for _, e := range w.Enemies {
		if !e.Targetable() || !t.Def.CanTarget(e.Def) || e.Pos.Dist2(t.Pos) > r2 {
			continue
		}
		v := t.score(e)
		if e.HP-e.pending <= 0 {
			if doomed == nil || v > dv {
				doomed, dv = e, v
			}
			continue
		}
		if best == nil || v > bv {
			best, bv = e, v
		}
	}
	if best == nil {
		return doomed
	}
	return best
}

func (w *World) updateTowers(dt float64) {
	for _, t := range w.Towers {
		t.ShotT += dt
		t.BuiltT += dt
		if t.Busy {
			t.Work += dt
			if t.Work >= t.WorkDur {
				w.finishJob(t)
			}
			continue
		}
		t.Cooldown -= dt
		t.patchCD -= dt
		if t.Def.Attack == AtkPulse {
			w.updatePulse(t)
		} else {
			w.updateAiming(t, dt)
		}
	}
}

// aimTolerance is how far off a turret may point and still fire.
var aimTolerance = map[AttackKind]float64{AtkBolt: 0.2, AtkMissile: 0.7, AtkFlame: 0.3, AtkRail: 0.06}

func (w *World) updateAiming(t *Tower, dt float64) {
	st := t.Stats()
	target := w.pickTarget(t, st.Range)
	if target == nil {
		t.TargetID = 0
		t.Cooldown = math.Max(t.Cooldown, 0)
		return
	}
	want := target.Pos.Sub(t.Pos).Angle()
	diff := AngleDiff(want, t.Angle)
	if turn := t.Def.TurnRate * dt; math.Abs(diff) <= turn {
		t.Angle = want
	} else {
		t.Angle = WrapAngle(t.Angle + math.Copysign(turn, diff))
	}
	if target.HP-target.pending <= 0 && t.Def.Attack != AtkFlame {
		// Keep tracking, but do not waste a shot on an enemy that is about to die.
		t.TargetID = 0
		t.Cooldown = math.Max(t.Cooldown, 0)
		return
	}
	t.TargetID = target.ID
	if t.Cooldown > 0 || math.Abs(AngleDiff(want, t.Angle)) > aimTolerance[t.Def.Attack] {
		return
	}
	t.Cooldown = math.Max(t.Cooldown, -dt) + 1/st.FireRate
	t.ShotT = 0
	switch t.Def.Attack {
	case AtkBolt:
		muzzle := t.Pos.Add(FromAngle(t.Angle).Mul(0.42))
		p := w.newBolt(t.Def, t.Level, muzzle, nil, target, st.Damage, st.ProjSpeed)
		if t.HasSuper() {
			p.Bounces = t.Def.Super.Bounces
		}
		w.emit(Event{Kind: EvShot, Pos: muzzle, Angle: t.Angle, ID: t.ID, Level: t.Level, Tower: t.Def})
	case AtkMissile:
		w.fireMissiles(t, target, st)
	case AtkFlame:
		w.flameTick(t, target, st)
	case AtkRail:
		w.fireRail(t, target, st)
	}
}

func (w *World) newBolt(td *TowerDef, level int, from Vec, fromDef *EnemyDef, target *Enemy, dmg, speed float64) *Projectile {
	eff := effDamage(td, target.Def, dmg)
	p := &Projectile{
		ID: w.id(), Kind: ProjBolt, Def: td, Start: from, Pos: from, Prev: from, From: fromDef,
		TargetID: target.ID, Target: target.Def, Aim: target.Pos, Speed: speed, Damage: dmg, eff: eff,
		Level: level, alive: true,
	}
	target.pending += eff
	w.Projectiles = append(w.Projectiles, p)
	return p
}

// fireMissiles launches one missile, or a salvo spread over several targets.
func (w *World) fireMissiles(t *Tower, target *Enemy, st TowerLevel) {
	n := 1
	if t.HasSuper() {
		n = t.Def.Super.Salvo
	}
	targets := []*Enemy{target}
	if n > 1 {
		var more []*Enemy
		r2 := st.Range * st.Range
		for _, e := range w.Enemies {
			if e != target && e.Targetable() && t.Def.CanTarget(e.Def) && e.Pos.Dist2(t.Pos) <= r2 && e.HP-e.pending > 0 {
				more = append(more, e)
			}
		}
		sort.Slice(more, func(i, j int) bool { return t.score(more[i]) > t.score(more[j]) })
		targets = append(targets, more...)
	}
	for i := 0; i < n; i++ {
		tg := targets[i%len(targets)]
		h := t.Angle
		if n > 1 {
			h += (float64(i) - float64(n-1)/2) * 0.6
		}
		from := t.Pos.Add(FromAngle(h).Mul(0.25))
		eff := effDamage(t.Def, tg.Def, st.Damage)
		w.Projectiles = append(w.Projectiles, &Projectile{
			ID: w.id(), Kind: ProjMissile, Def: t.Def, Start: from, Pos: from, Prev: from,
			TargetID: tg.ID, Target: tg.Def, Aim: tg.Pos, Speed: 2.2, MaxSpeed: st.ProjSpeed, Heading: h,
			Damage: st.Damage, eff: eff, Splash: st.Splash, Level: t.Level, Index: i, alive: true,
		})
		tg.pending += eff
	}
	w.emit(Event{Kind: EvShot, Pos: t.Pos, Angle: t.Angle, ID: t.ID, Level: t.Level, Value: n, Tower: t.Def})
}

// flameTick burns every ground enemy inside the fire cone.
func (w *World) flameTick(t *Tower, target *Enemy, st TowerLevel) {
	reach := st.Range + 0.15
	for _, e := range w.Enemies {
		if !e.Targetable() || e.Def.Flying {
			continue
		}
		d := e.Pos.Sub(t.Pos)
		l := d.Len()
		if l > reach || (l > 0.3 && math.Abs(AngleDiff(d.Angle(), t.Angle)) > t.Def.Cone) {
			continue
		}
		w.hurt(e, e.Def.EffDamage(st.Damage*e.Def.FireMul()))
		if e.alive {
			e.ignite(st.Burn, st.BurnTime)
		}
	}
	if t.HasSuper() && t.patchCD <= 0 {
		s := t.Def.Super
		t.patchCD = s.PatchEvery
		w.Patches = append(w.Patches, &Patch{ID: w.id(), Pos: target.Pos, R: s.PatchR, DPS: st.Burn, Life: s.PatchLife, Tower: t.ID})
		w.emit(Event{Kind: EvNapalm, Pos: target.Pos, ID: t.ID, R: s.PatchR})
	}
}

// fireRail shoots an instant beam; with the super it pierces the whole line.
func (w *World) fireRail(t *Tower, target *Enemy, st TowerLevel) {
	dir := FromAngle(t.Angle)
	muzzle := t.Pos.Add(dir.Mul(0.5))
	end := target.Pos
	pierce := t.HasSuper() && t.Def.Super.Pierce
	if pierce {
		end = t.Pos.Add(dir.Mul(st.Range + 0.4))
		var hit []*Enemy
		for _, e := range w.Enemies {
			if e.Targetable() && t.Def.CanTarget(e.Def) && (e == target || SegDist(e.Pos, muzzle, end) <= 0.34) {
				hit = append(hit, e)
			}
		}
		for _, e := range hit {
			w.emit(Event{Kind: EvHit, Pos: e.Pos, ID: e.ID, Level: t.Level, Angle: t.Angle, Enemy: e.Def, Tower: t.Def})
			w.damage(e, st.Damage)
		}
	} else {
		w.emit(Event{Kind: EvHit, Pos: target.Pos, ID: target.ID, Level: t.Level, Angle: t.Angle, Enemy: target.Def, Tower: t.Def})
		w.damage(target, effDamage(t.Def, target.Def, st.Damage))
	}
	w.emit(Event{Kind: EvRail, Pos: muzzle, To: end, Angle: t.Angle, ID: t.ID, Level: t.Level, Flag: pierce, Tower: t.Def, Enemy: target.Def})
}

// updatePulse sends a frost wave whenever a ground enemy is in range.
func (w *World) updatePulse(t *Tower) {
	st := t.Stats()
	if t.Cooldown > 0 {
		return
	}
	r2 := st.Range * st.Range
	any := false
	for _, e := range w.Enemies {
		if e.Targetable() && !e.Def.Flying && e.Pos.Dist2(t.Pos) <= r2 {
			any = true
			break
		}
	}
	if !any {
		t.Cooldown = 0
		return
	}
	t.Cooldown += 1 / st.FireRate
	t.ShotT = 0
	t.Pulses++
	s := t.Def.Super
	freeze := t.HasSuper() && s.FreezeEvery > 0 && t.Pulses%s.FreezeEvery == 0
	for _, e := range w.Enemies {
		if !e.Targetable() || e.Def.Flying || e.Pos.Dist2(t.Pos) > r2 {
			continue
		}
		slow := st.Slow
		if e.Def.Boss {
			slow *= 0.5
		}
		e.chill(slow, st.SlowTime)
		if freeze && !e.Def.Boss && !e.Def.NoChill {
			if e.FreezeT <= 0 {
				w.emit(Event{Kind: EvFreeze, Pos: e.Pos, ID: e.ID, Enemy: e.Def})
			}
			e.FreezeT = math.Max(e.FreezeT, s.FreezeTime)
		}
		if st.Damage > 0 {
			w.hurt(e, e.Def.EffDamage(st.Damage))
		}
	}
	w.emit(Event{Kind: EvPulse, Pos: t.Pos, ID: t.ID, Level: t.Level, R: st.Range, Flag: freeze, Tower: t.Def})
}

// --- projectiles -------------------------------------------------------------

type ProjKind int

const (
	ProjBolt ProjKind = iota
	ProjMissile
)

type Projectile struct {
	ID       int
	Kind     ProjKind
	Def      *TowerDef // turret type that fired it
	Start    Vec       // where it was fired from
	Pos      Vec
	Prev     Vec
	From     *EnemyDef // ricochets start at this enemy's body (nil: at a muzzle)
	TargetID int
	Target   *EnemyDef // type of the target (kept after it dies)
	Aim      Vec
	Speed    float64
	MaxSpeed float64 // missiles accelerate up to it
	Heading  float64 // missiles steer
	Damage   float64
	Splash   float64
	Bounces  int
	Level    int
	Index    int // missile number within a salvo
	Life     float64
	eff      float64 // damage reserved on the target
	hit      []int   // enemies a ricochet chain has hit already
	lost     bool    // a missile without a target flies on to its last aim point
	alive    bool
}

func (w *World) updateProjectiles(dt float64) {
	n := len(w.Projectiles) // ricochets spawned this step start moving next step
	for i := 0; i < n; i++ {
		p := w.Projectiles[i]
		if !p.alive {
			continue
		}
		p.Life += dt
		if p.Kind == ProjMissile {
			w.updateMissile(p, dt)
			continue
		}
		tgt := w.enemyByID(p.TargetID)
		if tgt != nil {
			p.Aim = tgt.Pos
		}
		p.Prev = p.Pos
		to := p.Aim.Sub(p.Pos)
		d := to.Len()
		stepLen := p.Speed * dt
		if d <= stepLen+0.08 {
			p.Pos = p.Aim
			p.alive = false
			if tgt != nil {
				tgt.pending -= p.eff
				w.emit(Event{Kind: EvHit, Pos: p.Pos, ID: tgt.ID, Level: p.Level, Angle: to.Angle(), Enemy: tgt.Def, Flag: tgt.Def.Armor > 0, Tower: p.Def})
				w.damage(tgt, effDamage(p.Def, tgt.Def, p.Damage))
				if p.Bounces > 0 {
					w.ricochet(p, tgt)
				}
			} else {
				w.emit(Event{Kind: EvFizzle, Pos: p.Pos, Level: p.Level, Enemy: p.Target, Tower: p.Def})
			}
			continue
		}
		p.Pos = p.Pos.Add(to.Mul(stepLen / d))
		if p.Life > 4 {
			p.alive = false
		}
	}
}

// ricochet sends a weaker bolt on to the nearest enemy the chain has not hit yet.
func (w *World) ricochet(p *Projectile, from *Enemy) {
	s := p.Def.Super
	hit := append(append([]int(nil), p.hit...), from.ID)
	var best *Enemy
	bd := s.BounceR * s.BounceR
	for _, e := range w.Enemies {
		if !e.Targetable() || !p.Def.CanTarget(e.Def) || e.HP-e.pending <= 0 || containsID(hit, e.ID) {
			continue
		}
		if d := e.Pos.Dist2(p.Pos); d <= bd {
			best, bd = e, d
		}
	}
	if best == nil {
		return
	}
	base := p.Damage
	if len(p.hit) > 0 {
		base /= s.BounceDmg // keep each ricochet at a share of the original bolt
	}
	q := w.newBolt(p.Def, p.Level, p.Pos, from.Def, best, base*s.BounceDmg, p.Speed)
	q.Bounces, q.hit = p.Bounces-1, hit
	w.emit(Event{Kind: EvBounce, Pos: p.Pos, ID: best.ID, Enemy: from.Def, Tower: p.Def})
}

func containsID(ids []int, id int) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func (w *World) updateMissile(p *Projectile, dt float64) {
	tgt := w.enemyByID(p.TargetID)
	if tgt == nil && !p.lost {
		// The target died: pick the nearest flying enemy close by, if any.
		var best *Enemy
		bd := 3.0 * 3.0
		for _, e := range w.Enemies {
			if e.Targetable() && p.Def.CanTarget(e.Def) && e.HP-e.pending > 0 {
				if d := e.Pos.Dist2(p.Pos); d < bd {
					best, bd = e, d
				}
			}
		}
		if best != nil {
			p.TargetID, p.Target = best.ID, best.Def
			p.eff = effDamage(p.Def, best.Def, p.Damage)
			best.pending += p.eff
			tgt = best
		} else {
			p.lost = true
		}
	}
	if tgt != nil {
		p.Aim = tgt.Pos
	}
	p.Speed = math.Min(p.MaxSpeed, p.Speed+9*dt)
	want := p.Aim.Sub(p.Pos).Angle()
	diff := AngleDiff(want, p.Heading)
	if turn := (2.5 + 9*p.Life) * dt; math.Abs(diff) <= turn {
		p.Heading = want
	} else {
		p.Heading = WrapAngle(p.Heading + math.Copysign(turn, diff))
	}
	p.Prev = p.Pos
	step := p.Speed * dt
	if p.Aim.Dist(p.Pos) <= step+0.1 {
		p.Pos = p.Aim
		w.explode(p, tgt)
		return
	}
	p.Pos = p.Pos.Add(FromAngle(p.Heading).Mul(step))
	if p.Life > 6 {
		w.explode(p, nil)
	}
}

// explode detonates a missile: full damage to its target, half to flyers nearby.
func (w *World) explode(p *Projectile, tgt *Enemy) {
	p.alive = false
	if tgt != nil {
		tgt.pending -= p.eff
		w.damage(tgt, effDamage(p.Def, tgt.Def, p.Damage))
	}
	r2 := p.Splash * p.Splash
	for _, e := range w.Enemies {
		if e.Targetable() && e != tgt && p.Def.CanTarget(e.Def) && e.Pos.Dist2(p.Pos) <= r2 {
			w.damage(e, effDamage(p.Def, e.Def, p.Damage*0.5))
		}
	}
	w.emit(Event{Kind: EvExplode, Pos: p.Pos, R: p.Splash, Enemy: p.Target, Tower: p.Def, Flag: tgt != nil})
}

// Patch is a pool of burning napalm on the road.
type Patch struct {
	ID      int
	Pos     Vec
	R       float64
	DPS     float64
	T, Life float64
	Tower   int
}

func (w *World) updatePatches(dt float64) {
	ps := w.Patches[:0]
	for _, p := range w.Patches {
		p.T += dt
		if p.T >= p.Life {
			continue
		}
		r2 := p.R * p.R
		for _, e := range w.Enemies {
			if e.Targetable() && !e.Def.Flying && e.Pos.Dist2(p.Pos) <= r2 {
				e.ignite(p.DPS, 0.8)
			}
		}
		ps = append(ps, p)
	}
	for i := len(ps); i < len(w.Patches); i++ {
		w.Patches[i] = nil
	}
	w.Patches = ps
}
