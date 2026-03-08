package endless

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	DefaultScreenWidth  = 1280
	DefaultScreenHeight = 720

	mapColumns = 10000
	mapRows    = 10000
	tileSize   = 16.0

	minZoom  = 0.01
	maxZoom  = 5.0
	zoomStep = 0.12
	panSpeed = 1400.0
	tps      = 60.0
)

type Game struct {
	cam   *camera.Camera
	world world.World
	atlas *assets.TileAtlas
	tile  *ebiten.Image

	screenWidth  int
	screenHeight int

	dragging    bool
	lastCursorX int
	lastCursorY int

	renderedTiles int
	assetErr      error
}

func NewGame() *Game {
	tile := ebiten.NewImage(1, 1)
	tile.Fill(color.White)

	gameWorld := world.New(world.Config{
		Columns:  mapColumns,
		Rows:     mapRows,
		TileSize: tileSize,
	})

	g := &Game{
		cam: camera.New(camera.Config{
			Scale:    1,
			MinScale: minZoom,
			MaxScale: maxZoom,
		}),
		world:        gameWorld,
		atlas:        assets.NewTileAtlas(),
		tile:         tile,
		screenWidth:  DefaultScreenWidth,
		screenHeight: DefaultScreenHeight,
	}

	g.cam.SetPosition(geom.Point{
		X: g.world.Width()/2 - float64(g.screenWidth)/2,
		Y: g.world.Height()/2 - float64(g.screenHeight)/2,
	})
	g.clampCamera()

	return g
}

func (g *Game) Update() error {
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		x, y := ebiten.CursorPosition()
		g.cam.Zoom(wheelY*zoomStep, geom.Point{X: float64(x), Y: float64(y)})
	}

	g.handleKeyboardPan()
	g.handleMouseDrag()
	g.clampCamera()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 17, G: 24, B: 31, A: 255})

	bounds := screen.Bounds()
	g.screenWidth = bounds.Dx()
	g.screenHeight = bounds.Dy()

	visible := g.world.VisibleRange(g.cam.ViewRect(float64(bounds.Dx()), float64(bounds.Dy())))
	scale := g.cam.Scale()
	camPos := g.cam.Position()
	tileScreenSize := g.world.TileSize() * scale
	gap := world.TileGap(tileScreenSize)
	drawSize := math.Max(tileScreenSize-gap, 1)
	offset := gap / 2
	quality := g.atlas.QualityForScreenSize(tileScreenSize)
	cursorX, cursorY := ebiten.CursorPosition()
	hoveredTileX, hoveredTileY, hovered := g.hoveredTile(cursorX, cursorY)

	g.renderedTiles = 0
	for y := visible.Min.Y; y < visible.Max.Y; y++ {
		for x := visible.Min.X; x < visible.Max.X; x++ {
			screenX := (float64(x)*g.world.TileSize() - camPos.X) * scale
			screenY := (float64(y)*g.world.TileSize() - camPos.Y) * scale

			var op ebiten.DrawImageOptions
			tileImage := g.tile
			if atlasTile, atlasTileSize, ok := g.tileImage(x, y, quality); ok {
				tileImage = atlasTile
				op.GeoM.Scale(drawSize/atlasTileSize, drawSize/atlasTileSize)
			} else {
				op.GeoM.Scale(drawSize, drawSize)
				op.ColorScale.ScaleWithColor(world.TileColor(x, y))
			}
			op.GeoM.Translate(screenX+offset, screenY+offset)

			screen.DrawImage(tileImage, &op)
			g.renderedTiles++
		}
	}

	if hovered {
		g.drawTileHighlight(screen, hoveredTileX, hoveredTileY, drawSize, offset, scale, camPos)
	}

	hoveredTileText := "Hovered tile: outside world"
	if hovered {
		hoveredTileText = fmt.Sprintf("Hovered tile: (%d, %d)", hoveredTileX, hoveredTileY)
	}

	debugText := fmt.Sprintf(
		"WASD/Arrows: move  Shift: faster  Middle mouse: drag  Wheel: zoom to cursor\nTPS: %.1f  RPS: %.1f  Zoom: %.2fx  Visible tiles: %d  Camera: (%.0f, %.0f)  %s",
		ebiten.ActualTPS(),
		ebiten.ActualFPS(),
		g.cam.Scale(),
		g.renderedTiles,
		g.cam.Position().X,
		g.cam.Position().Y,
		hoveredTileText,
	)
	if g.assetErr != nil {
		debugText += "\nAssets fallback: " + g.assetErr.Error()
	}
	ebitenutil.DebugPrint(screen, debugText)
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

func (g *Game) clampCamera() {
	g.cam.SetPosition(g.world.ClampCamera(g.cam.Position(), g.cam.Scale(), g.screenWidth, g.screenHeight))
}

func (g *Game) hoveredTile(cursorX, cursorY int) (int, int, bool) {
	worldPos := g.cam.ScreenToWorld(geom.Point{
		X: float64(cursorX),
		Y: float64(cursorY),
	})
	if worldPos.X < 0 || worldPos.Y < 0 || worldPos.X >= g.world.Width() || worldPos.Y >= g.world.Height() {
		return 0, 0, false
	}

	tileX := int(math.Floor(worldPos.X / g.world.TileSize()))
	tileY := int(math.Floor(worldPos.Y / g.world.TileSize()))
	if tileX < 0 || tileX >= g.world.Columns() || tileY < 0 || tileY >= g.world.Rows() {
		return 0, 0, false
	}

	return tileX, tileY, true
}

func (g *Game) drawTileHighlight(
	screen *ebiten.Image,
	tileX int,
	tileY int,
	drawSize float64,
	offset float64,
	scale float64,
	camPos geom.Point,
) {
	screenX := (float64(tileX)*g.world.TileSize() - camPos.X) * scale
	screenY := (float64(tileY)*g.world.TileSize() - camPos.Y) * scale
	border := math.Max(1, math.Round(scale))
	left := screenX + offset
	top := screenY + offset
	right := left + drawSize
	bottom := top + drawSize

	g.drawHighlightRect(screen, left, top, drawSize, border)
	g.drawHighlightRect(screen, left, bottom-border, drawSize, border)
	g.drawHighlightRect(screen, left, top+border, border, math.Max(bottom-top-border*2, 0))
	g.drawHighlightRect(screen, right-border, top+border, border, math.Max(bottom-top-border*2, 0))
}

func (g *Game) drawHighlightRect(screen *ebiten.Image, x float64, y float64, width float64, height float64) {
	if width <= 0 || height <= 0 {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)

	screen.DrawImage(g.tile, &op)
}

func (g *Game) tileImage(x, y int, quality assets.Quality) (*ebiten.Image, float64, bool) {
	tileSize := g.atlas.TileSize(quality)
	if tileSize == 0 {
		return nil, 0, false
	}

	tileImage, err := g.atlas.TileImage(world.TileIndex(x, y), quality)
	if err != nil {
		if g.assetErr == nil {
			g.assetErr = err
		}
		return nil, 0, false
	}

	return tileImage, float64(tileSize), true
}
