package endless

func (g *Game) Update() error {
	for i := range g.Units {
		if g.Units[i].wg.S > 0 {
			g.Units[i].wg.S--
		} else {
			g.wg.Add(1)
			g.Units[i].c <- &g.Units[i].wg
		}
	}
	g.wg.Wait()
	return g.ui.Update()
}
