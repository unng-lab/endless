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

const CountTile = 4096

type Board struct {
	Cells        [][]Cell
	CellOnScreen atomic.Int64
	EmptyCell    *ebiten.Image
	Camera       *camera.Camera
	world        *ebiten.Image
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
	empty, err := img.Img("empty.jpg", 16, 16)
	if err != nil {
		panic(err)
	}
	b.EmptyCell = empty
	b.Camera = c
	// TODO: change to real width and height
	b.world = ebiten.NewImage(camera.DefaultScreenWidth, camera.DefaultScreenHeight)

	return &b, nil
}

func (b *Board) Render(screen *ebiten.Image) {
	screen.DrawImage(b.world, &ebiten.DrawImageOptions{
		GeoM: b.Camera.WorldMatrix(),
	})
}

func (b *Board) Draw(screen *ebiten.Image) {
	//b.DrawOp.GeoM.Scale(
	//	b.Camera.GetScaleFactorX(),
	//	b.Camera.GetScaleFactorX(),
	//)
	//b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
	cellNumber := int64(0)
	for j := b.Camera.Coordinates.Min.Y; j < b.Camera.Coordinates.Max.Y; j++ {
		for i := b.Camera.Coordinates.Min.X; i < b.Camera.Coordinates.Max.X; i++ {
			if i < 0 || i > CountTile-1 || j < 0 || j > CountTile-1 {
				b.DrawOp.GeoM.Translate(b.Camera.TileSizeX, 0)
				continue
			}
			if int(j) == 2050 && int(i) == 2050 {
				b.world.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2052 && int(i) == 2052 {
				b.world.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				b.world.DrawImage(b.EmptyCell, &b.DrawOp)
			} else if int(j) == 2054 && int(i) == 2054 {
				b.world.DrawImage(b.EmptyCell, &b.DrawOp)
			} else {
				if b.Camera.GetZoomFactor() > hd {
					b.world.DrawImage(b.Cells[int(j)][int(i)].TileImage, &b.DrawOp)
				} else {
					//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
					b.world.DrawImage(b.Cells[int(j)][int(i)].TileImageSmall, &b.DrawOp)
				}
			}

			cellNumber++
			b.DrawOp.GeoM.Translate(b.Camera.TileSizeY, 0)
		}
		b.DrawOp.GeoM.Reset()
		//b.DrawOp.GeoM.Scale(
		//	b.Camera.GetScaleFactorX(),
		//	b.Camera.GetScaleFactorX(),
		//)
		//b.DrawOp.GeoM.Translate(b.Camera.RelativePixels.Min.X, b.Camera.RelativePixels.Min.Y)
		b.DrawOp.GeoM.Translate(0, (j+1-b.Camera.Coordinates.Min.Y)*b.Camera.TileSizeY)
	}
	b.CellOnScreen.Store(cellNumber)
	b.DrawOp.GeoM.Reset()
	b.Render(screen)
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

func (b *Board) Layout(outsideWidth, outsideHeight int) {
	if b.world.Bounds().Max.X != outsideWidth || b.world.Bounds().Max.Y != outsideHeight {
		b.world = ebiten.NewImage(outsideWidth, outsideHeight)
	}
}
