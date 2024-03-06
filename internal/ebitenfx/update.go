package ebitenfx

func (g *Game) Update() error {
	if err := g.scr.Update(); err != nil {
		return err
	}
	if err := g.ui.Update(); err != nil {
		return err
	}
	return nil
}
