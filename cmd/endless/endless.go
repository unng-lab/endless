package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/unng-lab/endless/pkg/endless"
)

func main() {
	ebiten.SetWindowTitle("Endless")
	ebiten.SetWindowSize(endless.DefaultScreenWidth, endless.DefaultScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(false)
	ebiten.SetVsyncEnabled(false)

	if err := ebiten.RunGame(endless.NewGame()); err != nil {
		log.Fatalf("run endless: %v", err)
	}
}
