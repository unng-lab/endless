package dstar

import "github/unng-lab/madfarmer/internal/geom"

// Node представляет узел в графе.
type Node struct {
	Position  geom.Point
	G         float64
	RHS       float64
	Key       [2]float64
	Neighbors []*Node
	InQueue   bool
	Index     int // Индекс в приоритетной очереди.
}
