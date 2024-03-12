package endless

import (
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

var B Board

const tileCount = 4096

type Board struct {
	Cells        [tileCount][tileCount]Cell
	CellOnScreen atomic.Int64
}

func NewBoard() error {
	NewTiles()
	rnd := rand.New(rand.NewSource(0))
	for i := range B.Cells {
		for j := range B.Cells[i] {
			B.Cells[i][j] = Cell{
				tileImage:      Tiles[rnd.Intn(len(Tiles))].Normal,
				tileImageSmall: Tiles[rnd.Intn(len(Tiles))].Small,
			}
		}
	}
	return nil
}

func (b *Board) Draw(screen *ebiten.Image, camera Camera) {
	tileSize := camera.GetTileSize()
	maxX, maxY := (W.GetWidth()+TileSize)/tileSize+1, (W.GetHeight()+TileSize)/tileSize+1
	op := &ebiten.DrawImageOptions{}
	shiftX, shiftY := math.Mod(camera.GetPositionX(), tileSize), math.Mod(camera.GetPositionY(), tileSize)
	if shiftX < 0 {
		shiftX = -shiftX
	}
	if shiftY < 0 {
		shiftY = -shiftY
	}
	dX := (camera.GetPositionX() + shiftX) / tileSize
	dY := (camera.GetPositionY() + shiftY) / tileSize

	cellNumber := int64(0)
	for j := float64(-1); j < maxY; j++ {
		for i := float64(-1); i < maxX; i++ {
			op.GeoM.Translate(float64(i*TileSize)-shiftX, float64(j*TileSize)-shiftY)
			//op.GeoM.Translate(W.ViewPortCenter(false))
			op.GeoM.Scale(
				camera.GetScaleFactor(),
				camera.GetScaleFactor(),
			)
			if posY := tileCount/2 + j + dY; posY < tileCount && posY >= 0 {
				if posX := tileCount/2 + i + dX; posX < tileCount && posX >= 0 {
					if camera.GetZoomFactor() > minZoom/2 {
						screen.DrawImage(b.Cells[int(posY)][int(posX)].tileImage, op)
					} else {
						//TODO оптимизация провалилась, нужно пробовать уменьшать кол-во объектов
						screen.DrawImage(b.Cells[int(posY)][int(posX)].tileImageSmall, op)
					}
					cellNumber++
				}
			}
			op.GeoM.Reset()
		}
	}
	b.CellOnScreen.Store(cellNumber)
}

func (b *Board) GetCellNumber() int64 {
	return b.CellOnScreen.Load()
}
