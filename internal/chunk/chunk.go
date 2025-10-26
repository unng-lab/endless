// chunk/chunk.go
package chunk

import (
	"math/rand/v2"
	"sync"

	"github.com/unng-lab/madfarmer/internal/geom"
)

const (
	ChunkSize = 32 // tiles per chunk (square)
)

// Grid представляет собой простую карту проходимых тайлов
type Grid struct {
	W, H  int
	cells []byte // 0 проходимый, 1 заблокированный
}

// NewGrid создает новую occupancy grid размера w x h
func NewGrid(w, h int) *Grid {
	return &Grid{W: w, H: h, cells: make([]byte, w*h)}
}

// InBounds проверяет, что координаты лежат в пределах сетки
func (g *Grid) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.W && y < g.H
}

// SetBlocked помечает тайл как блокированный/проходимый
func (g *Grid) SetBlocked(x, y int, b bool) {
	if !g.InBounds(x, y) {
		return
	}
	if b {
		g.cells[g.idx(x, y)] = 1
	} else {
		g.cells[g.idx(x, y)] = 0
	}
}

// Blocked возвращает true если тайл заблокирован
func (g *Grid) Blocked(x, y int) bool {
	if !g.InBounds(x, y) {
		return true
	}
	return g.cells[g.idx(x, y)] == 1
}

// idx возвращает индекс в линейном массиве из координат x,y
func (g *Grid) idx(x, y int) int {
	return y*g.W + x
}

// Chunk хранит данные чанка: идентификатор, версию, occupancy grid и NavMesh
type Chunk struct {
	ID      geom.ChunkID
	Version uint64
	Grid    *Grid
}

// ChunkManager отвечает за загрузку/генерацию чанков
type ChunkManager struct {
	m      sync.Mutex
	chunks map[geom.ChunkID]*Chunk
}

// NewChunkManager создает пустой менеджер чанков
func NewChunkManager() *ChunkManager {
	return &ChunkManager{chunks: make(map[geom.ChunkID]*Chunk)}
}

// EnsureLoaded гарантирует, что чанк с указанным ID загружен
func (m *ChunkManager) EnsureLoaded(id geom.ChunkID) *Chunk {
	m.m.Lock()
	defer m.m.Unlock()
	if c, ok := m.chunks[id]; ok {
		return c
	}

	// Создаем новый чанк
	c := &Chunk{ID: id}
	c.Grid = NewGrid(ChunkSize, ChunkSize)

	// Генерируем препятствия
	seed := uint64((id.X * 73856093) ^ (id.Y * 19349663))
	r := rand.New(rand.NewPCG(seed, seed+1))
	for y := 0; y < ChunkSize; y++ {
		for x := 0; x < ChunkSize; x++ {
			if r.Float64() < 0.12 { // плотность препятствий
				c.Grid.SetBlocked(x, y, true)
			}
		}
	}

	m.chunks[id] = c
	return c
}

// Get безопасно возвращает чанк; возвращает nil если чанк не загружен
func (m *ChunkManager) Get(id geom.ChunkID) *Chunk {
	return m.chunks[id]
}
