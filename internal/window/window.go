package window

import (
	"log/slog"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

type Config interface {
	GetScreenWidth() uint64
	GetScreenHeight() uint64
	GetWindowResizeMode() ebiten.WindowResizingModeType
}

type Default struct {
	log    *slog.Logger
	cfg    Config
	Width  atomic.Uint64
	Height atomic.Uint64
}

func New(log *slog.Logger, cfg Config) *Default {
	var d Default
	d.log = log.With("scr", "Canvas")
	d.cfg = cfg

	ebiten.SetWindowSize(800, 800)
	//ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MadFarmer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return &d
}

func Conv(perc float32, length uint64) float32 {
	return perc * float32(length) / 100
}

func (d *Default) Pos(perX, perY float32) (int, int) {
	return d.PosX(perX), d.PosY(perY)
}
func (d *Default) PosX(perX float32) int {
	return int(float32(d.Width.Load()) * perX / 100)
}
func (d *Default) PosY(perY float32) int {
	return int(float32(d.Height.Load()) * perY / 100)
}
