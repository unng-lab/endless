package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/endless"
)

func main() {
	ebiten.SetWindowSize(800, 800)
	//ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MadFarmer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(endless.NewGame())
	if err != nil {
		panic(err)
	}
}
