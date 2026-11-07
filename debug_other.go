//go:build !js

package main

type debugConfig struct {
	speed       float64
	noAutoPause bool
	fxTest      bool
	wave        int // start at this wave (testing)
	credits     int // start with these credits (testing)
	level       string
	screen      string
	unlock      bool
	win         bool
}

func loadDebug() debugConfig { return debugConfig{speed: 1} }

func (d debugConfig) applySave(s *Save) {}

func (d debugConfig) applyColony(s *Save)   {}
func (d debugConfig) applyCrystals(s *Save) {}

func publishState(a *App) {}
