package board

import (
	"image/color"
	"math/rand"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
	"github/unng-lab/madfarmer/internal/camera"
)

const (
	hd = -50
)

const CountTile = 1024

type Board struct {
	Cells        [][]Cell
	CellOnScreen atomic.Int64
	EmptyCell    *ebiten.Image
	ClearTile    *ebiten.Image
	Camera       *camera.Camera
	DrawOp       ebiten.DrawImageOptions
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
			} else if int(j) == 2050 && int(i) == 2050 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2052 && int(i) == 2052 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
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

func (b *Board) GetCell(x, y int) Cell {
	if x < 0 || x > CountTile-1 || y < 0 || y > CountTile-1 {
		return Cell{}
	}
	return b.Cells[y][x]
}
