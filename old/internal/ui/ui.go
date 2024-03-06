package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"go.uber.org/zap"

	"github/unng-lab/madfarmer/assets/img"
)

type UI struct {
	lg *zap.Logger
}

func (u *UI) Canvas(w, h uint64) *ebiten.Image {
	bgImg, err := img.Img("terrain.jpg", w, h)
	if err != nil {
		u.lg.Fatal(err.Error())
	}
	if bgImg == nil {
		panic("img null")
	}

	return ebiten.NewImageFromImage(bgImg)
}

func New(lg *zap.Logger) *UI {
	var ui UI
	ui.lg = lg

	return *ui
}
