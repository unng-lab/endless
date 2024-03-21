package endless

import (
	"fmt"
	"image/color"
	"math"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/internal/board"
)

var Counter int

func (g *Game) Draw(screen *ebiten.Image) {
	Counter++
	camera := g.camera
	camera.Prepare()
	board.B.Draw(screen, camera)
	unitNumber := 0
	for i := range g.Units {
		if g.Units[i].unit.Draw(screen, Counter, camera) {
			unitNumber++
		}
	}
	a, b := ebiten.CursorPosition()
	x, y := float64(a), float64(b)
	posX, posY := GetLeftAngle(camera.GetPositionX(), camera.GetPositionY(), x, y, camera.GetTileSize())
	vector.DrawFilledRect(
		screen,
		float32(posX),
		float32(posY),
		float32(camera.GetTileSize()),
		float32(camera.GetTileSize()),
		color.White,
		false,
	)
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(`TPS: %0.2f
FPS: %0.2f
Goroutines: %d
Memory in mb: %d
Last gc was: %0.2f
CameraX: %0.2f
CameraY: %0.2f
Zoom: %0.2f
CellNumber: %d
UnitNumber: %d
TileSize: %0.2f
posX: %0.2f
posY: %0.2f
CursorX: %0.2f
CursorY: %0.2f
CellX: %0.2f
CellY: %0.2f`,
			ebiten.ActualTPS(),
			ebiten.ActualFPS(),
			runtime.NumGoroutine(),
			m.Alloc/1024/1024,
			time.Now().Sub(time.Unix(0, int64(m.LastGC))).Seconds(),
			g.camera.GetPositionX(),
			g.camera.GetPositionY(),
			g.camera.GetZoomFactor(),
			board.B.GetCellNumber(),
			unitNumber,
			camera.GetTileSize(),
			posX,
			posY,
			x,
			y,
			GetCellNumber(x, camera.GetPositionX(), camera.GetTileSize()),
			GetCellNumber(y, camera.GetPositionY(), camera.GetTileSize()),
		),
	)
}

func GetCellNumber(cursor float64, camera float64, tileSize float64) float64 {
	return math.Trunc((cursor+camera)/tileSize) + board.CountTile/2
}

func GetLeftAngle(cameraX, cameraY, cursorX, cursorY, tileSize float64) (float64, float64) {
	var (
		x, y float64
	)

	shiftX, shiftY := math.Mod(cameraX, tileSize), math.Mod(cameraY, tileSize)
	if shiftX < 0 {
		x = -tileSize - shiftX
	} else if shiftX > 0 {
		x = -shiftX
	}

	if shiftY < 0 {
		y = -tileSize - shiftY
	} else if shiftY > 0 {
		y = -shiftY
	}

	return x + math.Trunc((cursorX-x)/tileSize)*tileSize, y + math.Trunc((cursorY-y)/tileSize)*tileSize
}
