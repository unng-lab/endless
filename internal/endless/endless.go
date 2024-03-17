package endless

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

var G Game

type Game struct {
	log    *slog.Logger
	camera camera.Camera
	Units  []Unit
}

func NewGame() *Game {
	G.Units = make([]Unit, 0)
	camera.TileSize = board.TileSize
	camera.CountTile = board.CountTile
	err := board.NewBoard()
	if err != nil {
		panic(err)
	}
	NewInverntory()
	G.Units = append(G.Units, I.Units["runner"].New(30, 10))
	G.Units = append(G.Units, I.Units["runner"].New(500, 10))
	G.Units = append(G.Units, I.Units["runner"].New(10, 10))
	G.Units = append(G.Units, I.Units["runner"].New(-500, 10))
	return &G
}
