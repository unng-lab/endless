package endless

import (
	"log/slog"
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
	moveChanBuffer = 1000000 // 1 миллион

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
	moveChan := make(chan unit.MoveMessage, moveChanBuffer)
	unitPiece := g.inventory.Units["runner"]
	for i := range unitCount {
		chanWg := make(chan *sync.WaitGroup, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRunner := unitPiece.Unit(
			i,
			gofakeit.Name(),
			moveChan,
			chanCameraTicks,
			chanWg,
		)

		g.Units = append(g.Units, newRunner)
		newRunner.Relocate(geom.Pt(0, 0), getRandomPoint(g.board))
		newRunner.Run()
	}
	slog.Info("units created")
	rockPiece := g.inventory.Units["rock"]
	for i := range rockCount {
		chanWg := make(chan *sync.WaitGroup, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRock := rockPiece.Unit(
			i,
			"Rock named "+gofakeit.Name(),
			moveChan,
			chanCameraTicks,
			chanWg,
		)
		g.Units = append(g.Units, newRock)
		newRock.Relocate(geom.Pt(0, 0), getRandomPoint(g.board))
		newRock.Run()
	}
	slog.Info("rocks created")

	g.MapGrid = mapgrid.NewMapGrid(g.board, g.camera, moveChan)
	slog.Info("mapgrid created")
	slog.Info("game created")

	return &g
}

func getRandomPoint(board *board.Board) geom.Point {
	return geom.Point{
		X: float64(rand.Uint64N(board.Width)),
		Y: float64(rand.Uint64N(board.Height)),
	}
}
