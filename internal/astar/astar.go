package astar

import (
	"errors"
	"log/slog"
	"sync"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

const (
	pathCapacity  = 32
	queueCapacity = 512
	smallCapacity = 8
	costsCapacity = 32
	fromsCapacity = 32
	costDiagonal  = 1.414
)

const (
	DirUp byte = iota
	DirUpRight
	DirRight
	DirDownRight
	DirDown
	DirDownLeft
	DirLeft
	DirUpLeft
)

var errNoPath = errors.New("no Path")

var (
	bigPool = sync.Pool{
		New: func() any {
			return make([]Item, 0, queueCapacity)
		},
	}
	smallPool = sync.Pool{
		New: func() any {
			return make([]Item, 0, smallCapacity)
		},
	}
	costsPool = sync.Pool{
		New: func() any {
			return make(map[Item]float64, costsCapacity)
		},
	}
	fromsPool = sync.Pool{
		New: func() any {
			return make(map[Item]Item, costsCapacity)
		},
	}
)

type Astar struct {
	B     *board.Board
	items []Item
	costs map[Item]float64
	froms map[Item]Item
	Path  []geom.Point
}

func NewAstar(b *board.Board) Astar {
	return Astar{
		B:    b,
		Path: make([]geom.Point, 0, pathCapacity),
	}
}

func (a *Astar) Len() int {
	return len(a.items)
}

func (a *Astar) Less(i, j int) bool {
	return a.items[i].priority < a.items[j].priority
}

func (a *Astar) Swap(i, j int) {
	a.items[i], a.items[j] = a.items[j], a.items[i]
}

func (a *Astar) Push(x Item) {
	a.items = append(a.items, x) // append to end
	a.up(a.Len() - 1)
}

func (a *Astar) Pop() Item {
	n := a.Len() - 1
	a.Swap(0, n)
	a.down(0, n)
	item := a.items[len(a.items)-1]
	a.items = a.items[0 : len(a.items)-1]
	return item
}

func (a *Astar) ResetPath() {
	if len(a.Path) > pathCapacity {
		slog.Debug("reset path", "len", len(a.Path))
		a.Path = make([]geom.Point, 0, pathCapacity)
	} else {
		a.Path = a.Path[:0]
	}
}

func (a *Astar) BuildPath(fromX, fromY, toX, toY float64) error {
	a.ResetPath()
	defer func() {
		if cap(a.items) > 8*smallCapacity {
			bigPool.Put(a.items[:0])
		} else {
			smallPool.Put(a.items[:0])
		}

		if len(a.costs) < 8*costsCapacity {
			clear(a.costs)
			costsPool.Put(a.costs)
		}

		if len(a.froms) < 8*fromsCapacity {
			clear(a.froms)
			fromsPool.Put(a.froms)
		}

		a.items = nil
		a.costs = nil
		a.froms = nil
	}()

	if fromX == toX && fromY == toY {
		return nil
	}
	from := Item{
		x: fromX,
		y: fromY,
	}

	if from.heuristic(toX, toY) > 8*smallCapacity {
		a.items = bigPool.Get().([]Item)
	} else {
		a.items = smallPool.Get().([]Item)
	}

	a.costs = costsPool.Get().(map[Item]float64)
	a.froms = fromsPool.Get().(map[Item]Item)

	a.Push(from)

	var newPoint geom.Point
	var dir geom.Direction
	for a.Len() > 0 {
		current := a.Pop()
		if current.x == toX && current.y == toY {
			for !(current.x == fromX && current.y == fromY) {
				newPoint = geom.Pt(current.x, current.y)
				// всегда добавляем первый элемент
				if len(a.Path) == 0 {
					a.Path = append(a.Path, newPoint)
					current = a.froms[current]
					continue
				}

				newDir := a.Path[len(a.Path)-1].To(newPoint)
				if newDir != dir {
					a.Path = append(a.Path, newPoint)
					dir = newDir
				} else {
					a.Path[len(a.Path)-1] = newPoint
				}

				current = a.froms[current]
			}
			lastPoint := geom.Pt(fromX, fromY)
			if a.Path[len(a.Path)-1].To(lastPoint) != dir {
				a.Path = append(a.Path, lastPoint)
			} else {
				a.Path[len(a.Path)-1] = lastPoint
			}

			a.Path = a.Path[:len(a.Path)-1]

			return nil
		}

		for i := range neighbors {
			neighbor := Item{
				x:        current.x + neighbors[i].X,
				y:        current.y + neighbors[i].Y,
				priority: 0,
			}

			score := a.B.GetCell(int(neighbor.x), int(neighbor.y)).MoveCost()
			if score <= 0 {
				continue
			}
			if neighbors[i].X != 0 && neighbors[i].Y != 0 {
				score = score * costDiagonal
			}
			totalScore := a.costs[current] + score
			if oldScore, ok := a.costs[neighbor]; !ok || totalScore < oldScore {
				a.costs[neighbor] = totalScore
				neighbor.priority = totalScore + neighbor.heuristic(toX, toY)
				a.Push(neighbor)
				a.froms[neighbor] = current
			}
		}
	}

	return errNoPath
}

func (a *Astar) reversePath() {
	for i, j := 0, len(a.Path)-1; i < j; i, j = i+1, j-1 {
		a.Path[i], a.Path[j] = a.Path[j], a.Path[i]
	}

}

// Fix re-establishes the heap ordering after the element at index i has changed its value.
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling [Remove](h, i) followed by a Push of the new value.
// The complexity is O(log n) where n = h.Len().
func (a *Astar) Fix(i int) {
	if !a.down(i, a.Len()) {
		a.up(i)
	}
}

func (a *Astar) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !a.Less(j, i) {
			break
		}
		a.Swap(i, j)
		j = i
	}
}

func (a *Astar) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && a.Less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !a.Less(j, i) {
			break
		}
		a.Swap(i, j)
		i = j
	}
	return i > i0
}

//func reverse(dir byte) byte {
//	switch dir {
//	case DirRight:
//		return DirLeft
//	case DirLeft:
//		return DirRight
//	case DirUp:
//		return DirDown
//	case DirDown:
//		return DirUp
//	case DirUpRight:
//		return DirDownLeft
//	case DirDownRight:
//		return DirUpLeft
//	case DirDownLeft:
//		return DirUpRight
//	case DirUpLeft:
//		return DirDownRight
//	default:
//		panic("unreachable")
//	}
//}
