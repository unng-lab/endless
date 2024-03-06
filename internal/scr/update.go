package scr

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func (c *Canvas) Update() error {
	c.counter++
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		c.log.Info("Left")
		c.camera.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		c.log.Info("Right")
		c.camera.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		c.log.Info("Up")
		c.camera.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		c.log.Info("Down")
		c.camera.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		c.log.Info("ZoomDown")
		c.camera.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		c.log.Info("ZoomUp")
		c.camera.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		c.log.Info("rotation")
		c.camera.Rotation(10)
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//c.log.Info("w/h", zap.Int("w", c.cfg.Width()), zap.Int("h", c.cfg.Height()))
		// TODO сделать по центру карты
		c.camera.Reset(0, 0)
	}

	//if dir, ok := c.input.Dir(); ok {
	//	switch dir {
	//	case DirUp:
	//		c.log.Info("DirUp")
	//		c.camera.Down()
	//	case DirDown:
	//		c.log.Info("DirDown")
	//		c.camera.Up()
	//	case DirLeft:
	//		c.log.Info("DirLeft")
	//		c.camera.Right()
	//	case DirRight:
	//		c.log.Info("DirRight")
	//		c.camera.Left()
	//	}
	//}
	return nil
}
