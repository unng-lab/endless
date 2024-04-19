package unit

import (
	"math/rand"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

// Update returns true if the unit are on the board
func (u *Unit) Update() error {
	u.OnBoard = !u.Camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y))
	u.Focused = false
	if u.OnBoard {
		if u.isFocused(u.Camera.Cursor) {
			u.Focused = true
		}
	}
	switch u.Status {
	case UnitStatusRunning:
		u.Move()
	case UnitStatusIdle:
		err := u.Pathing.BuildPath(u.Position.X, u.Position.Y, float64(rand.Intn(board.CountTile)), float64(rand.Intn(board.CountTile)))
		if err != nil {
			return err
		}
		u.Status = UnitStatusRunning
	}
	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)
	return nil
}
