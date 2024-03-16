package endless

import (
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
)

var B Board

const CountTile = 256

type Board struct {
	Cells        [CountTile][CountTile]Cell
	CellOnScreen atomic.Int64
	EmptyCell    *ebiten.Image
	DrawOp       ebiten.DrawImageOptions
}

func NewBoard() error {
	NewTiles()
	rnd := rand.New(rand.NewSource(0))
	for i := range B.Cells {
		for j := range B.Cells[i] {
			B.Cells[i][j] = Cell{
				TileImage:      Tiles[rnd.Intn(len(Tiles))].Normal,
				TileImageSmall: Tiles[rnd.Intn(len(Tiles))].Small,
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

func (b *Board) Draw(screen *ebiten.Image, camera Camera) (float64, float64, float64, float64) {
	tileSize := camera.GetTileSize()
	maxX, maxY := (W.GetWidth())/tileSize+1, (W.GetHeight())/tileSize+1
	defer b.DrawOp.GeoM.Reset()
	b.DrawOp.GeoM.Scale(
		camera.GetScaleFactor(),
		camera.GetScaleFactor(),
	)
	leftX, leftY, cellX, cellY := GetLeftXY(camera.GetPositionX(), camera.GetPositionY(), tileSize)
	b.DrawOp.GeoM.Translate(leftX, leftY)
	cellNumber := int64(0)
	for j := float64(0); j < maxY; j++ {
		for i := float64(0); i < maxX; i++ {
			if posY := cellY + j; posY < CountTile && posY >= 0 {
				if posX := cellX + i; posX < CountTile && posX >= 0 {
					if int(posY) == 2050 && int(posX) == 2050 {
						screen.DrawImage(b.EmptyCell, &b.DrawOp)
					} else if int(posY) == 2052 && int(posX) == 2052 {
						screen.DrawImage(b.EmptyCell, &b.DrawOp)
					} else if int(posY) == 2054 && int(posX) == 2054 {
						screen.DrawImage(b.EmptyCell, &b.DrawOp)
					} else if int(posY) == 2054 && int(posX) == 2054 {
						screen.DrawImage(b.EmptyCell, &b.DrawOp)
					} else {
						if camera.GetZoomFactor() > minZoom/2 {
							screen.DrawImage(b.Cells[int(posY)][int(posX)].TileImage, &b.DrawOp)
						} else {
							//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
							screen.DrawImage(b.Cells[int(posY)][int(posX)].TileImageSmall, &b.DrawOp)
						}
					}

					cellNumber++
				}
			}
			b.DrawOp.GeoM.Translate(tileSize, 0)
		}
		b.DrawOp.GeoM.Reset()
		b.DrawOp.GeoM.Scale(
			camera.GetScaleFactor(),
			camera.GetScaleFactor(),
		)
		b.DrawOp.GeoM.Translate(leftX, leftY)
		b.DrawOp.GeoM.Translate(0, (j+1)*tileSize)
	}
	b.CellOnScreen.Store(cellNumber)

	return cellX, cellY, cellX + maxX, cellY + maxY

}

func (b *Board) GetCellNumber() int64 {
	return b.CellOnScreen.Load()
}

func GetLeftXY(cameraX float64, cameraY float64, tileSize float64) (float64, float64, float64, float64) {
	var (
		x, y         float64
		cellX, cellY float64 = CountTile / 2, CountTile / 2
	)

	shiftX, shiftY := math.Mod(cameraX, tileSize), math.Mod(cameraY, tileSize)
	if shiftX < 0 {
		x = -tileSize - shiftX
		cellX += -1
	} else if shiftX > 0 {
		x = -shiftX
	}
	cellX += math.Trunc(cameraX / tileSize)

	if shiftY < 0 {
		y = -tileSize - shiftY
		cellY += -1
	} else if shiftY > 0 {
		y = -shiftY
	}
	cellY += math.Trunc(cameraY / tileSize)

	return math.Round(x*100) / 100, math.Round(y*100) / 100, cellX, cellY
}
