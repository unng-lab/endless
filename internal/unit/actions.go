package unit

import (
	"github/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Relocate(from, to geom.Point) {
	u.MoveChan <- MoveMessage{
		U:    u,
		From: from,
		To:   to,
	}
	u.Position.X = to.X
	u.Position.Y = to.Y
}

//func (u *Unit) MoveToNeighbor(direction geom.Direction) {
//	p := u.Position.GetNeighbor(direction)
//	u.Relocate(p)
//}
