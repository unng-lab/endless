package unit

import (
	"github/unng-lab/madfarmer/internal/astar"
)

var _ Task = new(Road)

type Road struct {
	astar.Astar
}

func (r *Road) Next() (Action, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Road) GetName() string {
	//TODO implement me
	panic("implement me")
}

func (r *Road) GetDescription() string {
	//TODO implement me
	panic("implement me")
}
