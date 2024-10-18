package endless

func (g *Game) Update() error {
	g.OnBoard = g.OnBoard[:0]
	if err := g.camera.Update(); err != nil {
		return err
	}
	if err := g.ui.Update(); err != nil {
		return err
	}
	for i := range g.Units {
		if g.Units[i].OnBoard {
			// сигнализируем о том, что мы находимся на карте и нужно работать с анимацией
			select {
			case g.Units[i].CameraTicks <- struct{}{}:
			default:
				//максимально не блокируем

			}

			g.OnBoard = append(g.OnBoard, g.Units[i])
		}
		if g.Units[i].SleepTicks > 0 {
			g.Units[i].SleepTicks--
		} else {
			g.wg.Add(1)
			g.Units[i].Ticks <- &g.wg
		}
	}
	g.wg.Wait()

	return nil
}
