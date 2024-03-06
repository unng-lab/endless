package ebitenfx

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	//g.log.Debug("layout", "width", outsideWidth, "height", outsideHeight)
	g.window.Width.Store(uint64(outsideWidth))
	g.window.Height.Store(uint64(outsideHeight))
	return outsideWidth, outsideHeight
}
