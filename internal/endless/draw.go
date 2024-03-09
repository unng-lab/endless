package endless

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) Draw(screen *ebiten.Image) {
	B.Draw(screen)
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"TPS: %0.2f\nFPS: %0.2f\nCameraX: %0.2f\nCameraY: %0.2f\nZoom: %0.2f\nCellNumber: %d",
			ebiten.ActualTPS(),
			ebiten.ActualFPS(),
			C.GetPositionX(),
			C.GetPositionY(),
			C.GetZoomFactor(),
			B.GetCellNumber(),
		),
	)
}
