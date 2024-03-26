package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func (ui *UIEngine) Draw(screen *ebiten.Image) {
	ui.UI.Draw(screen)
}
