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

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawCounter++
	g.camera.Prepare()
	g.board.Draw(screen)
	g.DrawOnBoard(screen, g.drawCounter)

	cursor := g.camera.Cursor
	tileSize := g.camera.TileSize()
	posX, posY := GetLeftAngle(
		g.camera.GetPositionX(),
		g.camera.GetPositionY(),
		cursor.X,
		cursor.Y,
		tileSize,
		tileSize,
	)
	drawTileOutline(screen, posX, posY, tileSize, color.White)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	lastGC := time.Unix(0, int64(m.LastGC))
	memoryMB := m.Alloc / 1024 / 1024

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
			memoryMB,
			time.Since(lastGC).Seconds(),
			g.camera.GetPositionX(),
			g.camera.GetPositionY(),
			g.camera.GetZoomFactor(),
			g.board.GetCellNumber(),
			g.OnBoardCounter,
			tileSize,
			tileSize,
			posX,
			posY,
			cursor.X,
			cursor.Y,
			GetCellNumber(cursor.X, g.camera.GetPositionX(), tileSize, float64(g.board.Width)),
			GetCellNumber(cursor.Y, g.camera.GetPositionY(), tileSize, float64(g.board.Height)),
		),
	)
}

func GetCellNumber(cursor float64, camera float64, tileSize float64, tileCount float64) float64 {
	tileHalf := tileSize / 2
	return math.Trunc((cursor+camera-tileHalf)/tileSize) + tileCount/2
}

func GetLeftAngle(cameraX, cameraY, cursorX, cursorY, tileSizeX, tileSizeY float64) (float64, float64) {
	offsetX := math.Mod(cameraX, tileSizeX)
	if offsetX < 0 {
		offsetX += tileSizeX
	}

	offsetY := math.Mod(cameraY, tileSizeY)
	if offsetY < 0 {
		offsetY += tileSizeY
	}

	originX := -offsetX
	originY := -offsetY

	snappedX := originX + math.Trunc((cursorX-originX)/tileSizeX)*tileSizeX
	snappedY := originY + math.Trunc((cursorY-originY)/tileSizeY)*tileSizeY

	return snappedX, snappedY
}

func (g *Game) DrawOnBoard(screen *ebiten.Image, counter int) {
	g.OnBoardCounter = 0

	startX := int(math.Max(g.camera.Coordinates.Min.X, 0))
	startY := int(math.Max(g.camera.Coordinates.Min.Y, 0))
	endX := int(math.Min(g.camera.Coordinates.Max.X, float64(g.board.Width-1)))
	endY := int(math.Min(g.camera.Coordinates.Max.Y, float64(g.board.Height-1)))

	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			cell := g.board.Cell(x, y)
			for _, unitRef := range cell.UnitList {
				g.Units[unitRef.Index].Draw(screen, counter)
				g.OnBoardCounter++
			}
		}
	}
}

func drawTileOutline(screen *ebiten.Image, x, y, tileSize float64, clr color.Color) {
	lines := [][4]float32{
		{float32(x), float32(y), float32(x + tileSize), float32(y)},
		{float32(x + tileSize), float32(y), float32(x + tileSize), float32(y + tileSize)},
		{float32(x), float32(y), float32(x), float32(y + tileSize)},
		{float32(x), float32(y + tileSize), float32(x + tileSize), float32(y + tileSize)},
	}

	for _, line := range lines {
		vector.StrokeLine(screen, line[0], line[1], line[2], line[3], 1, clr, false)
	}
}
