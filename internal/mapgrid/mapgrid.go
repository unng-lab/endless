package mapgrid

import (
	"sync"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/camera"
	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/unit"
)

const (
	// gridSize размер ячейки грида в квадратах карты
	gridSizeShift = 4 // gridSize = 16 (1 << 4)
	gridSize      = 1 << gridSizeShift
	buffer        = 1000
	squareBuffer  = 32
	unitCapacity  = 64
)

type MapGrid struct {
	board                  *board.Board
	camera                 *camera.Camera
	minX, minY, maxX, maxY int
	Grid                   [][]map[*unit.Unit]struct{}
	Points                 map[geom.Point]UnitList
	PointsMutex            sync.RWMutex
	Moves                  chan unit.MoveMessage
	Ticks                  chan int64
	updated                bool
	lastUpdateTick         int64
	onBoardList            map[*unit.Unit]struct{}
	onBoardListMutex       sync.RWMutex
	onBoardProcessChan     chan [3]int
	onBoardWriterChan      chan *unit.Unit
	onBoardWG              sync.WaitGroup
}

type UnitList struct {
	list []*unit.Unit
}

func (ul *UnitList) Add(u *unit.Unit) {
	ul.list = append(ul.list, u)
}

func (ul *UnitList) Remove(u *unit.Unit) int {
	for i := range ul.list {
		if ul.list[i] == u {
			ul.list = append(ul.list[:i], ul.list[i+1:]...)
			return len(ul.list)
		}
	}
	return len(ul.list)
}

func (m *MapGrid) getUnitsFromPos(pos geom.Point) []*unit.Unit {
	m.PointsMutex.RLock()
	defer m.PointsMutex.RUnlock()
	res := make([]*unit.Unit, 0, len(m.Points[pos].list))
	copy(res, m.Points[pos].list)
	return res
}

func NewMapGrid(board *board.Board, camera *camera.Camera, moves chan unit.MoveMessage) *MapGrid {
	var m MapGrid
	m.board = board
	m.camera = camera
	squareSize := board.Width / gridSize
	if board.Width%gridSize != 0 {
		squareSize++
	}
	m.Grid = make([][]map[*unit.Unit]struct{}, squareSize)
	for i := range m.Grid {
		m.Grid[i] = make([]map[*unit.Unit]struct{}, squareSize)
	}
	m.Ticks = make(chan int64, 1)
	m.Moves = moves
	m.onBoardList = make(map[*unit.Unit]struct{}, unitCapacity)
	m.minX, m.minY, m.maxX, m.maxY = 0, 0, int(board.Width)/gridSize-1, int(board.Height)/gridSize-1
	m.onBoardProcessChan = make(chan [3]int, squareSize/2)
	m.onBoardWriterChan = make(chan *unit.Unit, squareSize/2)
	go m.run(squareSize)

	return &m
}

func (m *MapGrid) hash(pos geom.Point) (int, int) {
	return max(min(int(pos.X)>>gridSizeShift, m.maxX), m.minX), max(min(int(pos.Y)>>gridSizeShift, m.maxY), m.minY)
}

func (m *MapGrid) run(workerPoolCount uint64) {
	for range workerPoolCount {
		go m.runSop()
	}
	go m.onBoardWriter()
	for {
		select {
		case <-m.Ticks:
			m.setUnitsOnboard()
		case msg := <-m.Moves:
			m.process(msg)
			m.board.AddUpdatedCells(msg.From, msg.To)
		}
	}
}

func (m *MapGrid) onBoardWriter() {
	for {
		select {
		case u := <-m.onBoardWriterChan:
			m.addUnitToList(u)
		}
	}
}

func (m *MapGrid) addUnitToList(u *unit.Unit) {
	defer m.onBoardWG.Done()
	m.onBoardList[u] = struct{}{}
}

func (m *MapGrid) runSop() {
	for {
		select {
		case x := <-m.onBoardProcessChan:
			m.subProcess(x)
		}
	}
}

func (m *MapGrid) subProcess(coords [3]int) {
	defer m.onBoardWG.Done()
	for y := coords[1]; y <= coords[2]; y++ {
		if m.Grid[coords[0]][y] != nil {
			m.onBoardWG.Add(len(m.Grid[coords[0]][y]))
			for u := range m.Grid[coords[0]][y] {
				m.onBoardWriterChan <- u
			}
		}
	}
}

func (m *MapGrid) process(msg unit.MoveMessage) {
	hashFromX, hashFromY := m.hash(msg.From)
	hashToX, hashToY := m.hash(msg.To)
	func() {
		m.PointsMutex.Lock()
		defer m.PointsMutex.Unlock()
		if msg.To != msg.From {
			fromList := m.Points[msg.From]
			fromList.Remove(msg.U)
			toList := m.Points[msg.To]
			toList.Add(msg.U)
		}
	}()

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
	m.onBoardListMutex.Lock()
	defer m.onBoardListMutex.Unlock()
	clear(m.onBoardList)
	x1, y1 := m.hash(m.camera.Coordinates.Min)
	x2, y2 := m.hash(m.camera.Coordinates.Max)
	for x := x1; x <= x2; x++ {
		m.onBoardWG.Add(1)
		m.onBoardProcessChan <- [3]int{x, y1, y2}
	}
	m.onBoardWG.Wait()
}

func (m *MapGrid) CheckOnBoard(u *unit.Unit) bool {
	// код через трайлок приводит к потере производительности в 2 раза
	// код через рлок на текущий момент оптимален
	//if m.onBoardListMutex.TryRLock() {
	//	defer m.onBoardListMutex.RUnlock()
	//	_, ok := m.onBoardList[u]
	//	return ok
	//}
	//return true

	m.onBoardListMutex.RLock()
	defer m.onBoardListMutex.RUnlock()
	_, ok := m.onBoardList[u]
	return ok
}
