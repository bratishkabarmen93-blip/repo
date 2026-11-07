package main

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"outpost/internal/game"
)

const saveVersion = 2

// Save is the player's progress and settings, kept between runs.
type Save struct {
	Version     int            `json:"version"`
	Stars       map[string]int `json:"stars"` // best result per level ID
	Seen        []string       `json:"seen"`  // enemy types met so far
	Fullscreen  bool           `json:"fullscreen"`
	NoAutoPause bool           `json:"noAutoPause"`
	Last        string         `json:"last"`           // level played last
	Open        []string       `json:"open,omitempty"` // bonus levels unlocked for good (Skirmish)
	Crystals    int            `json:"crystals"`       // to spend on the colonies
	Earned      int            `json:"earned"`         // crystals earned in all
	Paid        map[string]int `json:"paid"`           // stars already paid out per level (nil: not counted yet)
	Colony      map[string]int `json:"colony"`         // stages built per colony building
	// Trophies (v0.13): the record they are judged on and the ones won.
	Kills    map[string]int  `json:"kills"`              // enemies destroyed per type (nil: not counted yet)
	Wins     int             `json:"wins"`               // battles won, replays included
	Supers   []string        `json:"supers,omitempty"`   // turret types whose super has been built
	Flawless bool            `json:"flawless,omitempty"` // won a battle with the base untouched
	Close    bool            `json:"close,omitempty"`    // won a battle with 10% of the base or less
	Trophies map[string]bool `json:"trophies"`           // trophies won (and paid)
}

func loadSave() *Save {
	s := &Save{}
	if b, err := readSaveData(); err == nil {
		if err := json.Unmarshal(b, s); err != nil {
			log.Printf("save: %v", err)
			s = &Save{}
		}
	}
	s.Version = saveVersion
	if s.Stars == nil {
		s.Stars = map[string]int{}
	}
	if s.Colony == nil {
		s.Colony = map[string]int{}
	}
	return s
}

// payPast credits the stars won before crystals existed (saves of v0.11 and
// older) at the first-time rate; it returns the crystals given.
func (s *Save) payPast() int {
	if s.Paid != nil {
		return 0
	}
	s.Paid = map[string]int{}
	n := 0
	for id, st := range s.Stars {
		if l := game.LevelByID(id); l != nil && st > 0 {
			first, _ := game.StarPay(l.Tier)
			n += st * first
			s.Paid[id] = st
		}
	}
	s.Crystals += n
	s.Earned += n
	return n
}

// countPast credits the battles won before trophies existed (saves of v0.12
// and older): one win and the enemies of its waves for every level beaten.
func (s *Save) countPast() {
	if s.Kills != nil {
		return
	}
	s.Kills = map[string]int{}
	for id, st := range s.Stars {
		if l := game.LevelByID(id); l != nil && st > 0 {
			s.Wins++
			for e, n := range game.LevelKills(l) {
				s.Kills[e] += n
			}
		}
	}
}

// record is the progress the trophies are judged on.
func (s *Save) record() *game.Record {
	return &game.Record{Stars: s.Stars, Kills: s.Kills, Supers: s.Supers, Wins: s.Wins,
		Flawless: s.Flawless, Close: s.Close, Colony: s.Colony}
}

// claimTrophies marks the trophies the record has newly won, pays their
// crystals and returns them.
func (s *Save) claimTrophies() []*game.Trophy {
	if s.Trophies == nil {
		s.Trophies = map[string]bool{}
	}
	r := s.record()
	var won []*game.Trophy
	for _, t := range game.AllTrophies {
		if !s.Trophies[t.ID] && t.Done(r) {
			s.Trophies[t.ID] = true
			s.Crystals += t.Reward
			s.Earned += t.Reward
			won = append(won, t)
		}
	}
	if len(won) > 0 {
		s.store()
	}
	return won
}

// trophyCount is the number of trophies won among ts.
func (s *Save) trophyCount(ts []*game.Trophy) int {
	n := 0
	for _, t := range ts {
		if s.Trophies[t.ID] {
			n++
		}
	}
	return n
}

func (s *Save) store() {
	b, err := json.MarshalIndent(s, "", "  ")
	if err == nil {
		err = writeSaveData(b)
	}
	if err != nil {
		log.Printf("save: %v", err)
	}
}

// reset forgets all progress but keeps the settings.
func (s *Save) reset() {
	s.Stars = map[string]int{}
	s.Seen = nil
	s.Last = ""
	s.Crystals, s.Earned = 0, 0
	s.Paid = map[string]int{}
	s.Colony = map[string]int{}
	s.Kills = map[string]int{}
	s.Wins, s.Supers, s.Flawless, s.Close = 0, nil, false, false
	s.Trophies = map[string]bool{}
	s.store()
}

func (s *Save) hasSeen(d *game.EnemyDef) bool { return slices.Contains(s.Seen, d.ID) }

// best is the most stars earned on a level.
func (a *App) best(s *game.LevelSpec) int { return a.save.Stars[s.ID] }

// chapterOf finds the chapter of a level and its index in it.
func chapterOf(s *game.LevelSpec) (*game.Chapter, int) {
	for _, c := range game.Chapters {
		for i, l := range c.Levels {
			if l == s {
				return c, i
			}
		}
	}
	return nil, -1
}

// unlocked reports whether a level can be played: a campaign level opens
// when the one before it has been won.
func (a *App) unlocked(s *game.LevelSpec) bool {
	if a.dbg.unlock {
		return true
	}
	c, i := chapterOf(s)
	switch {
	case c == nil:
		return slices.Contains(a.save.Open, s.ID) || a.chapterDone(game.Chapters[0])
	case i > 0:
		return a.best(c.Levels[i-1]) > 0
	case c.Num > 1:
		prev := game.Chapters[c.Num-2]
		return a.best(prev.Levels[len(prev.Levels)-1]) > 0
	}
	return true
}

// chapterDone reports whether every level of a chapter has been won.
func (a *App) chapterDone(c *game.Chapter) bool {
	for _, l := range c.Levels {
		if a.best(l) == 0 {
			return false
		}
	}
	return true
}

// keepBonusOpen remembers the Skirmish map once it is open, so that new
// campaign levels in later versions do not lock it again.
func (a *App) keepBonusOpen() bool {
	sk := game.Skirmish
	if slices.Contains(a.save.Open, sk.ID) || !a.unlocked(sk) || a.dbg.unlock {
		return false
	}
	a.save.Open = append(a.save.Open, sk.ID)
	a.save.store()
	return true
}

// lockText tells how to open a locked level.
func lockText(s *game.LevelSpec) string {
	c, i := chapterOf(s)
	switch {
	case c == nil:
		return "Complete chapter 1 to unlock it."
	case i > 0:
		return "Win level " + c.Levels[i-1].ID + " to unlock it."
	}
	return fmt.Sprintf("Complete chapter %d to unlock it.", c.Num-1)
}

// nextLevel is the campaign level after s, or nil.
func nextLevel(s *game.LevelSpec) *game.LevelSpec {
	c, i := chapterOf(s)
	if c == nil {
		return nil
	}
	if i+1 < len(c.Levels) {
		return c.Levels[i+1]
	}
	if c.Num < len(game.Chapters) {
		return game.Chapters[c.Num].Levels[0]
	}
	return nil
}

// levelLabel is "1-2" for campaign levels and "Skirmish" for the rest.
func levelLabel(s *game.LevelSpec) string {
	if c, _ := chapterOf(s); c != nil {
		return s.ID
	}
	return "Skirmish"
}

// chapterStars sums the best results of a chapter.
func (a *App) chapterStars(c *game.Chapter) int {
	n := 0
	for _, l := range c.Levels {
		n += a.best(l)
	}
	return n
}
