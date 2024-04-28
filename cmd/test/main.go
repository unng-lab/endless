package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

func update(screen *ebiten.Image) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		ebitenutil.DrawLine(screen, float64(x)-10, float64(y)-10, float64(x)+10, float64(y)-10, color.White)
		ebitenutil.DrawLine(screen, float64(x)+10, float64(y)-10, float64(x)+10, float64(y)+10, color.White)
		ebitenutil.DrawLine(screen, float64(x)+10, float64(y)+10, float64(x)-10, float64(y)+10, color.White)
		ebitenutil.DrawLine(screen, float64(x)-10, float64(y)+10, float64(x)-10, float64(y)-10, color.White)
	}
	return nil
}

func main() {
	ebiten.Run(update, screenWidth, screenHeight, 2, "Draw Unfilled Square Example")
}
