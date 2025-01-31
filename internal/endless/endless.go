package endless

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"

	"github.com/brianvoe/gofakeit/v7"

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
	moveChanBuffer = 1000 // 1 миллион

	tileSize  = 16
	tileCount = 1024
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
	moveChan  chan unit.MoveMessage
}

func NewGame(
// analyticsDB *ch.AnaliticsDB,
) *Game {
	var g Game
	g.Units = make([]*unit.Unit, 0, unitCount)
	g.OnBoard = make([]*unit.Unit, 0, unitCount)
	g.camera = camera.New(tileSize, tileCount)
	slog.Info("camera created")
	g.ui = ui.New(g.camera)
	slog.Info("ui created")
	newBoard, err := board.NewBoard(g.camera, tileSize, tileSize, tileCount)
	if err != nil {
		panic(err)
	}
	g.board = newBoard
	slog.Info("board created")

	g.inventory = NewInverntory(g.board, g.camera)
	slog.Info("inventory created")
	g.moveChan = make(chan unit.MoveMessage, moveChanBuffer)
	g.MapGrid = mapgrid.NewMapGrid(g.board, g.camera, g.moveChan)
	slog.Info("mapgrid created")
	g.createRocks()
	g.createUnits()

	slog.Info("game created")

	return &g
}

func getRandomPoint(board *board.Board) geom.Point {
	p := geom.Point{
		X: float64(rand.Uint64N(board.Width)),
		Y: float64(rand.Uint64N(board.Height)),
	}
	cell := board.Cell(p.GetInts())
	if math.IsInf(cell.Cost, 1) {
		return getRandomPoint(board)
	}
	return p
}

func (g *Game) createUnits() {
	unitPiece := g.inventory.Units["runner"]
	for range unitCount {
		chanWg := make(chan int64, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRunner := unitPiece.Unit(
			gofakeit.Name(),
			g.moveChan,
			chanCameraTicks,
			chanWg,
		)

		g.Units = append(g.Units, newRunner)
		newRunner.WG = &g.wg
		newRunner.Spawn(getRandomPoint(g.board))
		newRunner.Run()
	}
	slog.Info("units created")
}

func (g *Game) createRocks() {
	rockPiece := g.inventory.Units["rock"]
	for range rockCount {
		chanWg := make(chan int64, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRock := rockPiece.Unit(
			"Rock named "+gofakeit.Name(),
			g.moveChan,
			chanCameraTicks,
			chanWg,
		)
		g.Units = append(g.Units, newRock)
		newRock.WG = &g.wg
		newRock.Spawn(getRandomPoint(g.board))
		newRock.Run()
	}
	slog.Info("rocks created")
}
