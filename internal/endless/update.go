package endless

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		slog.Info("Left")
		g.camera.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		slog.Info("Right")
		g.camera.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		slog.Info("Up")
		g.camera.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		slog.Info("Down")
		g.camera.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		slog.Info("ZoomDown")
		g.camera.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		slog.Info("ZoomUp")
		g.camera.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//slog.Info("w/h", zap.Int("w", g.camera.cfg.Width()), zap.Int("h", g.camera.cfg.Height()))
		// TODO сделать по центру карты
		g.camera.Reset(0, 0)
	}

	for i := range g.Units {
		g.Units[i].Update()
	}

	return UI.Update()
}
