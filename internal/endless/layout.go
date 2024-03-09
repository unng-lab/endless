package endless

func (g Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	W.Width.Store(int64(outsideWidth))
	W.Height.Store(int64(outsideHeight))
	return outsideWidth, outsideHeight
}
