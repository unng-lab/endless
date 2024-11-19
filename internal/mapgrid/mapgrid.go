package mapgrid

import (
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/geom"
	"github/unng-lab/madfarmer/internal/unit"
)

const (
	// gridSize размер ячейки грида в квадратах карты
	gridSize     = 16
	buffer       = 1000
	squareBuffer = 32
	unitCapacity = 64
)

type MapGrid struct {
	b                      *board.Board
	camera                 *camera.Camera
	minX, minY, maxX, maxY int
	Grid                   [][]map[*unit.Unit]struct{}
	Moves                  chan unit.MoveMessage
	Ticks                  chan struct{}
}

func NewMapGrid(b *board.Board, camera *camera.Camera, moves chan unit.MoveMessage) *MapGrid {
	var m MapGrid
	m.b = b
	m.camera = camera
	squareSize := board.CountTile / gridSize
	m.Grid = make([][]map[*unit.Unit]struct{}, squareSize)
	for i := range m.Grid {
		m.Grid[i] = make([]map[*unit.Unit]struct{}, squareSize)
	}
	m.Ticks = make(chan struct{}, 1)
	m.Moves = moves

	m.minX, m.minY, m.maxX, m.maxY = 0, 0, board.CountTile/gridSize-1, board.CountTile/gridSize-1

	go m.run()

	return &m
}

func (m *MapGrid) hash(pos geom.Point) (int, int) {
	x := int(pos.X / gridSize)
	y := int(pos.Y / gridSize)
	if x < m.minX {
		x = m.minX
	}
	if y < m.minY {
		y = m.minY
	}
	if x > m.maxX {
		x = m.maxX
	}
	if y > m.maxY {
		y = m.maxY
	}

	return x, y
}

func (m *MapGrid) run() {
	for {
		select {
		case <-m.Ticks:
			m.setUnitsOnboard()
		case msg := <-m.Moves:
			m.DeleteUnit(msg.From, msg.U)
			m.AddUnit(msg.To, msg.U)
		}
	}
}

func (m *MapGrid) DeleteUnit(from geom.Point, u *unit.Unit) {
	x, y := m.hash(from)
	if m.Grid[x][y] != nil {
		delete(m.Grid[x][y], u)
		if len(m.Grid[x][y]) == 0 {
			m.Grid[x][y] = nil
		}
	}
}

func (m *MapGrid) AddUnit(to geom.Point, u *unit.Unit) {
	x, y := m.hash(to)
	if m.Grid[x][y] == nil {
		m.Grid[x][y] = make(map[*unit.Unit]struct{}, unitCapacity)
	}
	m.Grid[x][y][u] = struct{}{}
}

func (m *MapGrid) setUnitsOnboard() {
	x1, y1 := m.hash(m.camera.Coordinates.Min)
	x2, y2 := m.hash(m.camera.Coordinates.Max)
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
