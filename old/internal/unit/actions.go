package unit

import (
	"github.com/unng-lab/endless/internal/geom"
)

func (u *Unit) Relocate(from, to geom.Point) {
	cell := u.Board.Cell(from.GetInts())
	err := cell.RemoveUnit(u.ID)
	if err != nil {
		panic(err)
	}
	u.set(to)
	return
	//slog.Info("Unit.Relocate", "from", from, "to", to)
}

func (u *Unit) Spawn(to geom.Point) {
	u.set(to)
}

func (u *Unit) set(to geom.Point) {
	cell := u.Board.Cell(to.GetInts())
	err := cell.AddUnit(u.ID, u.Index, u.Cost())
	if err != nil {
		//panic(err)
	}

	u.Positioning.Position.X = to.X
	u.Positioning.Position.Y = to.Y
	u.Positioning.PositionShiftModX = 0
	u.Positioning.PositionShiftModY = 0
}
