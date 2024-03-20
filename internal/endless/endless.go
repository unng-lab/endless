package endless

import (
	"log/slog"
	"math/rand"

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
	camera.DefaultTileSize = board.TileSize
	camera.CountTile = board.CountTile
	err := board.NewBoard()
	if err != nil {
		panic(err)
	}
	NewInverntory()
	for i := range 1000 {
		G.Units = append(G.Units, I.Units["runner"].New(
			i,
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
		))
	}

	return &G
}
