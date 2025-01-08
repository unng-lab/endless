package unit

import (
	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/geom"
	"math/rand"
)

// Update возвращает время до следующего вызова Update или ошибку
func (u *Unit) Update() (int, error) {
	// Как только юнит уходит с доски то он никогда не будет в фокусе
	if u.OnBoard.Load() {
		u.OnBoard.Store(!u.Camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y)))
	} else {
		if u.Focused {
			u.Focused = false
		}
	}

	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)
	return u.Tasks.Run(), nil
}

func (u *Unit) SetTask() {
	if u.Type == "runner" && u.Tasks.Current() == nil {
		// временное для добавление сходу задания на попиздовать куда то
		u.RoadTask = NewRoad(u.Board, u)
		if err := u.RoadTask.Path(
			geom.Pt(
				float64(rand.Intn(board.CountTile)),
				float64(rand.Intn(board.CountTile)),
			)); err != nil {
			panic(err)
		}

		u.Tasks.Add(&u.RoadTask)
	}

	if u.Type == "rock" && u.Tasks.Current() == nil {
		u.Tasks.Add(NewNoopTask(u.Board, u))
	}
}
