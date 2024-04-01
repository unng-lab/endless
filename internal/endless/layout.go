package endless

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.camera.W.Width = float64(outsideWidth)
	g.camera.W.Height = float64(outsideHeight)
	return outsideWidth, outsideHeight
}
