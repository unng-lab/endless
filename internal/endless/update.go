package endless

import (
	"time"
)

var gameTickCounter int64

func (g *Game) Update() error {
	t := time.Now()
	gameTickCounter++
	if err := g.camera.Update(); err != nil {
		return err
	}
	if err := g.ui.Update(); err != nil {
		return err
	}
	g.board.ClearUpdatedCells()
	//select {
	//case g.MapGrid.Ticks <- gameTickCounter:
	//default:
	//	g.log.Info("MapGrid ticks channel is full")
	//}
	for i := range g.workersPool {
		g.wg.Add(1)
		g.workersPool[i] <- gameTickCounter
	}
	g.wg.Wait()
	println(time.Since(t).Microseconds())
	return nil
}

func (g *Game) workerRun(offset, shift int, gameTickChan chan int64) {
	for {
		select {
		case gameCounter := <-gameTickChan:
			g.workersProcess(offset, shift, gameCounter)
		}
	}
}

func (g *Game) workersProcess(offset, shift int, gameTickCounter int64) {
	defer g.wg.Done()
	for i := offset; i < len(g.Units); i += shift {
		unitStatus := g.Units[i].Process()
		if unitStatus.OnBoard {
			// сигнализируем о том, что мы находимся на карте и нужно работать с анимацией
			select {
			case g.Units[i].CameraTicks <- struct{}{}:
			default:
				g.log.Info("Camera ticks channel is full")
				//максимально не блокируем

			}
		}
		if g.Units[i].SleepTicks > 0 {
			g.Units[i].SleepTicks--
		} else {
			if g.Units[i].Tasks.Current() != nil {
				g.wg.Add(1)
				g.Units[i].Ticks <- gameTickCounter
				//g.Units[i].Play(gameTickCounter)
			}
		}
	}
}
