package ui

import (
	"image/color"
	"log/slog"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/ebitenfx"
	"github/unng-lab/madfarmer/internal/window"
)

var _ ebitenfx.UI = (*Default)(nil)

type Config interface {
	Test()
}

type block interface {
	Draw(screen *ebiten.Image)
	Clicked(x, y int) bool
}

type Default struct {
	log       *slog.Logger
	cfg       Config
	blocks    []block
	showPopup atomic.Bool
	window    *window.Default
}

func (d *Default) Clicked(x, y int) bool {
	for i := range d.blocks {
		if d.blocks[i].Clicked(x, y) {
			return true
		}
	}
	return false
}

func (d *Default) Update() error {
	return nil
}

func (d *Default) Draw(screen *ebiten.Image) {
	for i := range d.blocks {
		d.blocks[i].Draw(screen)
	}
}

func New(log *slog.Logger, cfg Config, window *window.Default) *Default {
	var d Default
	d.log = log.With("ui", "Default")
	d.cfg = cfg
	d.window = window

	topPanel := NewPanel(
		d.log.With("name", "top panel"),
		10,
		0,
		80,
		10,
		d.window,
		color.White,
		true,
	)
	bottomPanel := NewPanel(
		d.log.With("name", "bottom panel"),
		10,
		80,
		80,
		15,
		d.window,
		color.White,
		true,
	)

	popup := NewPopup(
		d.log.With("name", "bottom panel"),
		10,
		20,
		80,
		50,
		d.window,
		color.White,
		true,
	)

	d.blocks = append(d.blocks, topPanel, bottomPanel, popup)
	return &d
}
