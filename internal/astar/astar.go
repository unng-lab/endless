package astar

import (
	"errors"
	"image"

	"github/unng-lab/madfarmer/internal/endless"
)

const (
	pathCapacity  = 256
	queueCapacity = 256
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

var errNoPath = errors.New("no path")

type Astar struct {
	b     *endless.Board
	items []Item
	path  []byte
}

func NewAstar(b *endless.Board) Astar {
	return Astar{
		b:     b,
		items: make([]Item, 0, queueCapacity),
		path:  make([]byte, 0, pathCapacity),
	}
}

func (a Astar) Len() int {
	return len(a.items)
}

func (a Astar) Less(i, j int) bool {
	return a.items[i].priority < a.items[j].priority
}

func (a Astar) Swap(i, j int) {
	a.items[i], a.items[j] = a.items[j], a.items[i]
}

func (a Astar) Push(x Item) {
	a.items = append(a.items, x) // append to end
}

func (a Astar) Pop() Item {
	item := a.items[len(a.items)-1]
	a.items = a.items[0 : len(a.items)-1]
	return item
}

func (a Astar) ResetPath() {
	a.path = a.path[:0]
}

func (a Astar) BuildPath(from, to image.Point) ([]byte, error) {
	a.ResetPath()
	if from == to {
		return a.path, nil
	}

	a.Push(Item{
		x:        from.X,
		y:        from.Y,
		priority: 0,
	})

	for a.Len() > 0 {

	}

	return nil, errNoPath
}

// Fix re-establishes the heap ordering after the element at index i has changed its value.
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling [Remove](h, i) followed by a Push of the new value.
// The complexity is O(log n) where n = h.Len().
func (a Astar) Fix(i int) {
	if !a.down(i, a.Len()) {
		a.up(i)
	}
}

func (a Astar) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !a.Less(j, i) {
			break
		}
		a.Swap(i, j)
		j = i
	}
}

func (a Astar) down(i0, n int) bool {
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
