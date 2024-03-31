package camera

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/geom"
	"github/unng-lab/madfarmer/internal/window"
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
	TileSizeX      float64
	TileSizeY      float64
	Coordinates    geom.Rectangle
	AbsolutePixels geom.Rectangle
	RelativePixels geom.Rectangle
	ScaleFactorX   float64
	ScaleFactorY   float64
	W              window.Window
	geom           ebiten.GeoM
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

func (c *Camera) scale() float64 {
	return math.Pow(1.01, c.zoomFactor)
}

func (c *Camera) scaleX() float64 {
	return c.scale() * c.W.GetWidth() / DefaultScreenWidth
}

func (c *Camera) scaleY() float64 {
	return c.scale() * c.W.GetHeight() / DefaultScreenHeight
}

func (c *Camera) getTileSizeX() float64 {
	return c.cfg.TileSize * c.ScaleFactorX
}

func (c *Camera) getTileSizeY() float64 {
	return c.cfg.TileSize * c.ScaleFactorY
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

func (c *Camera) GetScaleFactorX() float64 {
	return c.ScaleFactorX
}

func (c *Camera) GetCurrentCoordinates() geom.Rectangle {
	return c.Coordinates
}

func (c *Camera) Prepare() Camera {
	c.ScaleFactorX = c.scaleX()
	c.ScaleFactorY = c.scaleY()
	c.TileSizeX = c.getTileSizeX()
	c.TileSizeY = c.getTileSizeY()
	maxX, maxY := (c.W.GetWidth())/c.TileSizeX+1, (c.W.GetHeight())/c.TileSizeY+1

	var (
		x, y         float64
		cellX, cellY float64 = c.cfg.TileCount / 2, c.cfg.TileCount / 2
	)

	shiftX, shiftY := math.Mod(c.positionX, c.TileSizeX), math.Mod(c.positionY, c.TileSizeY)
	if shiftX < 0 {
		x = -c.TileSizeX - shiftX
		cellX += -1
	} else if shiftX > 0 {
		x = -shiftX
	}
	cellX += math.Trunc(c.positionX / c.TileSizeX)

	if shiftY < 0 {
		y = -c.TileSizeY - shiftY
		cellY += -1
	} else if shiftY > 0 {
		y = -shiftY
	}
	cellY += math.Trunc(c.positionY / c.TileSizeY)

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
			X: math.Trunc(c.TileSizeX*cellX*100)/100 - x,
			Y: math.Trunc(c.TileSizeX*cellY*100)/100 - y,
		},
		Max: geom.Point{
			X: math.Trunc(c.TileSizeX*(cellX+maxX)*100)/100 - x,
			Y: math.Trunc(c.TileSizeX*(cellY+maxY)*100)/100 - y,
		},
	}
	c.RelativePixels = geom.Rectangle{
		Min: geom.Point{
			X: x,
			Y: y,
		},
		Max: geom.Point{
			X: x + c.TileSizeX*maxX,
			Y: y + c.TileSizeX*maxY,
		},
	}
	return *c
}

func (c *Camera) GetCurrentPixels() geom.Rectangle {
	return c.AbsolutePixels
}

func (c *Camera) MiddleOfPointInRelativePixels(point geom.Point) geom.Point {
	distX, distY := c.Coordinates.Min.Distance(point)
	return geom.Point{
		X: c.RelativePixels.Min.X + distX*c.TileSizeX + c.TileSizeX/2,
		Y: c.RelativePixels.Min.Y + distY*c.TileSizeX + c.TileSizeX/2,
	}
}

//func AbsoluteToRelative(point geom.Point) geom.Point {
//	return geom.Point{
//		X: math.Trunc(point.X / c.cfg.TileSize),
//		Y: math.Trunc(point.Y / c.cfg.TileSize),
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
		X: point.X*c.TileSizeX - c.AbsolutePixels.Min.X,
		Y: point.Y*c.TileSizeY - c.AbsolutePixels.Min.Y,
	}
}

//func (c *Camera) DrawPixelMinPoint() (float64, float64) {
//	return c.AbsolutePixels.Min.X - c.TileSize*CountTile/2,
//		c.AbsolutePixels.Min.Y - c.TileSize*CountTile/2
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
		TileSizeX:  0,
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
		ScaleFactorX: 0,
		ScaleFactorY: 0,
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
	c.geom.Translate(-c.RelativePixels.Min.X, -c.RelativePixels.Min.Y)
	// We want to scale and rotate around center of image / screen
	c.geom.Translate(-c.W.GetWidth()/2, -c.W.GetHeight()/2)
	c.geom.Scale(
		c.GetScaleFactorX(),
		c.GetScaleFactorX(),
	)
	c.geom.Translate(c.W.GetWidth()/2, c.W.GetHeight()/2)
	return c.geom
}
