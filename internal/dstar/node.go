package dstar

import "github/unng-lab/madfarmer/internal/geom"

// Node представляет узел в графе.
type Node struct {
	Position  geom.Point // Позиция узла в пространстве.
	G         float64    // Стоимость пути от стартового узла до текущего.
	RHS       float64    // Оценка стоимости от текущего узла до целевого.
	Key       [2]float64 // Ключ узла для очереди с приоритетом.
	Neighbors []*Node    // Соседи текущего узла.
	InQueue   bool       // Флаг того, находится ли узел в очереди.
	Index     int        // Индекс в приоритетной очереди.
}
