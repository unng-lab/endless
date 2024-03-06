package ebitenfx

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/window"
)

var _ ebiten.Game = (*Game)(nil)

type Config interface {
}
type Screen interface {
	Update() error
	Draw(screen *ebiten.Image)
}

type UI interface {
	Update() error
	Draw(screen *ebiten.Image)
	Clicked(x, y int) bool
}

type Game struct {
	log    *slog.Logger
	cfg    Config
	scr    Screen
	ui     UI
	window *window.Default
}

func New(
	log *slog.Logger,
	cfg Config,
	ui UI,
	scr Screen,
	window *window.Default,
) *Game {
	var g Game
	g.log = log.With("ebitenfx", "Game")
	g.cfg = cfg

	g.ui = ui
	g.scr = scr
	g.window = window

	return &g
}
