package board

import (
	"image/color"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/geom"

	"github/unng-lab/madfarmer/assets/img"
	"github/unng-lab/madfarmer/internal/camera"
)

const (
	hd                 = -50
	pointBufferSize    = 1000
	gridMultiplier     = 16
	updatedCellsBuffer = 64
)

const CountTile = 1024

type Board struct {
	Cells              [][]Cell
	CellOnScreen       atomic.Int64
	EmptyCell          *ebiten.Image
	ClearTile          *ebiten.Image
	Camera             *camera.Camera
	DrawOp             ebiten.DrawImageOptions
	UpdatedTick        int64
	UpdatedCellsBefore []geom.Point
	UpdatedCells       []geom.Point
	UpdatedCellsMutex  sync.Mutex
}

func NewBoard(c *camera.Camera) (*Board, error) {
	var b Board
	NewTiles()
	rnd := rand.New(rand.NewSource(0))
	b.Cells = make([][]Cell, CountTile)
	for i := range b.Cells {
		b.Cells[i] = make([]Cell, CountTile)
		for j := range b.Cells[i] {
			seed := rnd.Intn(len(Tiles))
			b.Cells[i][j] = Cell{
				TileImage:      Tiles[seed].Normal,
				TileImageSmall: Tiles[seed].Small,
				Cost:           getCost(seed),
			}
		}
	}
	empty, err := img.Img("empty.jpg", TileSize, TileSize)
	if err != nil {
		panic(err)
	}
	b.EmptyCell = empty
	b.ClearTile = ebiten.NewImage(TileSize, TileSize)
	b.ClearTile.Fill(color.Black)
	b.Camera = c
	b.UpdatedCellsBefore = make([]geom.Point, 0, 16)
	b.UpdatedCells = make([]geom.Point, 0, 16)
	return &b, nil
}

func (b *Board) Draw(screen *ebiten.Image) {
	//b.DrawOp.GeoM.Translate(-b.Camera.W.GetWidth()/2, -b.Camera.W.GetHeight()/2)
	b.DrawOp.GeoM.Scale(
		b.Camera.ScaleFactor(),
		b.Camera.ScaleFactor(),
	)
	//b.DrawOp.GeoM.Translate(b.Camera.W.GetWidth()/2, b.Camera.W.GetHeight()/2)
	b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
	cellNumber := int64(0)
	for j := b.Camera.Coordinates.Min.Y; j <= b.Camera.Coordinates.Max.Y; j++ {
		for i := b.Camera.Coordinates.Min.X; i <= b.Camera.Coordinates.Max.X; i++ {
			if i < 0 || i > CountTile-1 || j < 0 || j > CountTile-1 {
				screen.DrawImage(b.ClearTile, &b.DrawOp)
			} else {
				if b.Camera.GetZoomFactor() > hd {
					screen.DrawImage(b.Cells[int(j)][int(i)].TileImage, &b.DrawOp)
				} else {
					//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
					screen.DrawImage(b.Cells[int(j)][int(i)].TileImageSmall, &b.DrawOp)
				}
			}

			cellNumber++
			b.DrawOp.GeoM.Translate(b.Camera.TileSize(), 0)
		}
		b.DrawOp.GeoM.Reset()
		//b.DrawOp.GeoM.Translate(-b.Camera.W.GetWidth()/2, -b.Camera.W.GetHeight()/2)
		b.DrawOp.GeoM.Scale(
			b.Camera.ScaleFactor(),
			b.Camera.ScaleFactor(),
		)
		//b.DrawOp.GeoM.Translate(b.Camera.W.GetWidth()/2, b.Camera.W.GetHeight()/2)
		b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
		b.DrawOp.GeoM.Translate(0, (j+1-b.Camera.Coordinates.Min.Y)*b.Camera.TileSize())
	}
	b.CellOnScreen.Store(cellNumber)
	b.DrawOp.GeoM.Reset()
}

func (b *Board) GetCellNumber() int64 {
	return b.CellOnScreen.Load()
}

func getCost(seed int) float64 {
	if (seed >= 0 && seed < 8) || (seed >= 16 && seed < 24) || (seed >= 32 && seed < 40) || (seed >= 48 && seed < 56) {
		return 2
	}

	return 1
}

func (b *Board) GetCell(x, y int) *Cell {
	if x < 0 || x > CountTile-1 || y < 0 || y > CountTile-1 {
		return &Cell{}
	}
	return &b.Cells[y][x]
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
		if nx >= 0 && nx < CountTile-1 && ny >= 0 && ny < CountTile-1 {
			neighbors = append(neighbors, geom.Point{float64(nx), float64(ny)})
		}
	}
	return neighbors
}
