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
	g.camera.Prepare()
	g.board.Draw(screen)
	for i := range g.OnBoard {
		g.OnBoard[i].Draw(screen, Counter)
	}
	a, b := ebiten.CursorPosition()
	x, y := float64(a), float64(b)
	posX, posY := GetLeftAngle(g.camera.GetPositionX(), g.camera.GetPositionY(), x, y, g.camera.TileSize(), g.camera.TileSize())
	vector.DrawFilledRect(
		screen,
		float32(posX),
		float32(posY),
		float32(g.camera.TileSize()),
		float32(g.camera.TileSize()),
		color.White,
		false,
	)
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	g.ui.Draw(screen)

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
tileSize: %0.2f
TileSizeY: %0.2f
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
			g.board.GetCellNumber(),
			len(g.OnBoard),
			g.camera.TileSize(),
			g.camera.TileSize(),
			posX,
			posY,
			x,
			y,
			GetCellNumber(x, g.camera.GetPositionX(), g.camera.TileSize()),
			GetCellNumber(y, g.camera.GetPositionY(), g.camera.TileSize()),
		),
	)
}

func GetCellNumber(cursor float64, camera float64, tileSize float64) float64 {
	return math.Trunc((cursor+camera)/tileSize) + board.CountTile/2
}

func GetLeftAngle(cameraX, cameraY, cursorX, cursorY, tileSizeX, tileSizeY float64) (float64, float64) {
	var (
		x, y float64
	)

	shiftX, shiftY := math.Mod(cameraX, tileSizeX), math.Mod(cameraY, tileSizeY)
	if shiftX < 0 {
		x = -tileSizeX - shiftX
	} else if shiftX > 0 {
		x = -shiftX
	}

	if shiftY < 0 {
		y = -tileSizeY - shiftY
	} else if shiftY > 0 {
		y = -shiftY
	}

	return x + math.Trunc((cursorX-x)/tileSizeX)*tileSizeX, y + math.Trunc((cursorY-y)/tileSizeY)*tileSizeY
}
