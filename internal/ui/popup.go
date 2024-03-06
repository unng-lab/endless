package ui

import (
	"image/color"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/internal/window"
)

var _ block = (*Popup)(nil)

type Popup struct {
	el
	log       *slog.Logger
	clr       color.Color
	antialias bool
	open      bool
	Toggle    func() error
}

func NewPopup(
	log *slog.Logger,
	x, y, width, height float32,
	window *window.Default,
	clr color.Color,
	antialias bool,
) *Popup {
	p := Popup{
		log:       log,
		clr:       clr,
		antialias: antialias,
		el: el{
			perX:      x,
			perY:      y,
			perWidth:  width,
			perHeight: height,
			window:    window,
		},
	}
	p.Toggle = func() error {
		p.open = !p.open
		return nil
	}
	return &p
}

func (p *Popup) Clicked(x, y int) bool {
	if p.x() <= x && x <= p.x()+p.w() && p.y() <= y && y <= p.y()+p.h() {
		return true
	}
	return false
}

func (p *Popup) Draw(screen *ebiten.Image) {
	//p.log.Debug("popup positions", float32(p.x()), float32(p.y()), float32(p.w()), float32(p.h()))
	if p.open {
		vector.DrawFilledRect(screen, float32(p.x()), float32(p.y()), float32(p.w()), float32(p.h()), p.clr, p.antialias)
	}
}
