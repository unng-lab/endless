package unit

// Update возвращает время до следующего вызова Update или ошибку
func (u *Unit) Update() (int, error) {
	// Как только юнит уходит с доски то он никогда не будет в фокусе

	u.Focused = false

	//slog.Info("unit position: ", "X: ", u.Position.X, "Y: ", u.Position.Y)
	return u.Tasks.Run(), nil
}

func (u *Unit) SetTask() {
	if u.Type == "runner" && u.Tasks.Current() == nil {
		// временное для добавление сходу задания на попиздовать куда то
		//u.RoadTask = NewRoad(u.Board, u)
		//err := u.RoadTask.Path(
		//	geom.Pt(
		//		float64(rand.Intn(board.CountTile)),
		//		float64(rand.Intn(board.CountTile)),
		//	))
		//if err != nil {
		//	slog.Error("road task ", "error", err)
		//
		//} else {
		//	u.Tasks.Add(&u.RoadTask)
		//}

	}

	if u.Type == "rock" && u.Tasks.Current() == nil {
		u.Tasks.Add(NewNoopTask(u.Board, u))
	}
}
