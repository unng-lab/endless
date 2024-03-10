package endless

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var Counter int

func (g *Game) Draw(screen *ebiten.Image) {
	Counter++
	B.Draw(screen)
	for i := range g.Units {
		g.Units[i].Draw(screen, Counter)
	}
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"TPS: %0.2f\nFPS: %0.2f\nCameraX: %0.2f\nCameraY: %0.2f\nZoom: %0.2f\nCellNumber: %d,\nUnitNumber: %d",
			ebiten.ActualTPS(),
			ebiten.ActualFPS(),
			C.GetPositionX(),
			C.GetPositionY(),
			C.GetZoomFactor(),
			B.GetCellNumber(),
			len(g.Units),
		),
	)
}
