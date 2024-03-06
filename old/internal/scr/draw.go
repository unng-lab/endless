package scr

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (s *Scr) Draw(screen *ebiten.Image) {
	g := s.game.Canvas(800, 800)
	gameOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(g, gameOp)

	ui := s.ui.Canvas(800, 800)
	uiOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(ui, uiOp)

	//bgImg, err := img.Img("terrain.jpg", 1024, 1024)
	//if err != nil {
	//	s.lg.Fatal(err.Error())
	//}
	//if bgImg == nil {
	//	panic("img null")
	//}
	//tilesImage := ebiten.NewImageFromImage(*bgImg)

	// Draw each tile with each DrawImage call.
	// As the source images of all DrawImage calls are always same,
	// this rendering is done very effectively.
	// For more detail, see https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Image.DrawImage
	//for _, l := range s.layers {
	//	for i, t := range l {
	//		op := &ebiten.DrawImageOptions{}
	//		op.GeoM.Translate(float64((i%worldSizeX)*tileSize), float64((i/worldSizeX)*tileSize))
	//
	//		sx := (t % tileXCount) * tileSize
	//		sy := (t / tileXCount) * tileSize
	//		s.world.DrawImage(tilesImage.SubImage(image.Rect(sx, sy, sx+tileSize, sy+tileSize)).(*ebiten.Image), op)
	//	}
	//}

	//op := &ebiten.DrawImageOptions{}
	//s.World.DrawImage(tilesImage, op)
	////
	////for k := range s.Units {
	////	op := &ebiten.DrawImageOptions{}
	////	op.GeoM.Translate(s.UnitPoints[k].X, s.UnitPoints[k].Y)
	////	s.World.DrawImage(s.Units[k].Image(s.UnitPolygons.Factor[k]), op)
	////}
	//
	//s.camera.Render(s.World, screen)
	//vector.DrawFilledRect(screen, 50, 50, 100, 100, color.RGBA{0x80, 0x80, 0x80, 0xc0}, true)
	//worldX, worldY := s.Camera.ScreenToWorld(ebiten.CursorPosition())
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf("TPS: %0.2f\nMove (WASD/Arrows)\nZoom (QE)\nRotate (R)\nReset (Space)", ebiten.ActualTPS()),
	)

	//ebitenutil.DebugPrintAt(
	//	screen,
	//	fmt.Sprintf("%s\nCursor World Pos: %.2f,%.2f",
	//		s.camera.String(),
	//		worldX, worldY),
	//	0, s.cfg.Height()-32,
	//)
}
