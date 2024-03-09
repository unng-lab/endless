package endless

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
)

var UI UIEngine

type UIEngine struct {
}

func (ui *UIEngine) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		slog.Info("Left")
		C.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		slog.Info("Right")
		C.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		slog.Info("Up")
		C.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		slog.Info("Down")
		C.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		slog.Info("ZoomDown")
		C.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		slog.Info("ZoomUp")
		C.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//slog.Info("w/h", zap.Int("w", C.cfg.Width()), zap.Int("h", C.cfg.Height()))
		// TODO сделать по центру карты
		C.Reset(0, 0)
	}

	return nil
}
