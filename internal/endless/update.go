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

const procCacheLineSize = 16

func (g *Game) workersProcess(offset, numWorkers int, gameTickCounter int64) {
	defer g.wg.Done()
	for i := offset * procCacheLineSize; i < len(g.Units); i += numWorkers * procCacheLineSize {
		for j := 0; j < procCacheLineSize; j++ {
			if i+j >= len(g.Units) {
				break
			}
			unitStatus := g.Units[j+i].Process()
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
}
