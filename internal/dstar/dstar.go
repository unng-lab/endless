package dstar

import (
	"errors"

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

type Dstar struct {
	B           *board.Board
	start, goal *Node
	nodes       []*Node
}

func NewDstar(b *board.Board) *Dstar {
	return &Dstar{
		B:     b,
		start: nil,
		goal:  nil,
		nodes: make([]*Node, 0, queueCapacity),
	}
}

func (d *Dstar) Path(start, goal geom.Point) ([]geom.Point, error) {
	if start == goal {
		return nil, errNoPath
	}
	d.start
}
