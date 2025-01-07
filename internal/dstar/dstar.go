package dstar

import (
	"container/heap"
	"errors"
	"math"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

const (
	pathCapacity  = 32
	queueCapacity = 512
	smallCapacity = 8
	costsCapacity = 32
	fromsCapacity = 32
)

var errNoPath = errors.New("no Path")

type DStar struct {
	B           *board.Board
	start, goal *Node
	nodes       []*Node
}

func NewDstar(b *board.Board) *DStar {
	return &DStar{
		B:     b,
		start: nil,
		goal:  nil,
		nodes: make([]*Node, 0, queueCapacity),
	}
}

func (ds *DStar) Path(start, goal geom.Point) ([]geom.Point, error) {
	if start == goal {
		return nil, errNoPath
	}
	ds.start = NewNode(start, ds.B)
	ds.goal = NewNode(goal, ds.B)
	ds.Push(ds.goal)
	return nil, nil
}

func (ds *DStar) CalculateKey(u *Node) [2]float64 {
	k := [2]float64{
		math.Min(u.G, u.RHS) + ds.start.heuristic(u.Position),
		math.Min(u.G, u.RHS),
	}
	return k
}

func (ds *DStar) ComputeShortestPath() {
	i := 0
	for len(ds.nodes) > 0 && (ds.nodes[0].Key[0] < ds.CalculateKey(ds.Start)[0] ||
		(ds.nodes[0].Key[0] == ds.CalculateKey(ds.start)[0] && ds.nodes[0].Key[1] < ds.CalculateKey(ds.Start)[1]) ||
		ds.start.RHS != ds.start.G) {
		i++
		u := ds.Pop()
		if u.G > u.RHS {
			u.G = u.RHS
			for _, s := range u.Neighbors {
				ds.UpdateVertex(s)
			}
		} else {
			u.G = math.Inf(1)
			ds.UpdateVertex(u)
			for _, s := range u.Neighbors {
				ds.UpdateVertex(s)
			}
		}
	}
	println("\nИтераций", i)
}

func (ds *DStar) UpdateVertex(u *Node) {
	if u != ds.goal {
		minRhs := math.Inf(1)
		for _, s := range u.Neighbors {
			if cost := s.G + ds.Cost(u, s, 0); cost < minRhs {
				minRhs = cost
			}
		}
		u.RHS = minRhs
	}
	ds.Remove(u)
	if u.G != u.RHS {
		ds.UpdateKey(u)
		ds.Push(u)
	}
}

func (ds *DStar) UpdateKey(u *Node) {
	u.Key = ds.CalculateKey(u)
}

func (ds *DStar) Cost(u *Node, s *Node, tick int64) float64 {
	if u.Obstacle || s.Obstacle {
		return math.Inf(1)
	}
	// Здесь может быть логика расчета стоимости, например, по-разному для соседей по диагонали и по прямой
	return ds.B.GetCost(u.Position, s.Position, 0)
}
