package ui_old

import (
	"image/color"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/internal/window"
)

var _ block = (*Panel)(nil)

type Panel struct {
	log       *slog.Logger
	clr       color.Color
	antialias bool
	el
}

func (p *Panel) Draw(screen *ebiten.Image) {
	//p.log.Debug("panel positions", float32(p.x()), float32(p.y()), float32(p.w()), float32(p.h()))
	vector.DrawFilledRect(screen, float32(p.x()), float32(p.y()), float32(p.w()), float32(p.h()), p.clr, p.antialias)
}

func (p *Panel) Clicked(x, y int) bool {
	if p.x() <= x && x <= p.x()+p.w() && p.y() <= y && y <= p.y()+p.h() {
		return true
	}
	return false
}

func NewPanel(
	log *slog.Logger,
	x, y, width, height float32,
	window *window.Default,
	clr color.Color,
	antialias bool,
) *Panel {
	return &Panel{
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
}

func (p *Panel) AddButton(x, y, width, height float32, clr color.Color, antialias bool) {
	return
}
