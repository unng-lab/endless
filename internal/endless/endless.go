package endless

import (
	"log/slog"
	"runtime"
	"sync"

	"github.com/brianvoe/gofakeit/v7"

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
	log            *slog.Logger
	camera         *camera.Camera
	wg             sync.WaitGroup
	ui             *ui.UIEngine
	inventory      *Inventory
	board          *board.Board
	Units          []*unit.Unit
	UnitsMutex     sync.Mutex
	OnBoardCounter int64
	MapGrid        *mapgrid.MapGrid
	moveChan       chan unit.MoveMessage
	workersPool    []chan int64
}

func NewGame(
// analyticsDB *ch.AnaliticsDB,
) *Game {
	var g Game
	g.Units = make([]*unit.Unit, 0, unitCount)
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
	//g.MapGrid = mapgrid.NewMapGrid(g.board, g.camera, g.moveChan)
	slog.Info("mapgrid created")
	g.createRocks()
	g.createUnits()

	g.runWorkers()

	slog.Info("game created")

	return &g
}

func (g *Game) runWorkers() {
	num := runtime.NumCPU() / 2
	for i := range num {
		ch := make(chan int64, 1)
		g.workersPool = append(g.workersPool, ch)
		go g.workerRun(i, num, ch)
	}
}

func (g *Game) createUnits() {
	unitPiece := g.inventory.Units["runner"]
	for range unitCount {
		chanWg := make(chan int64, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRunner := unitPiece.Unit(
			len(g.Units),
			gofakeit.Name(),
			g.moveChan,
			chanCameraTicks,
			chanWg,
		)

		g.Units = append(g.Units, newRunner)
		newRunner.WG = &g.wg
		newRunner.Spawn(g.board.GetRandomPoint())
		newRunner.Run()
		newRunner.SetTask()
	}
	slog.Info("units created")
}

func (g *Game) createRocks() {
	rockPiece := g.inventory.Units["rock"]
	for range rockCount {
		chanWg := make(chan int64, 1)
		chanCameraTicks := make(chan struct{}, 1)
		newRock := rockPiece.Unit(
			len(g.Units),
			"Rock named "+gofakeit.Name(),
			g.moveChan,
			chanCameraTicks,
			chanWg,
		)
		g.Units = append(g.Units, newRock)
		newRock.WG = &g.wg
		newRock.Spawn(g.board.GetRandomPoint())
		newRock.Run()
		newRock.SetTask()
	}
	slog.Info("rocks created")
}
