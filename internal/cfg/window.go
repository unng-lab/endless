package cfg

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/window"
)

var _ window.Config = (*Default)(nil)

func (d Default) GetScreenWidth() uint64 {
	return d.screenWidth
}

func (d Default) GetScreenHeight() uint64 {
	return d.screenHeight
}

func (d Default) GetWindowResizeMode() ebiten.WindowResizingModeType {
	return ebiten.WindowResizingModeType(d.windowResizeMode)
}
