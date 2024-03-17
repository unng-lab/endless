package board

import "github.com/hajimehoshi/ebiten/v2"

type Cell struct {
	TileImage      *ebiten.Image
	TileImageSmall *ebiten.Image
	Type           int
	Cost           float64
}

func (c Cell) MoveCost() float64 {
	return c.Cost // TODO
}
