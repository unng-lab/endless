package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/camera"
)

type UIEngine struct {
	camera *camera.Camera
}

func New(camera *camera.Camera) *UIEngine {
	return &UIEngine{
		camera: camera,
	}
}

func (ui *UIEngine) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		//slog.Info("Left")
		ui.camera.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		//slog.Info("Right")
		ui.camera.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		//slog.Info("Up")
		ui.camera.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		//slog.Info("Down")
		ui.camera.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		//slog.Info("ZoomDown")
		ui.camera.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		//slog.Info("ZoomUp")
		ui.camera.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//slog.Info("w/h", zap.Int("w", ui.camera.cfg.Width()), zap.Int("h", ui.camera.cfg.Height()))
		// TODO сделать по центру карты
		ui.camera.Reset(0, 0)
	}
	return nil
}
