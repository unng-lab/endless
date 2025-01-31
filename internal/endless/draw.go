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
)

var Counter int

func (g *Game) Draw(screen *ebiten.Image) {
	Counter++
	g.camera.Prepare()
	g.board.Draw(screen)
	g.DrawOnBoard(screen, Counter)
	cursor := g.camera.Cursor
	posX, posY := GetLeftAngle(
		g.camera.GetPositionX(),
		g.camera.GetPositionY(),
		cursor.X,
		cursor.Y,
		g.camera.TileSize(),
		g.camera.TileSize(),
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX+g.camera.TileSize()),
		float32(posY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX+g.camera.TileSize()),
		float32(posY),
		float32(posX+g.camera.TileSize()),
		float32(posY+g.camera.TileSize()),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX),
		float32(posY+g.camera.TileSize()),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY+g.camera.TileSize()),
		float32(posX+g.camera.TileSize()),
		float32(posY+g.camera.TileSize()),
		1,
		color.White,
		false,
	)
	//vector.DrawFilledRect(
	//	screen,
	//	float32(posX),
	//	float32(posY),
	//	float32(g.camera.tileSize()),
	//	float32(g.camera.tileSize()),
	//	color.White,
	//	false,
	//)
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
			cursor.X,
			cursor.Y,
			GetCellNumber(cursor.X, g.camera.GetPositionX(), g.camera.TileSize(), float64(g.board.Width)),
			GetCellNumber(cursor.Y, g.camera.GetPositionY(), g.camera.TileSize(), float64(g.board.Height)),
		),
	)
}

func GetCellNumber(cursor float64, camera float64, tileSize float64, tileCount float64) float64 {
	return math.Trunc((cursor+camera-tileSize/2)/tileSize) + tileCount/2
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

func (g *Game) DrawOnBoard(screen *ebiten.Image, counter int) {
	x1, y1 := max(g.camera.Coordinates.Min.X, 0), max(g.camera.Coordinates.Min.Y, 0)
	x2, y2 := min(g.camera.Coordinates.Max.X, float64(g.board.Width)), min(g.camera.Coordinates.Max.Y, float64(g.board.Height))
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			for i := range g.board.Cell(int(x), int(y)).UnitList {
				g.Units[g.board.Cell(int(x), int(y)).UnitList[i].Index].Draw(screen, counter)
			}
		}
	}
}
