package endless

import "github/unng-lab/madfarmer/internal/window"

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	window.W.Width = float64(outsideWidth)
	window.W.Height = float64(outsideHeight)
	return outsideWidth, outsideHeight
}
