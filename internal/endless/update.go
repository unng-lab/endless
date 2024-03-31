package endless

func (g *Game) Update() error {
	if err := g.camera.Update(); err != nil {
		return err
	}
	if err := g.ui.Update(); err != nil {
		return err
	}
	for i := range g.Units {
		if g.Units[i].wg.S > 0 {
			g.Units[i].wg.S--
		} else {
			g.wg.Add(1)
			g.Units[i].c <- &g.Units[i].wg
		}
	}
	g.wg.Wait()
	return nil
}
