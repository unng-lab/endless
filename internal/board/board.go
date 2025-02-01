package board

import (
	"image/color"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/geom"

	"github.com/unng-lab/madfarmer/assets/img"
	"github.com/unng-lab/madfarmer/internal/camera"
)

const (
	hd                 = -50
	pointBufferSize    = 1000
	gridMultiplier     = 16
	updatedCellsBuffer = 64
)

type Board struct {
	Cells                   []Cell
	Width, Height           uint64
	TileSize, SmallTileSize uint64
	CellOnScreen            atomic.Int64
	EmptyCell               *ebiten.Image
	ClearTile               *ebiten.Image
	Camera                  *camera.Camera
	UpdatedTick             int64
	UpdatedCellsBefore      []geom.Point
	UpdatedCells            []geom.Point
	UpdatedCellsMutex       sync.Mutex
}

func NewBoard(c *camera.Camera, tileSize, smallTileSize uint64, tileCount uint64) (*Board, error) {
	var b Board
	b.Width, b.Height = tileCount, tileCount
	b.TileSize, b.SmallTileSize = tileSize, smallTileSize

	NewTiles(b.TileSize, b.SmallTileSize)
	seed := func() int {
		return rand.IntN(5) + 1
	}
	b.Cells = make([]Cell, b.Width*b.Height)
	for i := range b.Cells {
		b.Cells[i] = NewCell(CellType(seed()), int(b.TileSize))
	}
	empty, err := img.Img("empty.jpg", tileSize, tileSize)
	if err != nil {
		panic(err)
	}
	b.EmptyCell = empty
	b.ClearTile = ebiten.NewImage(int(tileSize), int(tileSize))
	b.ClearTile.Fill(color.Black)
	b.Camera = c
	b.UpdatedCellsBefore = make([]geom.Point, 0, 16)
	b.UpdatedCells = make([]geom.Point, 0, 16)
	return &b, nil
}

func (b *Board) Cell(x, y int) *Cell {
	return &b.Cells[b.index(x, y)]
}

func (b *Board) index(x, y int) int {
	return y*int(b.Width) + x
}

//func (b *Board) Draw(screen *ebiten.Image) {
//	b.DrawOp.GeoM.Scale(
//		b.Camera.ScaleFactor(),
//		b.Camera.ScaleFactor(),
//	)
//	b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
//
//	cellNumber := int64(0)
//	maxX, maxY := float64(b.Width-1), float64(b.Height-1)
//
//	for j := b.Camera.Coordinates.Min.Y; j <= b.Camera.Coordinates.Max.Y; j++ {
//		for i := b.Camera.Coordinates.Min.X; i <= b.Camera.Coordinates.Max.X; i++ {
//			if i < 0 || i > maxX || j < 0 || j > maxY {
//				screen.DrawImage(b.ClearTile, &b.DrawOp)
//			} else {
//				if b.Camera.GetZoomFactor() > hd {
//					screen.DrawImage(b.Cell(int(i), int(j)).TileImage, &b.DrawOp)
//				} else {
//					//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
//					screen.DrawImage(b.Cell(int(i), int(j)).TileImageSmall, &b.DrawOp)
//				}
//			}
//
//			cellNumber++
//			b.DrawOp.GeoM.Translate(b.Camera.TileSize(), 0)
//		}
//		b.DrawOp.GeoM.Reset()
//		b.DrawOp.GeoM.Scale(
//			b.Camera.ScaleFactor(),
//			b.Camera.ScaleFactor(),
//		)
//		b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
//		b.DrawOp.GeoM.Translate(0, (j+1-b.Camera.Coordinates.Min.Y)*b.Camera.TileSize())
//	}
//	b.CellOnScreen.Store(cellNumber)
//	b.DrawOp.GeoM.Reset()
//}

func (b *Board) Draw(screen *ebiten.Image) {
	cam := b.Camera
	scale := cam.ScaleFactor()
	tileSize := cam.TileSize()
	tileSizeF := float64(tileSize)
	useHD := cam.GetZoomFactor() > hd

	// Рассчитываем границы доски
	maxXBoard := float64(b.Width - 1)
	maxYBoard := float64(b.Height - 1)

	// Получаем видимую область камеры
	visible := cam.Coordinates
	minX, maxX := visible.Min.X, visible.Max.X
	minY, maxY := visible.Min.Y, visible.Max.Y

	// Рассчитываем количество ячеек на экране
	cellCount := (maxX - minX + 1) * (maxY - minY + 1)
	b.CellOnScreen.Store(int64(cellCount))

	// Базовые трансформации
	baseGeoM := ebiten.GeoM{}
	baseGeoM.Scale(scale, scale)
	baseGeoM.Translate(
		cam.RelativePixels.Min.X,
		cam.RelativePixels.Min.Y,
	)

	// Оптимизация: кешируем параметры отрисовки
	drawOpts := &ebiten.DrawImageOptions{}

	// Основной цикл отрисовки
	for y, j := 0.0, minY; j <= maxY; j++ {
		rowGeoM := baseGeoM
		rowGeoM.Translate(0, y)
		xOffset := 0.0

		for i := minX; i <= maxX; i++ {
			// Выбираем изображение для отрисовки
			var img *ebiten.Image
			switch {
			case i < 0 || j < 0 || i > maxXBoard || j > maxYBoard:
				img = b.ClearTile
			default:
				cell := b.Cell(int(i), int(j))
				if useHD {
					img = cell.TileImage
				} else {
					img = cell.TileImageSmall
				}
			}

			// Устанавливаем геометрическую трансформацию
			drawOpts.GeoM = rowGeoM
			drawOpts.GeoM.Translate(xOffset, 0)
			screen.DrawImage(img, drawOpts)

			xOffset += tileSizeF
		}
		y += tileSizeF
	}
}

func (b *Board) GetCellNumber() int64 {
	return b.CellOnScreen.Load()
}

func (b *Board) GetCell(x, y int64) *Cell {
	if uint64(x) >= b.Width || uint64(y) >= b.Height {
		return nil
	}
	return b.Cell(int(x), int(y))
}

func (b *Board) AddUpdatedCells(from, to geom.Point) {
	b.UpdatedCellsMutex.Lock()
	defer b.UpdatedCellsMutex.Unlock()
	b.UpdatedCells = append(b.UpdatedCells, from, to)
}

func (b *Board) ClearUpdatedCells() {
	b.UpdatedCellsMutex.Lock()
	defer b.UpdatedCellsMutex.Unlock()
	b.UpdatedCellsBefore, b.UpdatedCells = b.UpdatedCells, b.UpdatedCellsBefore[:0]
	if len(b.UpdatedCells) > updatedCellsBuffer {
		slog.Warn("Board.ClearUpdatedCells", "len", len(b.UpdatedCells))
		b.UpdatedCells = make([]geom.Point, 0, updatedCellsBuffer/4)
	}
}

var directions = [8][]int{
	{0, -1},
	{0, 1},
	{-1, 0},
	{1, 0},
	{-1, -1},
	{-1, 1},
	{1, -1},
	{1, 1},
}

func (b *Board) GetNeighbours(target geom.Point) []geom.Point {
	var neighbors []geom.Point
	for _, dir := range directions {
		nx, ny := int(target.X)+dir[0], int(target.Y)+dir[1]
		// Проверяем, что новые координаты внутри границ карты
		if nx >= 0 && nx < int(b.Width)-1 && ny >= 0 && ny < int(b.Height)-1 {
			neighbors = append(neighbors, geom.Point{float64(nx), float64(ny)})
		}
	}
	return neighbors
}

func (b *Board) GetCost(from, to geom.Point, tick int64) float64 {
	cellA := b.GetCell(int64(from.X), int64(from.Y))
	cellB := b.GetCell(int64(to.X), int64(to.Y))
	if cellA == nil || cellB == nil {
		panic("Cell is nil")
	}
	if math.IsInf(cellA.Cost, 1) || math.IsInf(cellB.Cost, 1) {
		return math.Inf(1)
	}

	avgCost := (cellA.Cost + cellB.Cost) / 2
	length := from.Length(to)
	return avgCost * length
}

func (b *Board) IsInside(p geom.Point) bool {
	if p.X < 0 || p.X > float64(b.Width)-1 || p.Y < 0 || p.Y > float64(b.Height)-1 {
		return false
	}
	return true
}

func (b *Board) IsObstacle(p geom.Point) bool {
	cell := b.GetCell(int64(p.X), int64(p.Y))
	return math.IsInf(cell.Cost, 1)
}

func (b *Board) GetRandomPoint() geom.Point {
	p := geom.Point{
		X: float64(rand.Uint64N(b.Width)),
		Y: float64(rand.Uint64N(b.Height)),
	}
	cell := b.Cell(p.GetInts())
	if math.IsInf(cell.Cost, 1) {
		return b.GetRandomPoint()
	}
	return p
}
