package unit

import (
	"log/slog"

	"github.com/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Relocate(from, to geom.Point) {
	cell := u.Board.Cell(from.GetInts())
	err := cell.RemoveUnit(u.ID)
	if err != nil {
		panic(err)
	}
	u.set(to)
	u.tellToMapGrid(from, to)
	return
	//slog.Info("Unit.Relocate", "from", from, "to", to)
}

func (u *Unit) Spawn(to geom.Point) {
	u.set(to)
	u.tellToMapGrid(geom.Point{}, to)
}

func (u *Unit) set(to geom.Point) {
	cell := u.Board.Cell(to.GetInts())
	err := cell.AddUnit(u.ID, u.Cost())
	if err != nil {
		panic(err)
	}

	u.Positioning.Position.X = to.X
	u.Positioning.Position.Y = to.Y
	u.Positioning.PositionShiftModX = 0
	u.Positioning.PositionShiftModY = 0
}

func (u *Unit) tellToMapGrid(from, to geom.Point) {
	select {

	case u.MoveChan <- MoveMessage{
		U:    u,
		From: from,
		To:   to,
	}:
	default:
		slog.Warn("Unit.Relocate: channel is full", "unitType", u.Type, "unit", u)
	}
}

//func (u *Unit) MoveToNeighbor(direction geom.Direction) {
//	p := u.Position.GetNeighbor(direction)
//	u.Relocate(p)
//}
