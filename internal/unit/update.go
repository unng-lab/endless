package unit

import (
	"github/unng-lab/madfarmer/internal/geom"
)

// Update возвращает время до следующего вызова Update или ошибку
func (u *Unit) Update() (int, error) {
	// Изменить логику тк апдейт теперь прокает после сна
	u.Focused = false
	if u.OnBoard {
		if u.isFocused(u.Camera.Cursor) {
			u.Focused = true
		}
	}
	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)

	u.OnBoard = !u.Camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y))

	return u.Tasks.Run(), nil
}
