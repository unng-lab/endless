package mapgrid

import (
	"sync"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/geom"
	"github/unng-lab/madfarmer/internal/unit"
)

const (
	gridSize     = 16
	buffer       = 1000
	squareBuffer = 32
	unitCapacity = 64
)

var (
	pool = sync.Pool{
		New: func() any {
			return make(map[*unit.Unit]struct{}, unitCapacity)
		},
	}
)

type MapGrid struct {
	b      *board.Board
	camera *camera.Camera
	Grid   [][]map[*unit.Unit]struct{}
	Moves  chan MoveMessage
	Ticks  chan struct{}
}

type MoveMessage struct {
	u        *unit.Unit
	from, to geom.Point
}

func NewMapGrid(b *board.Board, camera *camera.Camera) *MapGrid {
	var m MapGrid
	m.b = b
	m.camera = camera
	squareSize := board.CountTile / gridSize
	m.Grid = make([][]map[*unit.Unit]struct{}, squareSize)
	for i := range m.Grid {
		m.Grid[i] = make([]map[*unit.Unit]struct{}, squareSize)
		for j := range m.Grid[i] {
			m.Grid[i][j] = make(map[*unit.Unit]struct{})
		}
	}
	m.Moves = make(chan MoveMessage, buffer)
	m.Ticks = make(chan struct{}, 1)

	return &m
}

func hash(pos geom.Point) (int, int) {
	x := int(pos.X / gridSize)
	y := int(pos.Y / gridSize)
	return x, y
}

func (m *MapGrid) run() {
	for {
		select {
		case <-m.Ticks:
			m.setUnitsOnbord()
		case msg := <-m.Moves:
			x, y := hash(msg.from)
			delete(m.Grid[x][y], msg.u)
			x, y = hash(msg.to)
			m.Grid[x][y][msg.u] = struct{}{}
		}
	}
}

func (m *MapGrid) setUnitsOnbord() {
	x1, y1 := hash(m.camera.Coordinates.Min)
	x2, y2 := hash(m.camera.Coordinates.Max)
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			if m.Grid[x][y] != nil {
				for u := range m.Grid[x][y] {
					u.SetOnBoard(true)
				}
			}
		}
	}
}
