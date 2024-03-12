package endless

import "C"
import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var Counter int

func (g *Game) Draw(screen *ebiten.Image) {
	Counter++
	camera := g.camera
	B.Draw(screen, camera)
	for i := range g.Units {
		g.Units[i].Draw(screen, Counter, camera)
	}
	a, b := ebiten.CursorPosition()
	x, y := float64(a), float64(b)
	shiftX, shiftY := math.Mod(camera.GetPositionX(), camera.GetTileSize()), math.Mod(camera.GetPositionY(), camera.GetTileSize())
	if shiftX < 0 {
		shiftX = -shiftX
	}
	if shiftY < 0 {
		shiftY = -shiftY
	}
	posX, posY := math.Floor((x+shiftX)/camera.GetTileSize()), math.Floor((y+shiftY)/camera.GetTileSize())
	posXClear, posYClear := float32(posX)*float32(camera.GetTileSize()), float32(posY)*float32(camera.GetTileSize())
	vector.DrawFilledRect(
		screen,
		posXClear-float32(shiftX),
		posYClear-float32(shiftY),
		float32(camera.GetTileSize()),
		float32(camera.GetTileSize()),
		color.White,
		false,
	)

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(`TPS: %0.2f
FPS: %0.2f
CameraX: %0.2f
CameraY: %0.2f
Zoom: %0.2f
CellNumber: %d
UnitNumber: %d
TileSize: %0.2f
posX: %0.2f
posY: %0.2f
shiftX: %0.2f
shiftY: %0.2f
posXClear: %0.2f
posYClear: %0.2f
CursorX: %0.2f
CursorY: %0.2f
CellX: %0.2f
CellY: %0.2f`,
			ebiten.ActualTPS(),
			ebiten.ActualFPS(),
			g.camera.GetPositionX(),
			g.camera.GetPositionY(),
			g.camera.GetZoomFactor(),
			B.GetCellNumber(),
			len(g.Units),
			camera.GetTileSize(),
			posX,
			posY,
			shiftX,
			shiftY,
			posXClear,
			posYClear,
			x,
			y,
			math.Floor((x+g.camera.GetPositionX())/g.camera.GetTileSize())+1,
			math.Floor((y+g.camera.GetPositionY())/g.camera.GetTileSize())+1,
		),
	)
}
