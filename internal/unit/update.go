package unit

import (
	"github/unng-lab/madfarmer/internal/geom"
)

// Update возвращает время до следующего вызова Update или ошибку
func (u *Unit) Update() (int, error) {
	// Как только юнит уходит с доски то он никогда не будет в фокусе
	if !u.OnBoard.Load() && u.Focused {
		u.Focused = false
	}
	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)

	u.OnBoard.Store(!u.Camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y)))

	return u.Tasks.Run(), nil
}
