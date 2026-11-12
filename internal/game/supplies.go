package game

import (
	"math"
	"slices"
)

// ---- Supplies: one-off help bought with crystals before an expedition ----

// Supply is a help packed for one expedition.
type Supply struct {
	ID, Name, Desc string
	Cost           int // crystals
}

// The supplies and what they add.
const (
	supplyCredits = 0.25 // of the starting credits
	supplyBase    = 0.5  // of the base
)

var (
	SupplyCredits = &Supply{ID: "credits", Name: "Extra credits", Desc: "+25% starting credits", Cost: 6}
	SupplyBase    = &Supply{ID: "base", Name: "Reinforced base", Desc: "+50% base strength", Cost: 6}

	Supplies = []*Supply{SupplyCredits, SupplyBase}
)

// SupplyByID finds a supply, nil when there is none of that name.
func SupplyByID(id string) *Supply {
	for _, sp := range Supplies {
		if sp.ID == id {
			return sp
		}
	}
	return nil
}

// Supplied is the level with supplies packed: s itself when none of them is
// known. The stars are still judged on the share of the base that is left.
func (s *LevelSpec) Supplied(ids []string) *LevelSpec {
	c := *s
	packed := false
	for _, sp := range Supplies {
		if !slices.Contains(ids, sp.ID) {
			continue
		}
		packed = true
		switch sp {
		case SupplyCredits:
			c.StartCredits = int(math.Round(float64(s.StartCredits)*(1+supplyCredits)/10)) * 10
		case SupplyBase:
			c.BaseHP = int(math.Round(float64(s.BaseHP) * (1 + supplyBase)))
		}
	}
	if !packed {
		return s
	}
	return &c
}
