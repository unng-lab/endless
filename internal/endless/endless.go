package endless

import (
	"log/slog"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/ui"
	"github/unng-lab/madfarmer/internal/unit"
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

var G Game

type u struct {
	unit *unit.Unit
	wg   unit.WG
	c    chan *unit.WG
}

type Game struct {
	log    *slog.Logger
	camera *camera.Camera
	wg     sync.WaitGroup
	ui     *ui.UIEngine
	Units  []u
}

func NewGame() *Game {
	G.Units = make([]u, 0, 10000)
	G.camera = camera.New(board.TileSize, board.CountTile)
	G.ui = ui.New(G.camera)
	err := board.NewBoard()
	if err != nil {
		panic(err)
	}
	NewInverntory()
	for i := range 100 {
		newUnit := I.Units["runner"].New(
			i,
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
		)
		newU := u{
			unit: &newUnit,
			wg: unit.WG{
				WG: &G.wg,
			},
			c: make(chan *unit.WG, 1),
		}
		G.Units = append(G.Units, newU)
		G.Units[i].unit.Run(newU.c)
	}

	return &G
}
