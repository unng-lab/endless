package scr

import (
	"github.com/hajimehoshi/ebiten/v2"
	"go.uber.org/zap"
)

var (
	_ ebiten.Game = (*Scr)(nil)
)

type Scr struct {
	lg   *zap.Logger
	game Game
	ui   UI
	cfg  Cfg
}

// Dir represents a direction.
type Dir int

const (
	DirNone Dir = iota
	DirUp
	DirRight
	DirDown
	DirLeft
)

type Game interface {
	Canvas(w, h int) *ebiten.Image
}

type UI interface {
	Canvas(w, h int) *ebiten.Image
}

//type Camera interface {
//	Left()
//	Right()
//	Up()
//	Down()
//	ZoomDown()
//	ZoomUp()
//	Rotation(int)
//	Reset(w, h int)
//	String() string
//}
//
//type Input interface {
//	Update()
//	Dir() (Dir, bool)
//}

type Cfg interface {
	Width() int
	Height() int
}

func New(
	lg *zap.Logger,
	game Game,
	ui UI,
) *Scr {
	return &Scr{
		game: game,
		ui:   ui,
		lg:   lg,
	}
}
