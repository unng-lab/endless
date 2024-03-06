package ebitenfx

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 800
	screenHeight = 800
)

type Config interface {
	GetScreenWidth() int
	GetScreenHeight() int
	GetWindowResizeMode() ebiten.WindowResizingModeType
}

func RunGame(g ebiten.Game) error {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("camera (Ebitengine Demo)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}
