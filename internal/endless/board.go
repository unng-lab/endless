package endless

import (
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
	for i := range B.Cells {
		for j := range B.Cells[i] {
			B.Cells[i][j] = Cell{
				tileImage:      Tiles[rand.Intn(len(Tiles))].Normal,
				tileImageSmall: Tiles[rand.Intn(len(Tiles))].Small,
			}
		}
	}
	return nil
}

func (b *Board) Draw(screen *ebiten.Image) {
	tileSize := C.GetTileSize()
	maxX, maxY := float64(W.GetWidth())/tileSize+1, float64(W.GetHeight())/tileSize+1
	op := &ebiten.DrawImageOptions{}
	dX := C.GetPositionX() / tileSize
	dY := C.GetPositionY() / tileSize
	cellNumber := int64(0)
	for j := float64(-1); j < maxY; j++ {
		for i := float64(-1); i < maxX; i++ {
			op.GeoM.Translate(i*TileSize, j*TileSize)
			//op.GeoM.Translate(W.ViewPortCenter(false))
			op.GeoM.Scale(
				C.GetScaleFactor(),
				C.GetScaleFactor(),
			)
			if posY := tileCount/2 + j + dY; posY < tileCount && posY >= 0 {
				if posX := tileCount/2 + i + dX; posX < tileCount && posX >= 0 {
					if C.GetZoomFactor() > minZoom/2 {
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
