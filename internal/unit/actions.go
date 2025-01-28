package unit

import (
	"log/slog"

	"github.com/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Relocate(from, to geom.Point) {
	select {

	case u.MoveChan <- MoveMessage{
		U:    u,
		From: from,
		To:   to,
	}:
	default:
		slog.Warn("Unit.Relocate: channel is full", "unitType", u.Type, "unit", u)

	}

	u.Positioning.Position.X = to.X
	u.Positioning.Position.Y = to.Y
	u.Positioning.PositionShiftModX = 0
	u.Positioning.PositionShiftModY = 0

	//slog.Info("Unit.Relocate", "from", from, "to", to)
}

//func (u *Unit) MoveToNeighbor(direction geom.Direction) {
//	p := u.Position.GetNeighbor(direction)
//	u.Relocate(p)
//}
