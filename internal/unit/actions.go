package unit

import (
	"log/slog"

	"github/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Relocate(from, to geom.Point) {
	select {

	case u.MoveChan <- MoveMessage{
		U:    u,
		From: from,
		To:   to,
	}:
	default:
		slog.Warn("Unit.Relocate: channel is full")

	}

	u.Position.X = to.X
	u.Position.Y = to.Y
	u.PositionShiftModX = 0
	u.PositionShiftModY = 0

	slog.Info("Unit.Relocate", "from", from, "to", to)
}

//func (u *Unit) MoveToNeighbor(direction geom.Direction) {
//	p := u.Position.GetNeighbor(direction)
//	u.Relocate(p)
//}
