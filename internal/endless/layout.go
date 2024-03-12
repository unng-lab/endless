package endless

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	W.Width = float64(outsideWidth)
	W.Height = float64(outsideHeight)
	return outsideWidth, outsideHeight
}
