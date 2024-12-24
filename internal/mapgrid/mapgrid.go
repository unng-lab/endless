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
	Ticks                  chan int64
	updated                bool
	lastUpdateTick         int64
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
	m.Ticks = make(chan int64, 1)
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
		case gtc := <-m.Ticks:
			m.setUnitsOnboard()
			m.SetUpdatedTick(gtc)
		case msg := <-m.Moves:
			m.process(msg)

		}
	}
}

func (m *MapGrid) process(msg unit.MoveMessage) {
	hashFromX, hashFromY := m.hash(msg.From)
	hashToX, hashToY := m.hash(msg.To)
	if hashFromX != hashToX || hashFromY != hashToY {
		if m.Grid[hashFromX][hashFromY] != nil {
			delete(m.Grid[hashFromX][hashFromY], msg.U)
			if len(m.Grid[hashFromX][hashFromY]) == 0 {
				m.Grid[hashFromX][hashFromY] = nil
			}
		}
		if m.Grid[hashToX][hashToY] == nil {
			m.Grid[hashToX][hashToY] = make(map[*unit.Unit]struct{}, unitCapacity)
		}
		m.Grid[hashToX][hashToY][msg.U] = struct{}{}
		m.updated = true
	}
}

func (m *MapGrid) SetUpdatedTick(tick int64) {
	if m.updated {
		m.lastUpdateTick = tick
		m.updated = false
	}

}

func (m *MapGrid) UpdatedAt() int64 {
	return m.lastUpdateTick
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
