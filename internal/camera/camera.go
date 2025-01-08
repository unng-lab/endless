package camera

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/window"
)

const (
	MaxZoom = 200
	MinZoom = -100
)

const (
	DefaultScreenWidth  = 1920
	DefaultScreenHeight = 1080
)

type cfg struct {
	TileSize  float64
	TileCount float64
}

// Camera TODO add maxX and maxY camera
type Camera struct {
	cfg        cfg
	positionX  float64
	positionY  float64
	zoomFactor float64
	// TODO вынести в отдельную структуру например screen
	tileSize float64
	// Координаты камеры
	Coordinates    geom.Rectangle
	AbsolutePixels geom.Rectangle
	RelativePixels geom.Rectangle
	Cursor         geom.Point
	scaleFactor    float64
	W              window.Window
	geom           ebiten.GeoM
}

func (c *Camera) TileSize() float64 {
	return c.tileSize
}

func (c *Camera) Reset(w, h int) {
	c.positionX = 0
	c.positionY = 0
	c.zoomFactor = 0
}

func (c *Camera) Up() {
	c.positionY += -50
}

func (c *Camera) Down() {
	c.positionY += 50
}

func (c *Camera) Left() {
	c.positionX += -50
}

func (c *Camera) Right() {
	c.positionX += 50
}

func (c *Camera) ZoomUp() {
	if c.zoomFactor < MaxZoom {
		c.zoomFactor += 10
	}
}

func (c *Camera) ZoomDown() {
	if c.zoomFactor > MinZoom {
		c.zoomFactor += -10
	}
}
func (c *Camera) ScaleFactor() float64 {
	return c.scaleFactor
}

func (c *Camera) scale() float64 {
	return math.Pow(1.01, c.zoomFactor)
}

func (c *Camera) getTileSize() float64 {
	return c.cfg.TileSize * c.scaleFactor
}

func (c *Camera) GetZoomFactor() float64 {
	return c.zoomFactor
}

func (c *Camera) GetPositionX() float64 {
	return c.positionX
}

func (c *Camera) GetPositionY() float64 {
	return c.positionY
}

func (c *Camera) GetCurrentCoordinates() geom.Rectangle {
	return c.Coordinates
}

func (c *Camera) Prepare() {
	c.scaleFactor = c.scale()
	c.tileSize = c.getTileSize()
	maxX, maxY := (c.W.GetWidth())/c.tileSize+1, (c.W.GetHeight())/c.tileSize+1

	var (
		x, y         float64
		cellX, cellY float64 = c.cfg.TileCount / 2, c.cfg.TileCount / 2
	)

	shiftX, shiftY := math.Mod(c.positionX, c.tileSize), math.Mod(c.positionY, c.tileSize)
	if shiftX < 0 {
		x = -c.tileSize - shiftX
		cellX += -1
	} else if shiftX > 0 {
		x = -shiftX
	}
	cellX += math.Trunc(c.positionX / c.tileSize)

	if shiftY < 0 {
		y = -c.tileSize - shiftY
		cellY += -1
	} else if shiftY > 0 {
		y = -shiftY
	}
	cellY += math.Trunc(c.positionY / c.tileSize)

	c.Coordinates = geom.Rectangle{
		Min: geom.Point{
			X: cellX,
			Y: cellY,
		},
		Max: geom.Point{
			X: cellX + maxX,
			Y: cellY + maxY,
		},
	}

	c.AbsolutePixels = geom.Rectangle{
		Min: geom.Point{
			X: math.Trunc(c.tileSize*cellX*100)/100 - x,
			Y: math.Trunc(c.tileSize*cellY*100)/100 - y,
		},
		Max: geom.Point{
			X: math.Trunc(c.tileSize*(cellX+maxX)*100)/100 - x,
			Y: math.Trunc(c.tileSize*(cellY+maxY)*100)/100 - y,
		},
	}
	c.RelativePixels = geom.Rectangle{
		Min: geom.Point{
			X: x,
			Y: y,
		},
		Max: geom.Point{
			X: x + c.tileSize*maxX,
			Y: y + c.tileSize*maxY,
		},
	}

	a, b := ebiten.CursorPosition()
	c.Cursor = geom.Point{float64(a), float64(b)}
}

func (c *Camera) GetCurrentPixels() geom.Rectangle {
	return c.AbsolutePixels
}

func (c *Camera) MiddleOfPointInRelativePixels(point geom.Point) geom.Point {
	distX, distY := c.Coordinates.Min.Distance(point)
	return geom.Point{
		X: c.RelativePixels.Min.X + distX*c.tileSize + c.tileSize/2,
		Y: c.RelativePixels.Min.Y + distY*c.tileSize + c.tileSize/2,
	}
}

//func AbsoluteToRelative(point geom.Point) geom.Point {
//	return geom.Point{
//		X: math.Trunc(point.X / c.cfg.tileSize),
//		Y: math.Trunc(point.Y / c.cfg.tileSize),
//	}
//}
//
//func RelativeToAbsolute(point geom.Point) geom.Point {
//	return geom.Point{
//		X: point.X * DefaultTileSize,
//		Y: point.Y * DefaultTileSize,
//	}
//}
//
//func MiddleOfRelativePointInAbsolutePixels(point geom.Point) geom.Point {
//	return geom.Point{
//		X: point.X*DefaultTileSize + DefaultTileSize/2,
//		Y: point.Y*DefaultTileSize + DefaultTileSize/2,
//	}
//}

func (c *Camera) PointToCameraPixel(point geom.Point) geom.Point {
	return geom.Point{
		X: point.X*c.tileSize - c.AbsolutePixels.Min.X,
		Y: point.Y*c.tileSize - c.AbsolutePixels.Min.Y,
	}
}

//func (c *Camera) DrawPixelMinPoint() (float64, float64) {
//	return c.AbsolutePixels.Min.X - c.tileSize*CountTile/2,
//		c.AbsolutePixels.Min.Y - c.tileSize*CountTile/2
//}

func New(tileSize float64, tileCount float64) *Camera {
	return &Camera{
		cfg: cfg{
			TileSize:  tileSize,
			TileCount: tileCount,
		},
		positionX:  0,
		positionY:  0,
		zoomFactor: 0,
		tileSize:   0,
		Coordinates: geom.Rectangle{
			Min: geom.Point{
				X: 0,
				Y: 0,
			},
			Max: geom.Point{
				X: 0,
				Y: 0,
			},
		},
		AbsolutePixels: geom.Rectangle{
			Min: geom.Point{
				X: 0,
				Y: 0,
			},
			Max: geom.Point{
				X: 0,
				Y: 0,
			},
		},
		RelativePixels: geom.Rectangle{
			Min: geom.Point{
				X: 0,
				Y: 0,
			},
			Max: geom.Point{
				X: 0,
				Y: 0,
			},
		},
		scaleFactor: 0,
	}
}

func (c *Camera) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		//slog.Info("Left")
		c.Left()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		//slog.Info("Right")
		c.Right()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		//slog.Info("Up")
		c.Up()
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		//slog.Info("Down")
		c.Down()
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		//slog.Info("ZoomDown")
		c.ZoomDown()
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		//slog.Info("ZoomUp")
		c.ZoomUp()
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		//slog.Info("w/h", zap.Int("w", c.cfg.Width()), zap.Int("h", c.cfg.Height()))
		// TODO сделать по центру карты
		c.Reset(0, 0)
	}
	return nil
}

func (c *Camera) WorldMatrix() ebiten.GeoM {
	c.geom.Reset()
	c.geom.Translate(c.RelativePixels.Min.X, c.RelativePixels.Min.Y)
	// We want to scale and rotate around center of image / screen
	c.geom.Translate(-c.W.GetWidth()/2, -c.W.GetHeight()/2)
	c.geom.Scale(
		c.scaleFactor,
		c.scaleFactor,
	)
	c.geom.Translate(c.W.GetWidth()/2, c.W.GetHeight()/2)
	return c.geom
}
