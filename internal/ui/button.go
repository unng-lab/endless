package ui

import (
	"image/color"
	"log/slog"

	"github/unng-lab/madfarmer/internal/window"
)

type Button struct {
	el
	log       *slog.Logger
	clr       color.Color
	antialias bool
	mouseDown bool
	onPressed func() error
}

func NewButton(
	log *slog.Logger,
	x, y, width, height float32,
	window *window.Default,
	clr color.Color,
	antialias bool,
	onPressed func() error,
) *Button {
	b := Button{
		log:       log,
		clr:       clr,
		antialias: antialias,
		onPressed: onPressed,
		el: el{
			perX:      x,
			perY:      y,
			perWidth:  width,
			perHeight: height,
			window:    window,
		},
	}

	return &b
}

func (b *Button) Clicked(x, y int) bool {
	if b.x() <= x && x <= b.x()+b.w() && b.y() <= y && y <= b.y()+b.h() {
		return true
	}
	return false
}
