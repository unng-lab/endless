package unit

import (
	"github/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Relocate(p geom.Point) {
	u.Position.X = p.X
	u.Position.Y = p.Y
}

func (u *Unit) MoveToNeighbor(direction geom.Direction) {
	p := u.Position.GetNeighbor(direction)
	u.Relocate(p)
}
