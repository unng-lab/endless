package endless

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

var G Game

type Game struct {
	log    *slog.Logger
	camera Camera
	Units  []Unit
}

func NewGame() *Game {
	G.Units = make([]Unit, 0)
	G.camera = Camera{
		positionX:   0,
		positionY:   0,
		zoomFactor:  0,
		tileSize:    TileSize,
		scaleFactor: 1,
	}

	err := NewBoard()
	if err != nil {
		panic(err)
	}
	NewInverntory()
	G.Units = append(G.Units, I.Units["runner"].New(10, 10))
	return &G
}
