// geom/geom.go
package geom

const (
	ChunkSize        = 32 // tiles per chunk (square)
	ClusterChunkSize = 4  // how many chunks per cluster side =
)

// Vec2 — целочисленные 2D-координаты
type Vec2 struct{ X, Y int }

// Add — сложение векторов
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Eq — сравнение на равенство
func (v Vec2) Eq(o Vec2) bool { return v.X == o.X && v.Y == o.Y }

// FloatX — преобразование X-координаты в float64
func (v Vec2) FloatX() float64 { return float64(v.X) }

// FloatY — преобразование Y-координаты в float64
func (v Vec2) FloatY() float64 { return float64(v.Y) }

// WorldToChunk — переводит глобальную позицию в идентификатор чанка и локальную позицию
func WorldToChunk(pos Vec2, chunkSize int) (ChunkID, Vec2) {
	x := pos.X
	y := pos.Y
	cx := floorDiv(x, chunkSize)
	cy := floorDiv(y, chunkSize)
	lx := x - cx*chunkSize
	ly := y - cy*chunkSize
	return ChunkID{cx, cy}, Vec2{lx, ly}
}

// floorDiv — целочисленное деление с округлением вниз
func floorDiv(a, b int) int {
	q := a / b
	r := a % b
	if r != 0 && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// ChunkID — координаты чанка в логической сетке чанков
type ChunkID struct{ X, Y int }

// ClusterID — идентификатор кластера
type ClusterID struct{ X, Y int }

// Dist2 — квадрат расстояния между двумя точками
func Dist2(a, b Vec2) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

// Max — возвращает большее из двух чисел
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min — возвращает меньшее из двух чисел
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Eq — сравнение на равенство
func (v ClusterID) Eq(o ClusterID) bool { return v.X == o.X && v.Y == o.Y }

// ClusterID — идентификатор кластера (группа чанков). Метод ClusterID вычисляет
// кластер, которому принадлежит чанк, по параметру ClusterChunkSize.
func (c ChunkID) ClusterID() ClusterID {
	// cluster grid groups ClusterChunkSize chunks
	cx := floorDiv(c.X, ClusterChunkSize)
	cy := floorDiv(c.Y, ClusterChunkSize)
	return ClusterID{cx, cy}
}
