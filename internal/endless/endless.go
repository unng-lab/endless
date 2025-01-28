package endless

import (
	"log/slog"
	"math/rand/v2"
	"sync"

	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/mapgrid"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/camera"
	"github.com/unng-lab/madfarmer/internal/ui"
	"github.com/unng-lab/madfarmer/internal/unit"
)

const (
	unitCount = 1000
	rockCount = 100000
	// TODO пересмотреть решение
	// пока нужно держать больше чем сумма всех юнитов
	moveChanBuffer = 1000000 // 1 миллион
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
	MapGrid   *mapgrid.MapGrid
}

func NewGame(
// analyticsDB *ch.AnaliticsDB,
) *Game {
	var g Game
	g.Units = make([]*unit.Unit, 0, unitCount)
	g.OnBoard = make([]*unit.Unit, 0, unitCount)
	g.camera = camera.New(board.TileSize, board.CountTile)
	slog.Info("camera created")
	g.ui = ui.New(g.camera)
	slog.Info("ui created")
	newBoard, err := board.NewBoard(g.camera)
	if err != nil {
		panic(err)
	}
	g.board = newBoard
	slog.Info("board created")

	g.inventory = NewInverntory(g.camera)
	slog.Info("inventory created")
	moveChan := make(chan unit.MoveMessage, moveChanBuffer)
	for i := range unitCount {
		newUnit := g.inventory.Units["runner"].New(
			i,
			getRandomPoint(g.board),
			g.board,
			moveChan,
			//analyticsDB,
		)
		wg := make(chan *sync.WaitGroup, 1)
		g.Units = append(g.Units, newUnit)
		newUnit.Run(wg)
	}
	slog.Info("units created")

	for i := range rockCount {
		newUnit := g.inventory.Units["rock"].New(
			i,
			getRandomPoint(g.board),
			g.board,
			moveChan,
		)
		wg := make(chan *sync.WaitGroup, 1)
		g.Units = append(g.Units, newUnit)
		newUnit.Run(wg)
	}
	slog.Info("rocks created")

	g.MapGrid = mapgrid.NewMapGrid(g.board, g.camera, moveChan)
	slog.Info("mapgrid created")
	slog.Info("game created")

	return &g
}

func getRandomPoint(board *board.Board) geom.Point {
	return geom.Point{
		X: float64(rand.Int64N(board.Width)),
		Y: float64(rand.Int64N(board.Height)),
	}
}
