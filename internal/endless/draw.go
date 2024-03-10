package endless

import "C"
import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var Counter int

func (g *Game) Draw(screen *ebiten.Image) {
	Counter++
	camera := g.camera
	B.Draw(screen, camera)
	for i := range g.Units {
		g.Units[i].Draw(screen, Counter, camera)
	}
	x, y := ebiten.CursorPosition()
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(`TPS: %0.2f
FPS: %0.2f
CameraX: %0.2f
CameraY: %0.2f
Zoom: %0.2f
CellNumber: %d,
UnitNumber: %d,
CursorX: %d,
CursorY: %d,
CellX: %d,
CellY: %d`,
			ebiten.ActualTPS(),
			ebiten.ActualFPS(),
			g.camera.GetPositionX(),
			g.camera.GetPositionY(),
			g.camera.GetZoomFactor(),
			B.GetCellNumber(),
			len(g.Units),
			x,
			y,
			(float64(x)+g.camera.GetPositionX())/g.camera.GetTileSize()+1,
			(float64(y)+g.camera.GetPositionY())/g.camera.GetTileSize()+1,
		),
	)
}
