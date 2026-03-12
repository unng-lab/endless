package endless

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/camera"
	gamescenario "github.com/unng-lab/endless/pkg/endless/scenario"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/pathfinding"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	DefaultScreenWidth  = 1280
	DefaultScreenHeight = 720

	mapColumns = 10000
	mapRows    = 10000
	tileSize   = 16.0

	minZoom  = 0.2
	maxZoom  = 5.0
	zoomStep = 0.12
	panStep  = 1400.0 / 60.0
)

type Game struct {
	cam      *camera.Camera
	world    world.World
	atlas    *assets.TileAtlas
	tile     *ebiten.Image
	units    *unit.Manager
	scenario gamescenario.Scenario

	tileRenderWorkers []chan tileRenderRequest
	tileRenderTargets []*ebiten.Image
	tileRenderWG      sync.WaitGroup

	screenWidth  int
	screenHeight int

	dragging    bool
	lastCursorX int
	lastCursorY int

	renderedTiles int
	assetErr      error
	pathErr       error
	fireErr       error
	tickCounter   int64

	startedAt         time.Time
	firstUpdateLogged bool
	firstDrawLogged   bool
}

// NewGame builds the regular lightweight scene used by the default desktop launcher.
func NewGame() (*Game, error) {
	return NewGameWithConfig(GameConfig{Mode: gamescenario.ModeBasic})
}

// NewStressGame builds the dedicated heavy-load scene used by the separate stress launcher.
func NewStressGame() (*Game, error) {
	return NewGameWithConfig(GameConfig{Mode: gamescenario.ModeStress})
}

// NewGameWithConfig constructs the game core and chooses one startup scenario that will seed
// units and optional orchestration on top of the shared world, camera and rendering systems.
func NewGameWithConfig(config GameConfig) (*Game, error) {
	startedAt := time.Now()
	log.Printf("[startup] game: NewGame started")
	config = normalizedGameConfig(config)

	tileStartedAt := time.Now()
	tile := ebiten.NewImage(1, 1)
	tile.Fill(color.White)
	log.Printf("[startup] game: base tile image prepared in %s", time.Since(tileStartedAt))

	worldStartedAt := time.Now()
	gameWorld := world.New(world.Config{
		Columns:  mapColumns,
		Rows:     mapRows,
		TileSize: tileSize,
	})
	log.Printf("[startup] game: world created in %s (%dx%d tiles, tile size %.1f)", time.Since(worldStartedAt), mapColumns, mapRows, tileSize)

	scenarioStartedAt := time.Now()
	selectedScenario, err := gamescenario.New(gamescenario.Config{
		Mode:   config.Mode,
		RLDuel: config.RLDuel,
	}, gameWorld)
	if err != nil {
		return nil, fmt.Errorf("create %s scenario: %w", config.Mode, err)
	}
	log.Printf("[startup] game: %s scenario prepared in %s", config.Mode, time.Since(scenarioStartedAt))

	structStartedAt := time.Now()
	g := &Game{
		cam: camera.New(camera.Config{
			Scale:    1,
			MinScale: minZoom,
			MaxScale: maxZoom,
		}),
		world:        gameWorld,
		atlas:        assets.NewTileAtlas(),
		tile:         tile,
		scenario:     selectedScenario,
		screenWidth:  DefaultScreenWidth,
		screenHeight: DefaultScreenHeight,
		startedAt:    startedAt,
	}
	log.Printf("[startup] game: camera, atlas and game struct initialized in %s", time.Since(structStartedAt))

	managerStartedAt := time.Now()
	g.units = unit.NewManager(g.world)
	log.Printf("[startup] game: unit manager initialized in %s", time.Since(managerStartedAt))

	tileRenderStartedAt := time.Now()
	g.startTileRenderWorkers()
	log.Printf("[startup] game: tile render worker pool started in %s", time.Since(tileRenderStartedAt))

	if g.scenario != nil {
		seedStartedAt := time.Now()
		g.scenario.SeedUnits(g.units)
		log.Printf("[startup] game: scenario units seeded in %s", time.Since(seedStartedAt))
	}

	cameraStartedAt := time.Now()
	g.centerCamera()
	g.clampCamera()
	log.Printf("[startup] game: initial camera placement completed in %s", time.Since(cameraStartedAt))
	log.Printf("[startup] game: NewGame finished in %s", time.Since(startedAt))

	return g, nil
}

func (g *Game) Update() error {
	if !g.firstUpdateLogged {
		g.firstUpdateLogged = true
		log.Printf("[startup] game: first Update reached after %s", time.Since(g.startedAt))
	}

	g.tickCounter++
	if g.scenario != nil {
		g.scenario.Update(g.tickCounter, g.units)
	}
	g.units.Update(g.tickCounter)
	g.updateCameraControls()
	g.handleGameplayInput()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.firstDrawLogged {
		g.firstDrawLogged = true
		log.Printf("[startup] game: first Draw reached after %s", time.Since(g.startedAt))
	}

	screen.Fill(color.NRGBA{R: 17, G: 24, B: 31, A: 255})

	g.updateScreenSize(screen)

	visible, quality, hoveredTileX, hoveredTileY, hovered := g.drawWorld(screen)
	if err := g.units.Draw(screen, g.cam, quality, visible, true); err != nil && g.assetErr == nil {
		g.assetErr = err
	}

	if hovered {
		g.drawTileHighlight(screen, hoveredTileX, hoveredTileY)
	}
	g.units.DrawOverlay(screen, g.cam, g.screenWidth, g.screenHeight)
	ebitenutil.DebugPrint(screen, g.debugText(hoveredTileX, hoveredTileY, hovered))
}

func (g *Game) updateCameraControls() {
	g.applyZoomInput()
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.centerCamera()
	}
	g.handleKeyboardPan()
	g.handleMouseDrag()
	g.clampCamera()
}

func (g *Game) applyZoomInput() {
	_, wheelY := ebiten.Wheel()
	if wheelY == 0 {
		return
	}

	x, y := ebiten.CursorPosition()
	g.cam.Zoom(wheelY*zoomStep, geom.Point{X: float64(x), Y: float64(y)})
}

func (g *Game) handleGameplayInput() {
	g.handleUnitSelection()
	g.handleUnitCommand()
	g.handleUnitFire()
}

func (g *Game) updateScreenSize(screen *ebiten.Image) {
	bounds := screen.Bounds()
	g.screenWidth = bounds.Dx()
	g.screenHeight = bounds.Dy()
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
	speed := panStep / g.cam.Scale()
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

func (g *Game) handleUnitSelection() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	x, y := ebiten.CursorPosition()
	cursor := geom.Point{X: float64(x), Y: float64(y)}
	g.units.SelectAtScreen(g.cam, cursor, g.screenWidth, g.screenHeight)
}

func (g *Game) handleUnitCommand() {
	if !g.units.HasSelected() || !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		return
	}

	x, y, _, ok := g.cursorWorldCommandTarget()
	if !ok {
		return
	}

	targetTileX, targetTileY, ok := g.hoveredTile(x, y)
	if !ok {
		g.pathErr = pathfinding.ErrNoPath
		return
	}

	if err := g.units.CommandSelectedMove(targetTileX, targetTileY); err != nil {
		g.pathErr = err
		return
	}

	g.pathErr = nil
}

func (g *Game) handleUnitFire() {
	if !g.units.HasSelected() || !inpututil.IsKeyJustPressed(ebiten.KeyF) {
		return
	}

	_, _, cursor, ok := g.cursorWorldCommandTarget()
	if !ok {
		return
	}

	if err := g.units.CommandSelectedFire(g.cam.ScreenToWorld(cursor)); err != nil {
		g.fireErr = err
		return
	}

	g.fireErr = nil
}

func (g *Game) centerCamera() {
	g.cam.SetPosition(geom.Point{
		X: g.world.Width()/2 - float64(g.screenWidth)/2,
		Y: g.world.Height()/2 - float64(g.screenHeight)/2,
	})
}

func (g *Game) clampCamera() {
	g.cam.SetPosition(g.world.ClampCamera(g.cam.Position(), g.cam.Scale(), g.screenWidth, g.screenHeight))
}

// hoveredTile converts the current cursor position from screen space into the tile grid and
// rejects points outside the procedural world. Keeping this in one helper avoids scattering
// float-to-tile boundary checks across input handling and debug rendering.
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

func (g *Game) cursorWorldCommandTarget() (int, int, geom.Point, bool) {
	x, y := ebiten.CursorPosition()
	cursor := geom.Point{X: float64(x), Y: float64(y)}
	if g.units.PointInPanel(g.cam, cursor, g.screenWidth, g.screenHeight) {
		return 0, 0, geom.Point{}, false
	}

	return x, y, cursor, true
}

// drawWorld renders the visible tile slice and returns the exact tile rectangle that was just
// drawn. The caller reuses that same rectangle for the unit pass so terrain and tile-stack
// bodies stay perfectly aligned in both coverage and traversal order.
func (g *Game) drawWorld(screen *ebiten.Image) (image.Rectangle, assets.Quality, int, int, bool) {
	visible := g.world.VisibleRange(g.cam.ViewRect(float64(g.screenWidth), float64(g.screenHeight)))
	scale := g.cam.Scale()
	camPos := g.cam.Position()
	tileScreenSize := g.world.TileSize() * scale
	drawSize := math.Max(tileScreenSize, 1)
	quality := g.atlas.QualityForScreenSize(tileScreenSize)
	cursorX, cursorY := ebiten.CursorPosition()
	hoveredTileX, hoveredTileY, hovered := g.hoveredTile(cursorX, cursorY)

	renderedTiles, err := g.drawVisibleTiles(screen, visible, drawSize, scale, camPos, quality)
	if err != nil && g.assetErr == nil {
		g.assetErr = err
	}
	g.renderedTiles = renderedTiles

	return visible, quality, hoveredTileX, hoveredTileY, hovered
}

func (g *Game) drawTileHighlight(screen *ebiten.Image, tileX int, tileY int) {
	scale := g.cam.Scale()
	drawSize := math.Max(g.world.TileSize()*scale, 1)
	screenX, screenY := g.tileScreenPosition(tileX, tileY, scale, g.cam.Position())
	border := math.Max(1, math.Round(scale))
	left := screenX
	top := screenY
	right := left + drawSize
	bottom := top + drawSize

	g.drawHighlightRect(screen, left, top, drawSize, border)
	g.drawHighlightRect(screen, left, bottom-border, drawSize, border)
	g.drawHighlightRect(screen, left, top+border, border, math.Max(bottom-top-border*2, 0))
	g.drawHighlightRect(screen, right-border, top+border, border, math.Max(bottom-top-border*2, 0))
}

// tileScreenPosition centralizes the world-to-screen projection for tile anchors so tile
// drawing and highlight rendering cannot drift apart when camera math changes.
func (g *Game) tileScreenPosition(tileX int, tileY int, scale float64, camPos geom.Point) (float64, float64) {
	return (float64(tileX)*g.world.TileSize() - camPos.X) * scale,
		(float64(tileY)*g.world.TileSize() - camPos.Y) * scale
}

func (g *Game) drawHighlightRect(screen *ebiten.Image, x float64, y float64, width float64, height float64) {
	g.drawFilledRect(screen, x, y, width, height, color.White)
}

func (g *Game) drawFilledRect(screen *ebiten.Image, x float64, y float64, width float64, height float64, fill color.Color) {
	if width <= 0 || height <= 0 {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(fill)

	screen.DrawImage(g.tile, &op)
}

func (g *Game) debugText(hoveredTileX, hoveredTileY int, hovered bool) string {
	hoveredTileText := "Hovered tile: outside world"
	if hovered {
		tileType := g.world.TileType(hoveredTileX, hoveredTileY)
		hoveredTileText = fmt.Sprintf(
			"Hovered tile: (%d, %d)  Type: %s  Speed: %.0f%%",
			hoveredTileX,
			hoveredTileY,
			tileType,
			tileType.SpeedMultiplier()*100,
		)
	}

	debugText := fmt.Sprintf(
		"WASD/Arrows: move  Shift: faster  Space: center  Middle mouse: drag  Wheel: zoom to cursor  Left mouse: select unit  Right mouse: move selected unit  F: fire to cursor\nTPS: %.1f  RPS: %.1f  Zoom: %.2fx  Visible tiles: %d  Camera: (%.0f, %.0f)  %s",
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
	if g.pathErr != nil {
		debugText += "\nPath command: " + g.pathErr.Error()
	}
	if g.fireErr != nil {
		debugText += "\nFire command: " + g.fireErr.Error()
	}
	if g.scenario != nil {
		if scenarioDebugText := g.scenario.DebugText(); scenarioDebugText != "" {
			debugText += "\n" + scenarioDebugText
		}
	}

	return debugText
}

func (g *Game) tileImage(x, y int, quality assets.Quality) (*ebiten.Image, float64, error) {
	tileSize := g.atlas.TileSize(quality)
	if tileSize == 0 {
		return nil, 0, nil
	}

	tileImage, err := g.atlas.TileImage(g.world.TileIndex(x, y), quality)
	if err != nil {
		return nil, 0, err
	}

	return tileImage, float64(tileSize), nil
}
