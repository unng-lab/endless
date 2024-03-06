package ui

import (
	"github/unng-lab/madfarmer/internal/window"
)

type el struct {
	perX, perY, perWidth, perHeight float32
	window                          *window.Default
}

func (e *el) x() int {
	return e.window.PosX(e.perX)
}

func (e *el) y() int {
	return e.window.PosY(e.perY)
}

func (e *el) w() int {
	return e.window.PosX(e.perWidth)
}

func (e *el) h() int {
	return e.window.PosY(e.perHeight)
}
