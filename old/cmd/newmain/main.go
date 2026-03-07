package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/game"
)

func main() {
	ebiten.SetWindowTitle("Endless Prototype")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g := game.New(game.Config{
		ScreenWidth:  1280,
		ScreenHeight: 720,
		TileColumns:  4096,
		TileRows:     4096,
		TileSize:     64,
		MinZoom:      0.3,
		MaxZoom:      6,
		ZoomStep:     0.1,
		PanSpeed:     900,
	})

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("run game: %v", err)
	}
}
