package endless

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

var G Game

type Game struct {
	log   *slog.Logger
	Units []Unit
}

func NewGame() *Game {
	err := NewBoard()
	if err != nil {
		panic(err)
	}

	return &G
}
