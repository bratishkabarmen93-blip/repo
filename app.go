package main

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"outpost/internal/art"
	"outpost/internal/game"
)

type Rect struct{ X, Y, W, H int }

func (r Rect) Has(x, y int) bool { return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H }

// Button is an immediate-mode UI button rebuilt every frame.
type Button struct {
	ID       string
	R        Rect
	Label    string
	Key      string
	Cost     int
	Enabled  bool
	Primary  bool
	Danger   bool // destructive action waiting for a confirm
	Icon     string
	CostIcon string // the icon in front of the cost ("cr" when empty)
}

type Banner struct {
	Title, Sub string
	Col        color.RGBA
	T, Life    float64
}

// pacer turns noisy frame timings into a stable time step.
type pacer struct {
	last time.Time
	avg  float64
}

func (p *pacer) tick() float64 {
	now := time.Now()
	if p.last.IsZero() {
		p.last = now
		return 1.0 / 60
	}
	dt := now.Sub(p.last).Seconds()
	p.last = now
	if dt > 0.25 {
		dt = 0.25
	}
	if p.avg == 0 {
		p.avg = dt
	} else {
		p.avg += (dt - p.avg) * 0.05
	}
	if math.Abs(dt-p.avg) < p.avg*0.35 {
		dt = p.avg // remove scheduler jitter for perfectly even motion
	}
	return math.Min(dt, 0.05)
}

// screen is the part of the game the player is in.
type screen int

const (
	scrMenu     screen = iota // title screen over the live backdrop
	scrChapters               // chapter select
	scrLevels                 // level select of a chapter
	scrGame                   // playing a level
	scrColony                 // the colonies
)

// panelKind is a window opened on top of a screen.
type panelKind int

const (
	panelNone panelKind = iota
	panelGuide
	panelSettings
	panelPrepare // supplies and orders of the expedition on offer
)

// card is a queued "new enemy" card or the lesson of a tutorial level.
type card struct {
	enemy  *game.EnemyDef
	lesson *game.Lesson
}

// result is the outcome of a finished level.
type result struct {
	stars, prev int
	unlocked    *game.LevelSpec // the level this win opened
	bonus       *game.LevelSpec // a bonus map this win opened (Skirmish)
	pay         game.Payout     // the crystals of the win
	colonized   *game.Location  // the world this win freed
	trophies    []*game.Trophy  // trophies won by the battle
}

type App struct {
	as    *Assets
	lvl   *game.Level
	spec  *game.LevelSpec // the level being played (or the menu backdrop)
	world *game.World
	cam   *Camera
	fx    *FX
	dbg   debugConfig
	save  *Save

	levels  map[string]*game.Level // parsed levels by ID
	demoLvl *game.Level
	scr     screen
	demo    bool // the world is the menu backdrop
	demoT   float64
	popup   panelKind     // window over the screen
	tab     int           // guide tab
	tabSel  [3]int        // selected guide entry per tab
	pick    int           // selected level card
	chap    int           // selected chapter card (len(game.Chapters): the skirmish)
	chapter *game.Chapter // chapter of the level select
	fadeIn  float64
	resetT  float64 // when "Reset progress" was armed
	res     result
	exp     *game.Expedition // the expedition on offer (made when first needed)
	expKey  string           // the seed, order, supplies and knowledge it was made from
	ord     game.ExpOrder    // the order being put together in the prepare window
	know    *game.Knowledge  // what the campaign has taught (see knowledge)
	knowKey string           // the levels won that it was gathered from

	sw, sh int
	dpi    float64
	u      int
	barH   int

	pace   pacer
	now    float64
	gameT  float64
	speed  int
	paused bool
	endT   float64
	runs   int

	sel       *game.Tower
	menuSlot  int     // empty pad with the build menu open, or -1
	menuT     float64 // seconds since the build menu opened
	menuKey   int     // build menu entry picked with a number key (highlight), -1
	sellArm   int     // ID of the turret whose Sell button waits for a confirm
	sellT     float64
	spin      map[int]float64 // cryo head angles
	hoverSlot int
	hoverBtn  string
	pressBtn  string
	buttons   []Button
	blockers  []Rect
	cursor    ebiten.CursorShapeType

	mx, my       int
	lastX, lastY int
	downX, downY int
	lmbDown      bool
	dragging     bool
	mmbDown      bool

	banner      *Banner
	seen        map[*game.EnemyDef]bool // enemy types already introduced (saved)
	cards       []card                  // queued cards, the first one is shown
	cardT       float64
	tipEnemy    *game.EnemyDef // wave preview entry under the cursor
	tipFrom     int            // and its entrance
	bossT       float64        // fade of the boss bar
	baseFlash   float64
	hpFlash     float64
	crFlash     float64
	crGain      float64
	lastCredits int
	spawnPulse  [3]float64 // per entrance
	heads       map[int]float64
	smokeT      float64
	fxTestT     float64
	quit        bool
	wasFocused  bool
	err         error
	op          ebiten.DrawImageOptions
	shadowImg   *ebiten.Image
	underImg    *ebiten.Image
	items       []drawItem
	enemies     []*game.Enemy
	wyrms       []wyrmLaid
	bossDef     *game.EnemyDef // the boss on the bar (kept while it fades out)
	col         colonyState
	br          bridgeState
	notes       []note // short messages at the bottom of the screen, one after another
	noteT       float64
}

// note is a toast: an icon and a line of text.
type note struct{ icon, text string }

func NewApp() (*App, error) {
	levels := map[string]*game.Level{}
	var list []*game.Level
	for _, sp := range game.AllLevels() {
		l, err := game.ParseLevel(sp)
		if err != nil {
			return nil, fmt.Errorf("level %s: %w", sp.ID, err)
		}
		levels[sp.ID] = l
		list = append(list, l)
	}
	demo, err := game.ParseLevel(demoSpec)
	if err != nil {
		return nil, err
	}
	as := NewAssets(list)
	a := &App{
		as:      as,
		levels:  levels,
		demoLvl: demo,
		fx:      NewFX(as),
		dbg:     loadDebug(),
		save:    loadSave(),
		speed:   1,
		heads:   map[int]float64{},
		spin:    map[int]float64{},
		seen:    map[*game.EnemyDef]bool{},
	}
	a.dbg.applySave(a.save)
	if n := a.save.payPast(); n > 0 {
		a.save.store()
		a.toast(fmt.Sprintf("+%d crystals for your past victories", n))
	}
	a.save.countPast()
	a.dbg.applyColony(a.save)
	if won := a.save.claimTrophies(); len(won) > 0 {
		a.trophyToast(won)
	}
	a.dbg.applyCrystals(a.save)
	a.keepBonusOpen()
	for _, id := range a.save.Seen {
		if d := game.EnemyDefs[id]; d != nil {
			a.seen[d] = true
		}
	}
	if s := game.LevelByID(a.dbg.level); s != nil {
		a.startLevel(s)
		return a, nil
	}
	if a.dbg.level == game.ExpeditionID && a.expOpen() {
		a.startExpedition()
		return a, nil
	}
	a.startDemo()
	a.chapter = game.Chapters[0]
	switch a.dbg.screen {
	case "chapters":
		a.openChapters()
	case "levels":
		a.openChapters()
		if c := chapterAt(a.chap); c != nil {
			a.showLevels(c)
		}
	case "colony":
		a.openColony()
	}
	return a, nil
}

// noteTime is how long a toast stays up (seconds).
const noteTime = 5

// toast shows a short message about crystals at the bottom of the screen.
func (a *App) toast(s string) { a.toastIcon("crystal", s) }

// toastIcon queues a toast with an icon; toasts show one after another.
func (a *App) toastIcon(icon, s string) {
	a.notes = append(a.notes, note{icon, s})
	if len(a.notes) == 1 {
		a.noteT = noteTime
	}
}

// trophyToast tells about trophies just won.
func (a *App) trophyToast(won []*game.Trophy) {
	pay := 0
	for _, t := range won {
		pay += t.Reward
	}
	if len(won) == 1 {
		a.toastIcon("trophy", fmt.Sprintf("Trophy won: %s · +%d crystals", won[0].Name, pay))
		return
	}
	a.toastIcon("trophy", fmt.Sprintf("%d trophies won · +%d crystals", len(won), pay)+bridgeHint)
}

// bridgeHint ends the toast of new trophies, unless it is read on the bridge.
const bridgeHint = " · see them on the bridge"

// bankRun keeps the record of a battle that is left before its end (kills,
// supers) and hands out the trophies it has won.
func (a *App) bankRun() {
	if a.world == nil || a.demo || a.world.Phase == game.PhaseWon || a.world.Phase == game.PhaseLost {
		return
	}
	a.save.store()
	if won := a.save.claimTrophies(); len(won) > 0 {
		a.trophyToast(won)
	}
}

// setTerrain shows the ground of a level and frames it with a fresh camera.
func (a *App) setTerrain(id string) {
	a.as.UseTerrain(id)
	tb := a.as.Terrain.Bounds()
	a.cam = NewCamera(float64(tb.Dx()), float64(tb.Dy()))
	a.applyView()
}

// applyView fits the camera to the window: the backdrop uses the whole
// screen, a level leaves room for the HUD bar.
func (a *App) applyView() {
	if a.cam == nil || a.sw == 0 {
		return
	}
	if a.scr == scrColony {
		v := a.colonyView()
		a.cam.SetViewRect(float64(v.X), float64(v.Y), float64(v.W), float64(v.H), float64(4*a.u))
		return
	}
	if a.demo {
		a.cam.SetView(float64(a.sw), float64(a.sh), 0, 0)
		return
	}
	a.cam.SetView(float64(a.sw), float64(a.sh), float64(a.barH), float64(6*a.u))
}

// startLevel begins playing a level.
func (a *App) startLevel(s *game.LevelSpec) {
	a.spec = s
	a.lvl = a.levels[s.ID]
	a.demo = false
	a.scr = scrGame
	a.popup = panelNone
	a.fadeIn = 1
	a.setTerrain(s.ID)
	if a.save.Last != s.ID {
		a.save.Last = s.ID
		a.save.store()
	}
	a.restart()
}

func (a *App) restart() {
	a.bankRun()
	a.runs++
	a.world = game.NewWorld(a.lvl, int64(a.runs)*7919+1)
	a.fx.Reset()
	a.sel = nil
	a.menuSlot, a.menuKey, a.sellArm = -1, -1, 0
	clear(a.spin)
	clear(a.heads)
	a.paused = false
	a.endT = 0
	a.banner = nil
	a.baseFlash, a.hpFlash, a.crFlash, a.crGain = 0, 0, 0, 0
	a.cards, a.cardT, a.bossT = a.cards[:0], 0, 0
	a.res = result{}
	if a.demo {
		a.setupDemo()
		a.lastCredits = a.world.Credits
		return
	}
	s := a.spec
	if a.dbg.win {
		a.world.DebugWin()
	}
	if a.dbg.wave > 0 {
		a.world.DebugStartAt(a.dbg.wave, a.dbg.credits)
	} else if a.dbg.credits > 0 {
		a.world.Credits = a.dbg.credits
	}
	if s.Lesson != nil && a.best(s) == 0 && a.dbg.wave == 0 {
		// The briefing of a new tutorial level holds the first wave until it is read.
		a.cards = append(a.cards, card{lesson: s.Lesson})
		a.world.Hold = true
	}
	a.banner = &Banner{Title: s.Name, Sub: levelSub(s), Col: art.Cyan, Life: 2.6}
	a.lastCredits = a.world.Credits
}

// levelSub is the line under a level name: "Level 1-2" or "Skirmish".
func levelSub(s *game.LevelSpec) string {
	if isExpedition(s) {
		return "Expedition · " + expPayNote(s)
	}
	if c, _ := chapterOf(s); c != nil {
		return "Level " + s.ID
	}
	return "Skirmish · every turret unlocked"
}

func (a *App) recoverTo(err *error) {
	if r := recover(); r != nil {
		*err = crashed(r)
		a.err = *err
	}
}

func (a *App) Layout(w, h int) (int, int) { return w, h }

func (a *App) LayoutF(w, h float64) (float64, float64) {
	s := 1.0
	if m := ebiten.Monitor(); m != nil {
		s = m.DeviceScaleFactor()
	}
	if s <= 0 {
		s = 1
	}
	sw, sh := int(math.Ceil(w*s)), int(math.Ceil(h*s))
	if sw < 1 {
		sw = 1
	}
	if sh < 1 {
		sh = 1
	}
	if sw != a.sw || sh != a.sh || s != a.dpi {
		a.sw, a.sh, a.dpi = sw, sh, s
		a.u = max(1, int(math.Round(math.Min(float64(sw)/640, float64(sh)/360)*0.8)))
		a.barH = 19 * a.u
		a.applyView()
	}
	return float64(sw), float64(sh)
}

func (a *App) Update() (err error) {
	defer a.recoverTo(&err)
	if a.err != nil {
		return a.err
	}
	if a.quit {
		return ebiten.Termination
	}
	dt := a.pace.tick()
	a.now += dt
	if a.sw == 0 {
		return nil
	}
	a.layoutUI()
	a.handleKeys()
	if a.quit {
		return ebiten.Termination
	}
	// Pause automatically when the window loses focus (only on a real focus change).
	focused := ebiten.IsFocused()
	if a.autoPause() && a.wasFocused && !focused && a.playing() {
		a.paused = true
	}
	a.wasFocused = focused
	a.handleMouse()
	if len(a.notes) > 0 {
		if a.noteT -= dt; a.noteT <= 0 {
			if a.notes = a.notes[1:]; len(a.notes) > 0 {
				a.noteT = noteTime
			}
		}
	}
	if a.scr == scrColony {
		a.cam.Update(dt)
		a.updateColony(dt)
		a.fx.Update(dt)
		a.layoutUI()
		publishState(a)
		return nil
	}
	if a.demo {
		a.cinematic(dt)
	} else {
		a.cam.Update(dt)
	}
	gdt := dt * a.timeScale()
	if gdt > 0 && a.world.Active() {
		a.world.Update(gdt)
	}
	a.handleEvents()
	a.fx.Update(gdt)
	a.gameT += gdt
	a.updateTimers(dt, gdt)
	if a.demo && !a.world.Active() && a.endT > 1.5 {
		a.restart() // the backdrop battle starts over
	}
	a.layoutUI()
	publishState(a)
	return nil
}

// timeScale is how fast the world runs: the backdrop always moves, a level
// stops while it is paused or a window is open.
func (a *App) timeScale() float64 {
	switch {
	case a.demo:
		return 1
	case a.scr != scrGame || a.paused || a.popup != panelNone:
		return 0
	}
	return float64(a.speed) * a.dbg.speed
}

// playing reports whether a level is running with nothing in front of it.
func (a *App) playing() bool {
	return a.scr == scrGame && !a.demo && a.world.Active() && !a.paused && a.popup == panelNone
}

func (a *App) autoPause() bool { return !a.save.NoAutoPause && !a.dbg.noAutoPause }

func (a *App) updateTimers(dt, gdt float64) {
	decay := func(v *float64, rate float64) {
		*v = math.Max(0, *v-rate*dt)
	}
	decay(&a.baseFlash, 3)
	decay(&a.hpFlash, 2)
	decay(&a.crFlash, 2)
	decay(&a.crGain, 4)
	if a.world.Credits > a.lastCredits {
		a.crGain = 1
	}
	a.lastCredits = a.world.Credits
	for i := range a.spawnPulse {
		decay(&a.spawnPulse[i], 3)
	}
	decay(&a.fadeIn, 2.5)
	if len(a.cards) > 0 {
		a.cardT += gdt
		if a.cards[0].lesson == nil && a.cardT > cardLife {
			a.dismissCard()
		}
	}
	if b := a.world.Boss(); b != nil {
		a.bossDef = b.Def
		a.bossT = math.Min(1, a.bossT+dt*3)
	} else {
		a.bossT = math.Max(0, a.bossT-dt*2)
	}
	if a.banner != nil {
		a.banner.T += dt
		if a.banner.T >= a.banner.Life {
			a.banner = nil
		}
	}
	if !a.world.Active() {
		a.endT += dt
	}
	bc := a.lvl.BaseCenter()
	bx, by := bc.X*art.CellPx, bc.Y*art.CellPx
	if a.dbg.fxTest {
		a.fxTestT -= dt
		if a.fxTestT < 0 {
			a.fxTestT = 1.5
			a.fx.Explosion(400, 208, 10, 0)
			a.fx.Kill(a.as.Looks[game.Crawler], 300, 208, 4, 0, 0, 0)
			a.fx.HitSparks(500, 208, 5, 0, 0, 2, false)
			a.fx.Build(600, 208, art.PadH)
			a.fx.Upgrade(200, 208, art.PadH, 2)
		}
	}
	switch a.world.Phase {
	case game.PhaseLost:
		if a.demo {
			break
		}
		a.smokeT -= dt
		for a.smokeT < 0 {
			a.smokeT += 0.07
			a.fx.Smoke(bx, by, art.BaseTopZ)
		}
	case game.PhaseWon:
		if a.endT < 2.5 && !a.demo {
			a.smokeT -= dt
			for a.smokeT < 0 {
				a.smokeT += 0.04
				a.fx.Sparkle(bx, by, art.BaseTopZ)
			}
		}
	}
	a.updateTowerFX(dt, gdt)
	a.updateWyrmFX(gdt)
	// Smooth enemy headings so turning at corners never snaps.
	for _, e := range a.world.Enemies {
		want := e.Dir.Angle()
		cur, ok := a.heads[e.ID]
		if !ok {
			a.heads[e.ID] = want
			continue
		}
		d := game.AngleDiff(want, cur)
		a.heads[e.ID] = game.WrapAngle(cur + d*(1-math.Exp(-gdt*14)))
	}
}

const cardLife = 20.0 // seconds of game time a "new enemy" card stays up

func (a *App) handleEvents() {
	w := a.world
	for _, ev := range w.Events {
		x, y := ev.Pos.X*art.CellPx, ev.Pos.Y*art.CellPx
		g := a.ground(x, y)
		lk := a.as.Looks[ev.Enemy]
		bodyZ := 4.5
		if lk != nil {
			bodyZ = lk.bodyZ + lk.hover
		}
		switch ev.Kind {
		case game.EvSpawn:
			if !ev.Flag {
				a.spawnPulse[min(ev.Value, len(a.spawnPulse)-1)] = 1
				a.fx.Spawn(x, y, g+3, g)
			}
			if a.demo {
				break
			}
			if !a.seen[ev.Enemy] {
				a.markSeen(ev.Enemy)
				if !ev.Flag { // offspring are explained on their parent's card
					a.cards = append(a.cards, card{enemy: ev.Enemy})
				}
			}
			if ev.Enemy.Boss {
				col := art.Magenta
				if isWyrm(ev.Enemy) {
					col = art.Cyan
				}
				a.banner = &Banner{Title: ev.Enemy.Name, Sub: "The boss has emerged from the burrow", Col: col, Life: 3}
			}
		case game.EvHit:
			switch {
			case ev.Tower == game.Railgun:
				a.fx.RailSparks(x, y, g+bodyZ, g, ev.Angle)
			default:
				a.fx.HitSparks(x, y, g+bodyZ, g, ev.Angle, min(ev.Level, 2), ev.Flag)
			}
		case game.EvFizzle:
			a.fx.Fizzle(x, y, g+bodyZ, g)
		case game.EvKill:
			h := angleIndex(a.heads[ev.ID], enemyAngles)
			if isWyrm(ev.Enemy) {
				a.wyrmKill(ev)
			} else {
				a.fx.Kill(lk, x, y, g+bodyZ, g, h, 0)
			}
			if !a.demo {
				a.fx.Float(x, y, g+lk.hover+lk.topZ+2, fmt.Sprintf("+%d", ev.Value), art.Yellow, 0.8)
				a.save.Kills[ev.Enemy.ID]++
			}
			delete(a.heads, ev.ID)
		case game.EvLeak:
			if !a.demo {
				a.baseFlash = 1
				a.hpFlash = 1
				if pct := ev.Value * 100 / max(1, w.BaseMaxHP); pct > 0 {
					bc := a.lvl.BaseCenter()
					a.fx.LeakFloat(bc.X*art.CellPx, bc.Y*art.CellPx, art.DomeCenter.Z+30, pct)
				}
			}
			a.fx.Leak(x, y, g+bodyZ, g, math.Max(1, lk.size))
			delete(a.heads, ev.ID)
		case game.EvHeal:
			a.fx.RepairPulse(x, y, g, ev.Enemy.Heal.Radius*art.CellPx, ev.Value > 0)
		case game.EvHealed:
			if src := a.healerNear(ev.Pos); src != 0 {
				a.fx.Repaired(src, ev.ID, x, y, g+bodyZ)
			}
		case game.EvBrood:
			back := game.FromAngle(ev.Angle).Mul(-0.6)
			bx, by := x+back.X*art.CellPx, y+back.Y*art.CellPx
			a.fx.Brood(bx, by, a.ground(bx, by))
		case game.EvSplit:
			a.fx.Split(x, y, g+bodyZ, g)
		case game.EvDive, game.EvSurface:
			rt := a.lvl.Routes[ev.Route]
			hd := nearestHole(rt, ev.Dist)
			hp, _ := rt.Path.At(hd)
			hx, hy := hp.X*art.CellPx, hp.Y*art.CellPx
			hg := a.ground(hx, hy)
			a.fx.Splash(hx, hy, hg)
			for i := 0; i < ev.Value; i++ {
				// Surfacing, it spits splitters onto the road ahead.
				sp, _ := rt.Path.At(math.Min(ev.Dist+1.1+0.45*float64(i), rt.Path.Len-1))
				sx, sy := sp.X*art.CellPx, sp.Y*art.CellPx
				a.fx.Spit(x, y, g+bodyZ+2, sx, sy, a.ground(sx, sy))
			}
		case game.EvWaveStart:
			if a.demo {
				break
			}
			spec := w.Lvl.Spec.Waves[ev.Value-1]
			title := fmt.Sprintf("Wave %d", ev.Value)
			sub := fmt.Sprintf("%d enemies incoming", spec.Total())
			col := art.Amber
			if ev.Value == w.TotalWaves() {
				title = "Final wave"
				switch {
				case spec.Has(game.Queen):
					sub = "The Hive Queen is coming"
					col = art.Magenta
				case spec.Has(game.Wyrm):
					sub = "The Ice Wyrm is coming"
					col = art.Cyan
				}
			}
			a.banner = &Banner{Title: title, Sub: sub, Col: col, Life: 2.4}
		case game.EvWaveClear:
			if ev.Level < w.TotalWaves() && !a.demo {
				a.banner = &Banner{Title: "Wave cleared", Sub: fmt.Sprintf("Bonus +%d credits", ev.Value), Col: art.Lime, Life: 2.4}
			}
		case game.EvWon:
			a.sel = nil
			a.banner = nil
			if !a.demo {
				a.recordWin()
			}
		case game.EvLost:
			a.sel = nil
			a.banner = nil
			if !a.demo {
				a.save.store()
				a.res.trophies = a.save.claimTrophies()
			}
			bc := a.lvl.BaseCenter()
			a.fx.Explosion(bc.X*art.CellPx, bc.Y*art.CellPx, art.BaseTopZ, 0)
			a.baseFlash = 1
		case game.EvBuild, game.EvUpgradeStart:
			a.fx.Rings = append(a.fx.Rings, Ring{X: x, Y: y, Z: art.PadH, R0: 20, R1: 12, Life: 0.3, Width: 1.5, Col: art.Cyan})
		case game.EvBuilt:
			a.fx.Build(x, y, art.PadH)
		case game.EvUpgrade:
			a.fx.Upgrade(x, y, art.PadH, ev.Level)
			if ev.Level == len(ev.Tower.Levels) {
				if !a.demo && !slices.Contains(a.save.Supers, ev.Tower.ID) {
					a.save.Supers = append(a.save.Supers, ev.Tower.ID)
				}
				lk := a.as.Towers[ev.Tower]
				a.fx.Rings = append(a.fx.Rings, Ring{X: x, Y: y, Z: art.PadH, R0: 8, R1: 40, Life: 0.7, Width: 3, Col: lk.accent})
				a.fx.Float(x, y, towerTopZ+4, ev.Tower.Super.Name+"!", lk.accent, 1.4)
			}
		case game.EvCancel:
			a.fx.Float(x, y, towerTopZ+4, fmt.Sprintf("+%d", ev.Value), art.Yellow, 1)
			if ev.Flag {
				a.fx.Cancelled(x, y, art.PadH)
				a.towerGone(ev.ID)
			}
		case game.EvSell:
			a.fx.Sold(x, y, art.PadH)
			a.fx.Float(x, y, towerTopZ+4, fmt.Sprintf("+%d", ev.Value), art.Yellow, 1)
			a.towerGone(ev.ID)
		case game.EvShot:
			a.towerShot(ev)
		case game.EvPulse:
			a.fx.FrostPulse(x, y, art.PadH+art.PivotZ+5, g, ev.R*art.CellPx, ev.Flag)
		case game.EvFreeze:
			a.fx.FreezeBurst(x, y, g+bodyZ, g)
		case game.EvRail:
			a.railBeam(ev)
		case game.EvExplode:
			z := g + bodyZ
			if ev.Enemy == nil || !ev.Flag {
				z = g + 12
			}
			a.fx.MissileBlast(x, y, z, g, ev.R*art.CellPx)
		case game.EvNapalm:
			a.fx.NapalmSplash(x, y, g)
			a.fx.decal(Decal{X: x, Y: y, Z: g, Img: a.as.Scorch[a.fx.rng.Intn(len(a.as.Scorch))], S: 0.55, Flip: a.fx.rng.Intn(2) == 0, Life: 8})
		case game.EvGrab:
			tx, ty := ev.To.X*art.CellPx, ev.To.Y*art.CellPx
			tg := a.ground(tx, ty)
			grabStart[ev.ID] = a.gameT
			a.fx.Burst(tx, ty, tg+art.PadH+7, tg, 10, []color.RGBA{art.Purple, art.Magenta, art.Pink}, 55, 30, 110, 0.35, 1.5, false, 0, math.Pi)
		case game.EvPuff:
			a.fx.Burst(x, y, g+5, g, 12, []color.RGBA{art.Tan, art.Sand, art.Peach}, 30, 25, 30, 0.8, 1.8, false, 0, math.Pi)
		case game.EvBlast:
			a.fx.MissileBlast(x, y, g+bodyZ, g, ev.R*art.CellPx)
			a.fx.decal(Decal{X: x, Y: y, Z: g, Img: a.as.Scorch[a.fx.rng.Intn(len(a.as.Scorch))], S: 0.7, Flip: a.fx.rng.Intn(2) == 0, Life: 8})
		case game.EvHide, game.EvReveal:
			a.fx.Burst(x, y, g+bodyZ, g, 6, []color.RGBA{art.Cyan, art.Silver, art.White}, 40, 20, 60, 0.3, 1.2, true, 0, math.Pi)
		case game.EvBounce:
			a.fx.Burst(x, y, g+bodyZ, g, 5, []color.RGBA{art.Cyan, art.White}, 60, 30, 120, 0.25, 1.4, true, 0, math.Pi)
		}
	}
	w.Events = w.Events[:0]
}

// healerNear finds the living healer closest to a repaired enemy.
func (a *App) healerNear(p game.Vec) int {
	best, bd := 0, math.Inf(1)
	for _, e := range a.world.Enemies {
		if e.Alive() && e.Def.Heal != nil {
			if d := e.Pos.Dist2(p); d < bd {
				best, bd = e.ID, d
			}
		}
	}
	return best
}

func (a *App) dismissCard() {
	if len(a.cards) == 0 {
		return
	}
	if a.cards[0].lesson != nil {
		a.world.Hold = false // the briefing was read: start the countdown
	}
	a.cards = a.cards[1:]
	a.cardT = 0
}

// markSeen remembers that the player has met an enemy type.
func (a *App) markSeen(d *game.EnemyDef) {
	a.seen[d] = true
	if !a.save.hasSeen(d) {
		a.save.Seen = append(a.save.Seen, d.ID)
		a.save.store()
	}
}

// recordWin saves the stars of a won level and notes what it unlocked.
func (a *App) recordWin() {
	s := a.spec
	if isExpedition(s) {
		a.recordExpedition()
		return
	}
	nx := nextLevel(s)
	wasLocked := nx != nil && !a.unlocked(nx)
	a.res = result{stars: a.world.Stars(), prev: a.best(s)}
	// Crystals: the first time a star is won it pays in full, later a little.
	if a.save.Paid == nil {
		a.save.Paid = map[string]int{}
	}
	paid := a.save.Paid[s.ID]
	a.res.pay = game.Pay(s, a.res.stars, paid)
	a.save.Paid[s.ID] = max(paid, a.res.stars)
	a.save.Crystals += a.res.pay.Total
	a.save.Earned += a.res.pay.Total
	if a.res.stars > a.res.prev {
		a.save.Stars[s.ID] = a.res.stars
	}
	w := a.world
	a.save.Wins++
	if w.BaseHP >= w.BaseMaxHP {
		a.save.Flawless = true
	}
	if w.BaseHP*10 <= w.BaseMaxHP {
		a.save.Close = true
	}
	a.save.store()
	a.res.trophies = a.save.claimTrophies()
	for _, l := range game.Colonies {
		if l.Level == s && a.res.prev == 0 {
			a.res.colonized = l
		}
	}
	if wasLocked && a.unlocked(nx) {
		a.res.unlocked = nx
	}
	if a.keepBonusOpen() {
		a.res.bonus = game.Skirmish
	}
}

func (a *App) handleKeys() {
	jp := inpututil.IsKeyJustPressed
	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	if jp(ebiten.KeyF11) || (jp(ebiten.KeyEnter) && alt) {
		a.toggleFullscreen()
	}
	enter := (jp(ebiten.KeyEnter) && !alt) || jp(ebiten.KeyNumpadEnter)
	a.cam.SetKeyInput(0, 0)
	if a.popup != panelNone {
		a.panelKeys()
		return
	}
	switch a.scr {
	case scrMenu:
		if enter {
			a.openChapters()
		}
		if jp(ebiten.KeyC) {
			a.openColony()
		}
		if jp(ebiten.KeyQ) {
			a.quit = true
		}
		return
	case scrColony:
		a.colonyKeys(enter)
		return
	case scrChapters:
		a.chapterKeys(enter)
		return
	case scrLevels:
		a.levelKeys(enter)
		return
	}
	if !a.world.Active() {
		if a.endT < endKeysAfter {
			return
		}
		isExp, expWon := isExpedition(a.spec), a.world.Phase == game.PhaseWon
		switch {
		case isExp && expWon && (enter || jp(ebiten.KeyN)):
			a.startExpedition() // the next one: a won expedition is done
		case isExp && expWon && jp(ebiten.KeyR):
			// A won expedition is not replayed.
		case isExp && jp(ebiten.KeyN):
			a.swapExpedition()
		case enter:
			if nx := a.nextPlayable(); nx != nil {
				a.startLevel(nx)
			} else {
				a.restart()
			}
		case jp(ebiten.KeyR):
			a.restart()
		case jp(ebiten.KeyEscape) || jp(ebiten.KeyM):
			a.openLevels()
		case jp(ebiten.KeyC) && a.res.colonized != nil && !a.res.colonized.Soon:
			a.openColonyAt(a.res.colonized)
		case jp(ebiten.KeyB) && len(a.res.trophies) > 0:
			a.openColonyAt(nil)
		}
		return
	}
	if a.paused {
		switch {
		case jp(ebiten.KeyEscape) || jp(ebiten.KeyP):
			a.paused = false
		case jp(ebiten.KeyR):
			a.restart()
		case jp(ebiten.KeyI):
			a.openPanel(panelGuide)
		case jp(ebiten.KeyM):
			a.openLevels()
		case jp(ebiten.KeyQ):
			a.bankRun()
			a.quit = true
		}
		return
	}
	if enter && len(a.cards) > 0 {
		a.dismissCard()
	}
	if jp(ebiten.KeyEscape) {
		switch {
		case a.menuSlot >= 0:
			a.closeMenu()
		case a.sel != nil:
			a.sel = nil
		default:
			a.paused = true
		}
	}
	if jp(ebiten.KeyP) {
		a.paused = true
	}
	if jp(ebiten.KeyI) {
		a.openPanel(panelGuide)
	}
	if jp(ebiten.KeyHome) {
		a.cam.Home()
	}
	if jp(ebiten.KeySpace) {
		a.skipTimer()
	}
	if jp(ebiten.KeyU) {
		a.upgradeSelected()
	}
	if jp(ebiten.KeyX) || jp(ebiten.KeyDelete) {
		a.sellOrCancel()
	}
	if jp(ebiten.KeyT) {
		a.cycleMode()
	}
	for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5} {
		if jp(k) || jp(ebiten.KeyNumpad1+ebiten.Key(i)) {
			a.quickBuild(i)
		}
	}
	if jp(ebiten.KeyF) {
		a.toggleSpeed()
	}
	pr := ebiten.IsKeyPressed
	kx, ky := 0.0, 0.0
	if pr(ebiten.KeyA) || pr(ebiten.KeyArrowLeft) {
		kx--
	}
	if pr(ebiten.KeyD) || pr(ebiten.KeyArrowRight) {
		kx++
	}
	if pr(ebiten.KeyW) || pr(ebiten.KeyArrowUp) {
		ky--
	}
	if pr(ebiten.KeyS) || pr(ebiten.KeyArrowDown) {
		ky++
	}
	if kx != 0 && ky != 0 {
		kx, ky = kx*math.Sqrt2/2, ky*math.Sqrt2/2
	}
	a.cam.SetKeyInput(kx, ky)
}

// endKeysAfter is how long the result screen ignores keys (seconds).
const endKeysAfter = 0.8

// nextPlayable is the level the "Next level" button starts after a win, or nil.
func (a *App) nextPlayable() *game.LevelSpec {
	if a.world.Phase != game.PhaseWon {
		return nil
	}
	nx := nextLevel(a.spec)
	if c, i := chapterOf(a.spec); nx == nil && c != nil && c.Num == len(game.Chapters) && i == len(c.Levels)-1 {
		nx = game.Skirmish // the end of the campaign leads to the bonus map
	}
	if nx != nil && a.unlocked(nx) {
		return nx
	}
	return nil
}

func (a *App) toggleFullscreen() {
	fs := !ebiten.IsFullscreen()
	ebiten.SetFullscreen(fs)
	a.save.Fullscreen = fs
	a.save.store()
}

// modal reports whether something covers the map, so it cannot be used.
func (a *App) modal() bool {
	return a.scr != scrGame || a.popup != panelNone || a.paused || !a.world.Active()
}

func (a *App) overBlocker(x, y int) bool {
	for _, r := range a.blockers {
		if r.Has(x, y) {
			return true
		}
	}
	return false
}

func (a *App) button(id string) *Button {
	for i := range a.buttons {
		if a.buttons[i].ID == id {
			return &a.buttons[i]
		}
	}
	return nil
}

func (a *App) handleMouse() {
	mx, my := ebiten.CursorPosition()
	a.mx, a.my = mx, my
	a.hoverBtn = ""
	for _, b := range a.buttons {
		if b.R.Has(mx, my) {
			a.hoverBtn = b.ID
			break
		}
	}
	overUI := a.hoverBtn != "" || a.overBlocker(mx, my)
	colony := a.scr == scrColony && a.popup == panelNone
	modal := a.modal()
	if colony {
		// Only the map of the colony screen can be zoomed, dragged and clicked.
		modal = !a.colonyView().Has(mx, my)
	}

	if _, wy := ebiten.Wheel(); wy != 0 && !modal {
		a.cam.ZoomAt(float64(mx), float64(my), wy)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		a.pressBtn = a.hoverBtn
		if a.pressBtn == "" && !overUI && !modal {
			a.lmbDown, a.dragging = true, false
			a.downX, a.downY = mx, my
		}
	}
	if a.lmbDown && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !a.dragging && math.Hypot(float64(mx-a.downX), float64(my-a.downY)) > 6*a.dpi {
			a.dragging = true
			a.cam.Pan(float64(mx-a.downX), float64(my-a.downY))
		} else if a.dragging {
			a.cam.Pan(float64(mx-a.lastX), float64(my-a.lastY))
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if a.pressBtn != "" && a.pressBtn == a.hoverBtn {
			if b := a.button(a.pressBtn); b != nil && b.Enabled {
				a.press(b.ID)
			} else if b != nil && b.ID == "upgrade" && a.sel != nil && !a.sel.MaxLevel() && !a.sel.Busy {
				a.notEnough(a.sel.Pos)
			} else if b != nil && strings.HasPrefix(b.ID, "build:") && a.menuSlot >= 0 {
				a.notEnough(a.lvl.Slots[a.menuSlot].Center())
			} else if b != nil && b.ID == "cbuild" && colony {
				a.col.flash = 1
			}
		}
		if a.lmbDown && !a.dragging {
			if colony && a.onBridge() {
				a.bridgeClick(mx, my)
			} else if colony {
				a.colonyClick(mx, my)
			} else {
				a.worldClick(mx, my)
			}
		}
		a.pressBtn, a.lmbDown, a.dragging = "", false, false
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) && !modal {
		a.mmbDown = true
	}
	if a.mmbDown {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			a.cam.Pan(float64(mx-a.lastX), float64(my-a.lastY))
		} else {
			a.mmbDown = false
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && !modal {
		a.sel = nil
		a.closeMenu()
		a.col.sel = nil
		a.br.sel = nil
	}
	a.lastX, a.lastY = mx, my

	a.hoverSlot = -1
	if colony {
		a.col.hover = nil
		a.br.hover, a.br.table = nil, false
		free := !overUI && !modal && !a.dragging && !a.mmbDown
		switch {
		case a.onBridge() && free:
			a.br.hover, a.br.table = a.trophyAt(mx, my)
		case a.onBridge():
			if n, ok := cutIndex(a.hoverBtn, "tsel:"); ok && n < len(game.AllTrophies) {
				a.br.hover = game.AllTrophies[n]
			}
		case free:
			a.col.hover = a.lotAt(mx, my)
		default:
			if n, ok := cutIndex(a.hoverBtn, "csel:"); ok && n < len(a.colonyLoc().Buildings) {
				a.col.hover = a.colonyLoc().Buildings[n]
			}
		}
	} else if !overUI && !modal && !a.dragging && !a.mmbDown && a.scr == scrGame {
		a.hoverSlot = a.slotAt(mx, my)
	}
	shape := ebiten.CursorShapeDefault
	switch {
	case a.dragging || a.mmbDown:
		shape = ebiten.CursorShapeMove
	case a.hoverBtn != "":
		shape = ebiten.CursorShapePointer
	case a.hoverSlot >= 0 && a.world.Active():
		shape = ebiten.CursorShapePointer
	case colony && a.col.hover != nil && !a.colonyLocked(a.colonyLoc()):
		shape = ebiten.CursorShapePointer
	case colony && a.br.hover != nil:
		shape = ebiten.CursorShapePointer
	}
	if shape != a.cursor {
		ebiten.SetCursorShape(shape)
		a.cursor = shape
	}
}

func (a *App) press(id string) {
	if a.scr == scrColony && a.popup == panelNone && a.pressColony(id) {
		return
	}
	switch id {
	case "colony":
		if a.scr == scrGame && a.res.colonized != nil {
			a.openColonyAt(a.res.colonized)
		} else {
			a.openColony()
		}
	case "bridge":
		a.openColonyAt(nil)
	case "skip":
		a.skipTimer()
	case "speed":
		a.toggleSpeed()
	case "pause":
		a.paused = true
	case "resume":
		a.paused = false
	case "restart":
		a.restart()
	case "expnext":
		a.startExpedition()
	case "expnew":
		a.swapExpedition()
	case "next":
		if nx := a.nextPlayable(); nx != nil {
			a.startLevel(nx)
		}
	case "levels":
		a.openLevels()
	case "quit":
		a.bankRun()
		a.quit = true
	case "play":
		a.openChapters()
	case "back":
		if a.scr == scrLevels {
			a.scr = scrChapters
		} else {
			a.scr = scrMenu
		}
	case "open":
		a.enterChapter()
	case "prep":
		a.openPrepare()
	case "order":
		a.placeOrder()
	case "start":
		a.playPick()
	case "guide":
		a.openPanel(panelGuide)
	case "settings":
		a.openPanel(panelSettings)
	case "close":
		a.closePanel()
	case "fullscreen":
		a.toggleFullscreen()
	case "autopause":
		a.save.NoAutoPause = !a.save.NoAutoPause
		a.save.store()
	case "reset":
		a.resetProgress()
	case "upgrade":
		a.upgradeSelected()
	case "sell", "cancel":
		a.sellOrCancel()
	case "target":
		a.cycleMode()
	case "gotit":
		a.dismissCard()
	default:
		if id, ok := strings.CutPrefix(id, "build:"); ok && a.menuSlot >= 0 {
			a.buildAt(a.menuSlot, game.TowerDefs[id])
		} else if id, ok := strings.CutPrefix(id, "sup:"); ok {
			a.buySupply(id)
		} else if part, ok := strings.CutPrefix(id, "ord:"); ok {
			a.cycleOrder(part)
		} else if n, ok := cutIndex(id, "lv:"); ok {
			a.pickCard(n)
		} else if n, ok := cutIndex(id, "ch:"); ok {
			a.pickChapter(n)
		} else if n, ok := cutIndex(id, "tab:"); ok {
			a.tab = n
		} else if n, ok := cutIndex(id, "entry:"); ok {
			a.tabSel[a.tab] = n
		}
	}
}

// cutIndex parses IDs like "lv:3".
func cutIndex(id, prefix string) (int, bool) {
	rest, ok := strings.CutPrefix(id, prefix)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

func (a *App) worldClick(mx, my int) {
	if !a.world.Active() {
		return
	}
	slot := a.slotAt(mx, my)
	if slot < 0 {
		a.sel = nil
		a.closeMenu()
		return
	}
	if t := a.world.SlotTower[slot]; t != nil {
		a.closeMenu()
		a.selectTower(t)
		return
	}
	if a.menuSlot == slot {
		a.closeMenu()
		return
	}
	a.sel = nil
	a.menuSlot, a.menuT, a.menuKey = slot, 0, -1
	// Bring a menu that would stick out of the screen into view.
	if bm, ok := a.buildMenuLayout(); ok && (bm.dx != 0 || bm.dy != 0) {
		a.cam.Nudge(float64(bm.dx), float64(bm.dy))
	}
}

// cycleMode switches the selected turret to its next targeting mode.
func (a *App) cycleMode() {
	t := a.sel
	if t == nil || !a.world.Active() || !t.Def.Aims() || !a.world.CanTarget() {
		return
	}
	a.world.CycleMode(t)
}

func (a *App) closeMenu() { a.menuSlot, a.menuKey = -1, -1 }

func (a *App) selectTower(t *game.Tower) {
	if a.sel != t {
		a.sellArm = 0
	}
	a.sel = t
}

// buildAt starts a turret of type def on an empty pad and selects it.
func (a *App) buildAt(slot int, def *game.TowerDef) {
	t, err := a.world.Build(slot, def)
	switch {
	case err == nil:
		a.closeMenu()
		a.selectTower(t)
	case errors.Is(err, game.ErrNoCredits):
		a.notEnough(a.lvl.Slots[slot].Center())
	}
}

// quickBuild builds menu entry i on the pad with the open menu (or under the cursor).
func (a *App) quickBuild(i int) {
	if !a.world.Active() || i >= len(game.TowerOrder) || !a.world.CanBuild(game.TowerOrder[i]) {
		return
	}
	slot := a.menuSlot
	if slot < 0 && a.hoverSlot >= 0 && a.world.SlotTower[a.hoverSlot] == nil {
		slot = a.hoverSlot
	}
	if slot < 0 {
		return
	}
	a.buildAt(slot, game.TowerOrder[i])
}

// sellOrCancel cancels a construction job, or sells the selected turret after a confirm.
func (a *App) sellOrCancel() {
	t := a.sel
	if t == nil || !a.world.Active() {
		return
	}
	if t.Busy {
		a.world.Cancel(t)
		return
	}
	if a.sellArm != t.ID || a.now-a.sellT > sellConfirmTime {
		a.sellArm, a.sellT = t.ID, a.now
		return
	}
	a.world.Sell(t)
}

const sellConfirmTime = 3.0

// towerGone forgets a turret that was sold or cancelled.
func (a *App) towerGone(id int) {
	if a.sel != nil && a.sel.ID == id {
		a.sel = nil
	}
	if a.sellArm == id {
		a.sellArm = 0
	}
	delete(a.spin, id)
}

// towerShot plays the muzzle effects of a turret firing.
func (a *App) towerShot(ev game.Event) {
	if ev.Tower == nil {
		return
	}
	var tw *game.Tower
	for _, t := range a.world.Towers {
		if t.ID == ev.ID {
			tw = t
			break
		}
	}
	if tw == nil {
		return
	}
	lk := a.as.Towers[tw.Def]
	tx, ty, tz := lk.tip(tw, tw.Level)
	tz += art.PadH
	switch tw.Def.Attack {
	case game.AtkFlame:
		st := tw.Stats()
		a.fx.Flame(tx, ty, tz, a.ground(tx, ty), tw.Angle, tw.Def.Cone, (st.Range-0.3)*art.CellPx)
	case game.AtkMissile:
		a.fx.Launch(tx, ty, tz, tw.Angle)
	}
}

// railBeam shows a railgun shot from the muzzle to its end point.
func (a *App) railBeam(ev game.Event) {
	P := float64(art.CellPx)
	lk := a.as.Towers[ev.Tower]
	lv := min(ev.Level, towerLevels-1)
	dir := game.FromAngle(ev.Angle)
	tp := ev.Pos.Sub(dir.Mul(0.5))
	x0, y0 := tp.X*P+dir.X*lk.tipFwd[lv], tp.Y*P+dir.Y*lk.tipFwd[lv]
	z0 := art.PadH + lk.tipZ[lv]
	x1, y1 := ev.To.X*P, ev.To.Y*P
	z1 := a.ground(x1, y1) + 6
	if !ev.Flag {
		if el := a.as.Looks[ev.Enemy]; el != nil {
			z1 = a.ground(x1, y1) + el.bodyZ + el.hover
		}
	}
	a.fx.Rail(x0, y0, z0, x1, y1, z1, ev.Flag)
}

// updateTowerFX runs the continuous effects of turrets, burning enemies and napalm.
func (a *App) updateTowerFX(dt, gdt float64) {
	w := a.world
	P := float64(art.CellPx)
	if a.menuSlot >= 0 {
		a.menuT += dt
		if w.SlotTower[a.menuSlot] != nil || !w.Active() {
			a.closeMenu()
		}
	}
	if a.sellArm != 0 && a.now-a.sellT > sellConfirmTime {
		a.sellArm = 0
	}
	if gdt <= 0 {
		return
	}
	chance := func(rate float64) bool { return a.fx.rng.Float64() < rate*gdt }
	for _, tw := range w.Towers {
		if !tw.Def.Aims() {
			sp := 0.9
			if tw.Busy {
				sp = 0.25
			} else if tw.ShotT < 0.8 {
				sp += 5 * (1 - tw.ShotT/0.8)
			}
			a.spin[tw.ID] = math.Mod(a.spin[tw.ID]+sp*gdt, 2*math.Pi)
		}
		if tw.Busy && chance(18) {
			lk := a.as.Towers[tw.Def]
			cut := easeInOut(tw.Progress()) * (lk.topZ[tw.JobLevel()] + 1)
			a.fx.BuildSparks(tw.Pos.X*P, tw.Pos.Y*P, art.PadH+cut, jobColour(tw, lk))
		}
	}
	for _, e := range w.Enemies {
		if e.Alive() && e.BurnT > 0 {
			lk := a.as.Looks[e.Def]
			if chance(14 * math.Max(1, lk.size)) {
				a.fx.Burning(e.Pos.X*P, e.Pos.Y*P, a.enemyZ(e, lk)+lk.bodyZ, 3*lk.size)
			}
		}
	}
	for _, p := range w.Patches {
		x, y := p.Pos.X*P, p.Pos.Y*P
		if chance(22) {
			a.fx.NapalmFire(x, y, a.ground(x, y), p.R*P*0.8)
		}
	}
	for _, p := range w.Projectiles {
		if p.Kind == game.ProjMissile && p.Life > 0.03 {
			x, y := p.Pos.X*P, p.Pos.Y*P
			a.fx.MissileTrail(x, y, a.flightZ(p, p.Pos))
		}
	}
}

func (a *App) notEnough(p game.Vec) {
	a.crFlash = 1
	a.fx.Float(p.X*art.CellPx, p.Y*art.CellPx, towerTopZ+4, "Not enough credits", art.Red, 1.3)
}

// slotAt returns the pad under the screen point: turret bodies that stick up
// above their pads are tested first (front to back), then the pad tops.
func (a *App) slotAt(mx, my int) int {
	P := float64(art.CellPx)
	cx, cy := a.cam.ToWorld(float64(mx), float64(my))
	best, bestY := -1, math.Inf(-1)
	for i, t := range a.world.SlotTower {
		if t == nil {
			continue
		}
		tx, ty := a.at(t.Pos.X*P, t.Pos.Y*P, art.PadH)
		if math.Abs(cx-tx) <= 13 && cy >= ty-(towerTopZ-art.PadH)*art.HZ && cy <= ty+11 && ty > bestY {
			best, bestY = i, ty
		}
	}
	if best >= 0 {
		return best
	}
	wx := cx
	wy := (cy - float64(a.as.TI.OY) + art.PadH*art.HZ) / art.GY
	return a.lvl.SlotAt(int(math.Floor(wx/P)), int(math.Floor(wy/P)))
}

func (a *App) skipTimer() {
	if a.world.Phase == game.PhasePrep {
		a.world.SkipTimer()
	}
}

func (a *App) upgradeSelected() {
	if a.sel == nil || !a.world.Active() || a.sel.MaxLevel() {
		return
	}
	if err := a.world.Upgrade(a.sel); errors.Is(err, game.ErrNoCredits) {
		a.notEnough(a.sel.Pos)
	}
}

func (a *App) toggleSpeed() {
	if a.speed == 1 {
		a.speed = 2
	} else {
		a.speed = 1
	}
}
