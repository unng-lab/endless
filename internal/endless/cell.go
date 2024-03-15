package endless

import "github.com/hajimehoshi/ebiten/v2"

type Cell struct {
	TileImage      *ebiten.Image
	TileImageSmall *ebiten.Image
	Type           int
}

func (c *Cell) MoveCost() int {
	return 0 // TODO
}
