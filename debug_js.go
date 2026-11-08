//go:build js

package main

import (
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

type debugConfig struct {
	speed       float64
	noAutoPause bool
	fxTest      bool
	wave        int    // start at this wave (testing)
	credits     int    // start with these credits (testing)
	level       string // start this level right away
	screen      string // "levels": open the level select
	unlock      bool   // every level is open
	win         bool   // win the level right away (result screen)
	stars       []int  // pretend results of the campaign levels
	crystals    int    // pretend crystals (-1: keep)
	colony      string // pretend colony stages: "habitat:2,pad:1"
	kills       string // pretend kills: "crawler:400,walker:30"
	wins        int    // pretend battles won (0: keep)
	supers      string // pretend supers built: "blaster,cryo"
	flags       string // pretend records: "flawless,close"
	expSeed     int64  // pretend expedition on offer (?exp=1234)
}

// loadDebug reads test options from the page URL, e.g. ?speed=4.
func loadDebug() debugConfig {
	d := debugConfig{speed: 1, noAutoPause: true}
	href := js.Global().Get("location").Get("href").String()
	if u, err := url.Parse(href); err == nil {
		if s, err := strconv.ParseFloat(u.Query().Get("speed"), 64); err == nil && s > 0 {
			d.speed = s
		}
		d.fxTest = u.Query().Get("fx") == "1"
		d.wave, _ = strconv.Atoi(u.Query().Get("wave"))
		d.credits, _ = strconv.Atoi(u.Query().Get("credits"))
		d.level = u.Query().Get("level")
		d.screen = u.Query().Get("screen")
		d.unlock = u.Query().Get("unlock") == "1"
		d.win = u.Query().Get("win") == "1"
		d.crystals = -1
		if n, err := strconv.Atoi(u.Query().Get("crystals")); err == nil {
			d.crystals = n
		}
		d.colony = u.Query().Get("colony")
		d.kills = u.Query().Get("kills")
		d.wins, _ = strconv.Atoi(u.Query().Get("wins"))
		d.supers = u.Query().Get("supers")
		d.flags = u.Query().Get("flags")
		d.expSeed, _ = strconv.ParseInt(u.Query().Get("exp"), 10, 64)
		if st := u.Query().Get("stars"); st != "" {
			for _, f := range strings.Split(st, ",") {
				n, _ := strconv.Atoi(f)
				d.stars = append(d.stars, n)
			}
		}
	}
	return d
}

// applySave fills in pretend campaign results (?stars=3,2,1).
func (d debugConfig) applySave(s *Save) {
	if d.expSeed > 0 {
		s.Exp.Seed = d.expSeed
	}
	var ids []string
	for _, c := range game.Chapters {
		for _, l := range c.Levels {
			ids = append(ids, l.ID)
		}
	}
	ids = append(ids, game.Skirmish.ID)
	for i, n := range d.stars {
		if i < len(ids) && n > 0 {
			s.Stars[ids[i]] = n
		}
	}
	if len(d.stars) > 0 {
		for _, e := range game.EnemyOrder[:min(len(game.EnemyOrder), len(d.stars)+1)] {
			s.Seen = append(s.Seen, e.ID)
		}
	}
}

// applyCrystals sets pretend crystals (?crystals=200).
func (d debugConfig) applyCrystals(s *Save) {
	if d.crystals >= 0 {
		s.Crystals = d.crystals
	}
}

// applyColony fills in pretend colony stages and trophy records
// (?colony=habitat:2&kills=crawler:400&wins=20&supers=blaster&flags=close).
func (d debugConfig) applyColony(s *Save) {
	for _, f := range strings.Split(d.kills, ",") {
		if id, n, ok := strings.Cut(f, ":"); ok {
			s.Kills[id], _ = strconv.Atoi(n)
		}
	}
	if d.wins > 0 {
		s.Wins = d.wins
	}
	for _, id := range strings.Split(d.supers, ",") {
		if id != "" && !slices.Contains(s.Supers, id) {
			s.Supers = append(s.Supers, id)
		}
	}
	s.Flawless = s.Flawless || strings.Contains(d.flags, "flawless")
	s.Close = s.Close || strings.Contains(d.flags, "close")
	for _, f := range strings.Split(d.colony, ",") {
		if id, n, ok := strings.Cut(f, ":"); ok {
			v, _ := strconv.Atoi(n)
			if !strings.Contains(id, ".") {
				id = "mars." + id
			}
			s.Colony[id] = v
		}
	}
}

type slotInfo struct {
	X, Y  float64
	Tower int // level, -1 for an empty pad
	Type  string
	Busy  bool
}

// publishState exposes a JSON snapshot for browser-driven tests.
func publishState(a *App) {
	w := a.world
	dpr := a.dpi
	st := map[string]any{
		"phase": int(w.Phase), "wave": w.Wave, "credits": w.Credits, "hp": w.BaseHP,
		"timer": w.Timer, "enemies": len(w.Enemies), "towers": len(w.Towers), "kills": w.Kills,
		"zoom": a.cam.Z, "fit": a.cam.Fit, "camX": a.cam.X, "camY": a.cam.Y, "paused": a.paused,
		"sw": a.sw, "sh": a.sh, "u": a.u, "selected": a.sel != nil, "hoverBtn": a.hoverBtn,
		"screen": int(a.scr), "panel": int(a.popup), "demo": a.demo, "level": a.spec.ID, "hold": w.Hold,
		"pick": a.pick, "resStars": a.res.stars, "endT": a.endT, "tab": a.tab, "saved": a.save.Stars,
		"fps": ebiten.ActualFPS(), "tps": ebiten.ActualTPS(),
		"exp":  map[string]any{"seed": a.save.Exp.Seed, "count": a.save.Exp.Count, "won": a.save.Exp.Won},
		"tier": int(a.spec.Tier), "levelName": a.spec.Name,
	}
	var slots []slotInfo
	for i, s := range a.lvl.Slots {
		c := s.Center()
		x, y := a.screenAt(c.X*art.CellPx, c.Y*art.CellPx, art.PadH)
		si := slotInfo{X: x / dpr, Y: y / dpr, Tower: -1}
		if t := w.SlotTower[i]; t != nil {
			si.Tower, si.Type, si.Busy = t.Level, t.Def.ID, t.Busy
		}
		slots = append(slots, si)
	}
	st["slots"] = slots
	btns := map[string][4]float64{}
	for _, b := range a.buttons {
		btns[b.ID] = [4]float64{float64(b.R.X) / dpr, float64(b.R.Y) / dpr, float64(b.R.W) / dpr, float64(b.R.H) / dpr}
	}
	st["buttons"] = btns
	if a.sel != nil {
		x, y := a.screenAt(a.sel.Pos.X*art.CellPx, a.sel.Pos.Y*art.CellPx, art.PadH)
		st["selPos"] = [2]float64{x / dpr, y / dpr}
	}
	st["menu"] = a.menuSlot
	st["sellArm"] = a.sellArm != 0
	st["cards"] = len(a.cards)
	if len(a.cards) > 0 {
		st["lesson"] = a.cards[0].lesson != nil
	}
	st["boss"] = w.Boss() != nil
	if b := w.Boss(); b != nil {
		x, y := a.screenAt(b.Pos.X*art.CellPx, b.Pos.Y*art.CellPx, 0)
		st["bossPos"] = [2]float64{x / dpr, y / dpr}
		st["bossDist"] = b.Dist
		st["bossUnder"] = b.Under
		st["bossHP"] = b.HP
	}
	st["seen"] = len(a.seen)
	var strip [][4]float64
	for _, it := range a.stripItems() {
		strip = append(strip, [4]float64{float64(it.R.X) / dpr, float64(it.R.Y) / dpr, float64(it.R.W) / dpr, float64(it.R.H) / dpr})
	}
	st["strip"] = strip
	var en [][3]float64
	for _, e := range w.Enemies {
		if e.Alive() {
			x, y := a.screenAt(e.Pos.X*art.CellPx, e.Pos.Y*art.CellPx, 0)
			en = append(en, [3]float64{x / dpr, y / dpr, e.HP})
		}
	}
	st["enemyPos"] = en
	for _, e := range w.Enemies {
		if e.Alive() && e.Def.Flying {
			x, y := a.screenAt(e.Pos.X*art.CellPx, e.Pos.Y*art.CellPx, 0)
			st["flyPos"] = [2]float64{x / dpr, y / dpr}
			break
		}
	}
	st["crystals"] = a.save.Crystals
	st["colony"] = a.save.Colony
	if a.scr == scrColony {
		st["colTab"] = a.col.tab
		st["colLocation"] = a.colonyLoc().ID
		st["colLocked"] = !a.onBridge() && a.colonyLocked(a.colonyLoc())
		lots := map[string][2]float64{}
		for _, b := range a.colonyLoc().Buildings {
			x0, y0, x1, y1 := a.lotRect(b)
			sx, sy := a.cam.ToScreen((x0+x1)/2, (y0+y1)/2)
			lots[b.ID] = [2]float64{sx / dpr, sy / dpr}
		}
		st["lots"] = lots
		if a.col.sel != nil {
			st["colSel"] = a.col.sel.ID
		}
		st["walkers"] = len(a.col.walkers)
		st["ship"] = int(a.col.ship.phase)
		st["anims"] = len(a.col.anims)
		st["bridge"] = a.onBridge()
		if a.onBridge() {
			at := func(x0, y0, x1, y1 float64) [2]float64 {
				sx, sy := a.cam.ToScreen((x0+x1)/2, (y0+y1)/2)
				return [2]float64{sx / dpr, sy / dpr}
			}
			var peds, jars [][2]float64
			for i := range game.Trophies {
				peds = append(peds, at(a.pedestalBox(i)))
			}
			for i := range game.Specimens {
				jars = append(jars, at(a.jarBox(i)))
			}
			st["pedestals"], st["jars"] = peds, jars
			st["table"] = at(a.tableBox())
			var won []string
			for _, t := range game.AllTrophies {
				if a.won(t) {
					won = append(won, t.ID)
				}
			}
			st["trophies"] = won
			if a.br.sel != nil {
				st["brSel"] = a.br.sel.ID
			}
			if a.br.hover != nil {
				st["brHover"] = a.br.hover.ID
			}
			st["brTable"] = a.br.table
			st["brShow"] = a.br.show
		}
	}
	b, _ := json.Marshal(st)
	js.Global().Set("tdState", string(b))
}
