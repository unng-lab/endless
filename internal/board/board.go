package board

import (
	"math/rand"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
	"github/unng-lab/madfarmer/internal/camera"
)

const (
	hd = -50
)

var B Board

const CountTile = 4096

type Board struct {
	Cells        [][]Cell
	CellOnScreen atomic.Int64
	EmptyCell    *ebiten.Image
	DrawOp       ebiten.DrawImageOptions
}

func NewBoard() error {
	NewTiles()
	rnd := rand.New(rand.NewSource(0))
	B.Cells = make([][]Cell, CountTile)
	for i := range B.Cells {
		B.Cells[i] = make([]Cell, CountTile)
		for j := range B.Cells[i] {
			seed := rnd.Intn(len(Tiles))
			B.Cells[i][j] = Cell{
				TileImage:      Tiles[seed].Normal,
				TileImageSmall: Tiles[seed].Small,
				Cost:           getCost(seed),
			}
		}
	}
	empty, err := img.Img("empty.jpg", 16, 16)
	if err != nil {
		panic(err)
	}
	B.EmptyCell = empty

	return nil
}

func (b *Board) Draw(screen *ebiten.Image, camera camera.Camera) {
	defer b.DrawOp.GeoM.Reset()
	b.DrawOp.GeoM.Scale(
		camera.GetScaleFactor(),
		camera.GetScaleFactor(),
	)
	b.DrawOp.GeoM.Translate(camera.DrawArea.Min.X, camera.DrawArea.Min.Y)
	cellNumber := int64(0)
	for j := camera.Coordinates.Min.Y; j < camera.Coordinates.Max.Y; j++ {
		for i := camera.Coordinates.Min.X; i < camera.Coordinates.Max.X; i++ {
			if i < 0 || i > CountTile-1 || j < 0 || j > CountTile-1 {
				b.DrawOp.GeoM.Translate(camera.TileSize, 0)
				continue
			}
			if int(j) == 2050 && int(i) == 2050 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2052 && int(i) == 2052 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				screen.DrawImage(b.EmptyCell, &b.DrawOp)
			} else {
				if camera.GetZoomFactor() > hd {
					screen.DrawImage(b.Cells[int(j)][int(i)].TileImage, &b.DrawOp)
				} else {
					//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
					screen.DrawImage(b.Cells[int(j)][int(i)].TileImageSmall, &b.DrawOp)
				}
			}

			cellNumber++
			b.DrawOp.GeoM.Translate(camera.TileSize, 0)
		}
		b.DrawOp.GeoM.Reset()
		b.DrawOp.GeoM.Scale(
			camera.GetScaleFactor(),
			camera.GetScaleFactor(),
		)
		b.DrawOp.GeoM.Translate(camera.DrawArea.Min.X, camera.DrawArea.Min.Y)
		b.DrawOp.GeoM.Translate(0, (j+1-camera.Coordinates.Min.Y)*camera.TileSize)
	}
	b.CellOnScreen.Store(cellNumber)

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
