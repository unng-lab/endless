package endless

var Counter int64

func (g *Game) Update() error {
	Counter++
	return UI.Update()
}
