package unit

import (
	"math/rand"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

// Update возвращает время до следующего вызова Update или ошибку
func (u *Unit) Update() (int, error) {
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
			return 0, err
		}
		u.Status = UnitStatusRunning
	default:
		panic("unknown unit status")
	}
	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)

	u.OnBoard = !u.Camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y))

	return 0, nil
}
