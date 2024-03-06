package scr

import (
	"github.com/hajimehoshi/ebiten/v2"
	"go.uber.org/zap"
)

func (s *Scr) Update() error {
	s.input.Update()
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		s.lg.Info("Left")
		s.camera.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		s.lg.Info("Right")
		s.camera.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		s.lg.Info("Up")
		s.camera.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		s.lg.Info("Down")
		s.camera.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		s.lg.Info("ZoomDown")
		s.camera.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		s.lg.Info("ZoomUp")
		s.camera.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		s.lg.Info("rotation")
		s.camera.Rotation(10)
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		s.lg.Info("w/h", zap.Int("w", s.cfg.Width()), zap.Int("h", s.cfg.Height()))
		s.camera.Reset(s.cfg.Width(), s.cfg.Height())
	}

	if dir, ok := s.input.Dir(); ok {
		switch dir {
		case DirUp:
			s.lg.Info("DirUp")
			s.camera.Down()
		case DirDown:
			s.lg.Info("DirDown")
			s.camera.Up()
		case DirLeft:
			s.lg.Info("DirLeft")
			s.camera.Right()
		case DirRight:
			s.lg.Info("DirRight")
			s.camera.Left()
		}
	}

	return nil
}
