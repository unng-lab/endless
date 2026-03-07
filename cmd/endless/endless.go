package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	defaultScreenWidth  = 1280
	defaultScreenHeight = 720

	mapColumns = 10000
	mapRows    = 10000
	tileSize   = 64.0

	minZoom  = 0.01
	maxZoom  = 5.0
	zoomStep = 0.12
	panSpeed = 1400.0
	tps      = 60.0
)

type Game struct {
	cam *Camera

	tile *ebiten.Image

	screenWidth  int
	screenHeight int

	dragging    bool
	lastCursorX int
	lastCursorY int

	renderedTiles int
}

func NewGame() *Game {
	tile := ebiten.NewImage(1, 1)
	tile.Fill(color.White)

	g := &Game{
		cam: NewCamera(CameraConfig{
			Scale:    1,
			MinScale: minZoom,
			MaxScale: maxZoom,
		}),
		tile:         tile,
		screenWidth:  defaultScreenWidth,
		screenHeight: defaultScreenHeight,
	}

	g.cam.SetPosition(Point{
		X: float64(mapColumns)*tileSize/2 - float64(g.screenWidth)/2,
		Y: float64(mapRows)*tileSize/2 - float64(g.screenHeight)/2,
	})
	g.clampCamera()

	return g
}

func (g *Game) Update() error {
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		x, y := ebiten.CursorPosition()
		g.cam.Zoom(wheelY*zoomStep, Point{X: float64(x), Y: float64(y)})
	}

	g.handleKeyboardPan()
	g.handleMouseDrag()
	g.clampCamera()

	return nil
}

func (g *Game) handleKeyboardPan() {
	speed := panSpeed / g.cam.Scale() / tps
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		speed *= 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.cam.Move(-speed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.cam.Move(speed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.cam.Move(0, -speed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.cam.Move(0, speed)
	}
}

func (g *Game) handleMouseDrag() {
	x, y := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		if g.dragging {
			dx := float64(x-g.lastCursorX) / g.cam.Scale()
			dy := float64(y-g.lastCursorY) / g.cam.Scale()
			g.cam.Move(-dx, -dy)
		}
		g.dragging = true
		g.lastCursorX = x
		g.lastCursorY = y
		return
	}

	g.dragging = false
	g.lastCursorX = x
	g.lastCursorY = y
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 17, G: 24, B: 31, A: 255})

	bounds := screen.Bounds()
	g.screenWidth = bounds.Dx()
	g.screenHeight = bounds.Dy()

	visible := g.visibleRange(bounds.Dx(), bounds.Dy())
	scale := g.cam.Scale()
	camPos := g.cam.Position()
	tileScreenSize := tileSize * scale
	gap := tileGap(tileScreenSize)
	drawSize := math.Max(tileScreenSize-gap, 1)
	offset := gap / 2

	g.renderedTiles = 0
	for y := visible.Min.Y; y < visible.Max.Y; y++ {
		for x := visible.Min.X; x < visible.Max.X; x++ {
			screenX := (float64(x)*tileSize - camPos.X) * scale
			screenY := (float64(y)*tileSize - camPos.Y) * scale

			var op ebiten.DrawImageOptions
			op.GeoM.Scale(drawSize, drawSize)
			op.GeoM.Translate(screenX+offset, screenY+offset)
			op.ColorScale.ScaleWithColor(tileColor(x, y))

			screen.DrawImage(g.tile, &op)
			g.renderedTiles++
		}
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"WASD/Arrows: move  Shift: faster  Middle mouse: drag  Wheel: zoom to cursor\nZoom: %.2fx  Visible tiles: %d  Camera: (%.0f, %.0f)",
		g.cam.Scale(),
		g.renderedTiles,
		g.cam.Position().X,
		g.cam.Position().Y,
	))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth > 0 {
		g.screenWidth = outsideWidth
	}
	if outsideHeight > 0 {
		g.screenHeight = outsideHeight
	}

	g.clampCamera()

	return g.screenWidth, g.screenHeight
}

func (g *Game) visibleRange(screenWidth, screenHeight int) image.Rectangle {
	rect := g.cam.ViewRect(float64(screenWidth), float64(screenHeight))

	minX := clampInt(int(math.Floor(rect.Min.X/tileSize)), 0, mapColumns)
	minY := clampInt(int(math.Floor(rect.Min.Y/tileSize)), 0, mapRows)
	maxX := clampInt(int(math.Ceil(rect.Max.X/tileSize))+1, 0, mapColumns)
	maxY := clampInt(int(math.Ceil(rect.Max.Y/tileSize))+1, 0, mapRows)

	return image.Rect(minX, minY, maxX, maxY)
}

func (g *Game) clampCamera() {
	viewWidth := float64(g.screenWidth) / g.cam.Scale()
	viewHeight := float64(g.screenHeight) / g.cam.Scale()
	maxX := math.Max(float64(mapColumns)*tileSize-viewWidth, 0)
	maxY := math.Max(float64(mapRows)*tileSize-viewHeight, 0)

	pos := g.cam.Position()
	pos.X = clampFloat(pos.X, 0, maxX)
	pos.Y = clampFloat(pos.Y, 0, maxY)
	g.cam.SetPosition(pos)
}

func tileGap(tileScreenSize float64) float64 {
	switch {
	case tileScreenSize >= 28:
		return 2
	case tileScreenSize >= 12:
		return 1
	default:
		return 0
	}
}

func tileColor(x, y int) color.NRGBA {
	switch {
	case (x+y)%17 == 0:
		return color.NRGBA{R: 165, G: 121, B: 73, A: 255}
	case (x/8+y/8)%2 == 0:
		return color.NRGBA{R: 69, G: 118, B: 82, A: 255}
	case (x+y)%2 == 0:
		return color.NRGBA{R: 78, G: 138, B: 93, A: 255}
	default:
		return color.NRGBA{R: 95, G: 148, B: 108, A: 255}
	}
}

func clampFloat(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

type Point struct {
	X float64
	Y float64
}

type Rect struct {
	Min Point
	Max Point
}

type CameraConfig struct {
	Position Point
	Scale    float64
	MinScale float64
	MaxScale float64
}

type Camera struct {
	position Point
	scale    float64
	minScale float64
	maxScale float64
}

func NewCamera(cfg CameraConfig) *Camera {
	scale := cfg.Scale
	if scale == 0 {
		scale = 1
	}
	minScale := cfg.MinScale
	if minScale == 0 {
		minScale = 0.25
	}
	maxScale := cfg.MaxScale
	if maxScale == 0 {
		maxScale = 4
	}

	return &Camera{
		position: cfg.Position,
		scale:    clampFloat(scale, minScale, maxScale),
		minScale: minScale,
		maxScale: maxScale,
	}
}

func (c *Camera) Position() Point {
	return c.position
}

func (c *Camera) SetPosition(pos Point) {
	c.position = pos
}

func (c *Camera) Move(dx, dy float64) {
	c.position.X += dx
	c.position.Y += dy
}

func (c *Camera) Scale() float64 {
	return c.scale
}

func (c *Camera) Zoom(delta float64, cursor Point) bool {
	if delta == 0 {
		return false
	}

	newScale := clampFloat(c.scale*(1+delta), c.minScale, c.maxScale)
	if almostEqual(newScale, c.scale) {
		return false
	}

	worldBefore := c.ScreenToWorld(cursor)
	c.scale = newScale
	c.position.X = worldBefore.X - cursor.X/c.scale
	c.position.Y = worldBefore.Y - cursor.Y/c.scale

	return true
}

func (c *Camera) ScreenToWorld(screen Point) Point {
	return Point{
		X: c.position.X + screen.X/c.scale,
		Y: c.position.Y + screen.Y/c.scale,
	}
}

func (c *Camera) ViewRect(screenWidth, screenHeight float64) Rect {
	return Rect{
		Min: c.position,
		Max: Point{
			X: c.position.X + screenWidth/c.scale,
			Y: c.position.Y + screenHeight/c.scale,
		},
	}
}

func almostEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) < eps
}

func main() {
	ebiten.SetWindowTitle("Endless")
	ebiten.SetWindowSize(defaultScreenWidth, defaultScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(false)

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatalf("run endless: %v", err)
	}
}
