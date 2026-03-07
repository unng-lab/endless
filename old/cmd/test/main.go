package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/assets"
)

type Game struct {
	img *ebiten.Image
}

func NewGame() *Game {
	atlas := assets.NewTileAtlas()
	img, err := atlas.TileImage(145, 1)
	if err != nil {
		log.Fatal(err)
	}

	return &Game{img: img}
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	// по центру окна
	sw, sh := screen.Size()
	iw, ih := g.img.Size()
	op.GeoM.Scale(10, 10)
	op.GeoM.Translate(float64(sw-iw)/2, float64(sh-ih)/2)

	screen.DrawImage(g.img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 360
}

func main() {
	ebiten.SetWindowSize(640, 640)
	ebiten.SetWindowTitle("Minimal Ebiten Image")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
