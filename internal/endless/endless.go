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

const (
	unitCount = 1000
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

type Game struct {
	log       *slog.Logger
	camera    *camera.Camera
	wg        sync.WaitGroup
	ui        *ui.UIEngine
	inventory *Inventory
	board     *board.Board
	Units     []*unit.Unit
	OnBoard   []*unit.Unit
}

func NewGame(
// analyticsDB *ch.AnaliticsDB,
) *Game {
	var g Game
	g.Units = make([]*unit.Unit, 0, unitCount)
	g.OnBoard = make([]*unit.Unit, 0, unitCount)
	g.camera = camera.New(board.TileSize, board.CountTile)
	g.ui = ui.New(g.camera)
	newBoard, err := board.NewBoard(g.camera)
	if err != nil {
		panic(err)
	}
	g.board = newBoard

	g.inventory = NewInverntory(g.camera)
	for i := range unitCount {
		newUnit := g.inventory.Units["runner"].New(
			i,
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
			g.board,
			//analyticsDB,
		)
		wg := make(chan *sync.WaitGroup, 1)
		g.Units = append(g.Units, newUnit)
		g.Units[i].Run(wg)
	}

	return &g
}
