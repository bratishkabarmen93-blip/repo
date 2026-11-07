// Outpost Defense: a sci-fi tower defense game.
package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fatal(crashed(r))
		}
	}()
	app, err := NewApp()
	if err != nil {
		fatal(err)
	}
	ebiten.SetWindowTitle("Outpost Defense")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(800, 450, -1, -1)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowIcon(WindowIcons())
	w, h := 1280, 720
	if m := ebiten.Monitor(); m != nil {
		mw, mh := m.Size()
		if mw > 0 && mh > 0 {
			w = mw * 8 / 10
			h = w * 9 / 16
			if h > mh*8/10 {
				h = mh * 8 / 10
				w = h * 16 / 9
			}
		}
	}
	ebiten.SetWindowSize(max(w, 800), max(h, 450))
	if app.save.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.RunGameWithOptions(app, &ebiten.RunGameOptions{}); err != nil && !errors.Is(err, ebiten.Termination) {
		fatal(err)
	}
}

func fatal(err error) {
	path := writeCrashLog(err.Error())
	showError("Outpost Defense has stopped because of an error.\n\n" + firstLine(err.Error()) + "\n\nDetails were saved to:\n" + path)
	log.Fatal(err)
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}
