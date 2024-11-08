package unit

import (
	"github/unng-lab/madfarmer/internal/astar"
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

var _ Task = new(Road)

type Road struct {
	unit     *Unit
	nextMove func() error
	position geom.Point
	astar.Astar
}

func NewRoad(b *board.Board, unit *Unit) Road {
	return Road{
		unit:  unit,
		Astar: astar.NewAstar(b),
	}
}

func (r *Road) Next() (int, error) {
	if r.nextMove != nil {
		if r.position == r.unit.Position {
			if err := r.nextMove(); err != nil {
				return 0, err
			}
		}
		// если передвинулся то надо логику обработать
	}
	nextPoint := r.unit.Position.GetNeighbor(
		r.unit.Position.To(r.Astar.Path[len(r.Astar.Path)-1]),
	)
	if nextPoint == r.Astar.Path[len(r.Astar.Path)-1] {
		r.Astar.Path = r.Astar.Path[:len(r.Astar.Path)-1]
	}
	timeToWalkOnePoint := r.unit.Speed *
		r.Astar.Path

	return 0, nil
}

func (r *Road) GetName() string {
	return "Walk Task"
}

func (r *Road) GetDescription() string {
	return "Task to walk"
}

func (r *Road) Path(to geom.Point) error {
	err := r.BuildPath(r.unit.Position.X, r.unit.Position.Y, to.X, to.Y)
	if err != nil {
		return err
	}
	return nil
}
