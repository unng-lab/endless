package endless

func (g *Game) Update() error {
	g.tickCounter++
	if err := g.camera.Update(); err != nil {
		return err
	}

	g.board.ClearUpdatedCells()
	g.action.Update()
	for i := range g.workersPool {
		g.wg.Add(1)
		g.workersPool[i] <- g.tickCounter
	}
	g.wg.Wait()
	return nil
}

func (g *Game) workerRun(offset, shift int, gameTickChan <-chan int64) {
	for gameCounter := range gameTickChan {
		g.workersProcess(offset, shift, gameCounter)
	}
}

const procCacheLineSize = 16

func (g *Game) workersProcess(offset, numWorkers int, tick int64) {
	defer g.wg.Done()
	stride := numWorkers * procCacheLineSize
	for blockStart := offset * procCacheLineSize; blockStart < len(g.Units); blockStart += stride {
		for j := 0; j < procCacheLineSize; j++ {
			idx := blockStart + j
			if idx >= len(g.Units) {
				break
			}
			currentUnit := g.Units[idx]
			unitStatus := currentUnit.Process()
			if unitStatus.OnBoard {
				// сигнализируем о том, что мы находимся на карте и нужно работать с анимацией
				select {
				case currentUnit.CameraTicks <- struct{}{}:
				default:
					//g.log.Info("Camera ticks channel is full")
					//максимально не блокируем

				}
			}
			if currentUnit.SleepTicks > 0 {
				currentUnit.SleepTicks--
			} else {
				if currentUnit.Tasks.Current() != nil {
					g.wg.Add(1)
					currentUnit.Ticks <- tick
					// currentUnit.Play(tick)
				}
			}
		}
	}
}
