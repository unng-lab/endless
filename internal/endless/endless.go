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

type u struct {
	unit *unit.Unit
	wg   unit.WG
	c    chan *unit.WG
}

type Game struct {
	log       *slog.Logger
	camera    *camera.Camera
	wg        sync.WaitGroup
	ui        *ui.UIEngine
	inventory *Inventory
	Units     []u
}

func NewGame() *Game {
	var g Game
	g.Units = make([]u, 0, 10000)
	g.camera = camera.New(board.TileSize, board.CountTile)
	g.ui = ui.New(g.camera)
	err := board.NewBoard()
	if err != nil {
		panic(err)
	}
	g.inventory = NewInverntory(g.camera)
	for i := range 100 {
		newUnit := g.inventory.Units["runner"].New(
			i,
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
		)
		newU := u{
			unit: &newUnit,
			wg: unit.WG{
				WG: &g.wg,
			},
			c: make(chan *unit.WG, 1),
		}
		g.Units = append(g.Units, newU)
		g.Units[i].unit.Run(newU.c)
	}

	return &g
}
