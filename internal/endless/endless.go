package endless

import (
	"log/slog"
	"runtime"
	"sync"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/unng-lab/madfarmer/internal/actions"
	"github.com/unng-lab/madfarmer/internal/geom"

	"github.com/unng-lab/madfarmer/internal/board"

	"github.com/unng-lab/madfarmer/internal/camera"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/unit"
)

const (
	unitCount      = 1
	rockCount      = 0
	moveChanBuffer = 1000

	tileSize  = 16
	tileCount = 1024
)

var _ ebiten.Game = (*Game)(nil) // ensure Game implements ebiten.Game

type Game struct {
	log            *slog.Logger
	camera         *camera.Camera
	wg             sync.WaitGroup
	inventory      *Inventory
	board          *board.Board
	Units          []*unit.Unit
	UnitsMutex     sync.Mutex
	OnBoardCounter int64
	workersPool    []chan int64
	action         *actions.Action
}

func NewGame() *Game {
	var g Game
	g.log = slog.Default()
	g.Units = make([]*unit.Unit, 0, unitCount)
	g.camera = camera.New(tileSize, tileCount)
	g.log.Info("camera created")

	newBoard, err := board.NewBoard(g.camera, tileSize, tileSize, tileCount)
	if err != nil {
		panic(err)
	}
	g.board = newBoard
	g.log.Info("board created")
	g.action = actions.NewAction(g.camera)
	g.log.Info("action created")
	g.inventory = NewInverntory(g.board, g.camera)
	g.log.Info("inventory created")
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
			chanCameraTicks,
			chanWg,
		)

		g.Units = append(g.Units, newRunner)
		newRunner.WG = &g.wg
		//newRunner.Spawn(g.board.GetRandomFreePoint())
		newRunner.Spawn(geom.Point{X: 0, Y: 0})
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
			chanCameraTicks,
			chanWg,
		)
		g.Units = append(g.Units, newRock)
		newRock.WG = &g.wg
		newRock.Spawn(g.board.GetRandomFreePoint())
		newRock.Run()
		newRock.SetTask()
	}
	slog.Info("rocks created")
}
