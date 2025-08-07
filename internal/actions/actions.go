package actions

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/unng-lab/madfarmer/internal/camera"
)

type Action struct {
	Camera *camera.Camera
}

func NewAction(camera *camera.Camera) *Action {
	return &Action{Camera: camera}
}

func (a *Action) Update() {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		slog.Info("Left Mouse")
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		slog.Info("Right Mouse")
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		//slog.Info("Left")
		a.Camera.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		//slog.Info("Right")
		a.Camera.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		//slog.Info("Up")
		a.Camera.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		//slog.Info("Down")
		a.Camera.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		//slog.Info("ZoomDown")
		a.Camera.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		//slog.Info("ZoomUp")
		a.Camera.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//slog.Info("w/h", zap.Int("w", c.cfg.Width()), zap.Int("h", c.cfg.Height()))
		// TODO сделать по центру карты
		a.Camera.Reset(0, 0)
	}
}
