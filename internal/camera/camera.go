package camera

import (
	"math"

	"github/unng-lab/madfarmer/internal/geom"
	"github/unng-lab/madfarmer/internal/window"
)

const (
	MaxZoom = 200
	MinZoom = -100
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
	TileSize       float64
	Coordinates    geom.Rectangle
	AbsolutePixels geom.Rectangle
	RelativePixels geom.Rectangle
	ScaleFactor    float64
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

func (c *Camera) GetTileSize() float64 {
	return c.cfg.TileSize * c.ScaleFactor
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

func (c *Camera) GetScaleFactor() float64 {
	return c.ScaleFactor
}

func (c *Camera) GetCurrentCoordinates() geom.Rectangle {
	return c.Coordinates
}

func (c *Camera) Prepare() Camera {
	c.ScaleFactor = c.scale()
	c.TileSize = c.GetTileSize()
	maxX, maxY := (window.W.GetWidth())/c.TileSize+1, (window.W.GetHeight())/c.TileSize+1

	var (
		x, y         float64
		cellX, cellY float64 = c.cfg.TileCount / 2, c.cfg.TileCount / 2
	)

	shiftX, shiftY := math.Mod(c.positionX, c.TileSize), math.Mod(c.positionY, c.TileSize)
	if shiftX < 0 {
		x = -c.TileSize - shiftX
		cellX += -1
	} else if shiftX > 0 {
		x = -shiftX
	}
	cellX += math.Trunc(c.positionX / c.TileSize)

	if shiftY < 0 {
		y = -c.TileSize - shiftY
		cellY += -1
	} else if shiftY > 0 {
		y = -shiftY
	}
	cellY += math.Trunc(c.positionY / c.TileSize)

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
			X: math.Trunc(c.TileSize*cellX*100)/100 - x,
			Y: math.Trunc(c.TileSize*cellY*100)/100 - y,
		},
		Max: geom.Point{
			X: math.Trunc(c.TileSize*(cellX+maxX)*100)/100 - x,
			Y: math.Trunc(c.TileSize*(cellY+maxY)*100)/100 - y,
		},
	}
	c.RelativePixels = geom.Rectangle{
		Min: geom.Point{
			X: x,
			Y: y,
		},
		Max: geom.Point{
			X: x + c.TileSize*maxX,
			Y: y + c.TileSize*maxY,
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
		X: c.RelativePixels.Min.X + distX*c.TileSize + c.TileSize/2,
		Y: c.RelativePixels.Min.Y + distY*c.TileSize + c.TileSize/2,
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
		X: point.X*c.TileSize - c.AbsolutePixels.Min.X,
		Y: point.Y*c.TileSize - c.AbsolutePixels.Min.Y,
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
		TileSize:   0,
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
		ScaleFactor: 0,
	}
}
