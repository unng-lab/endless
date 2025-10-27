package game

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/pathfinding"
	"github.com/unng-lab/endless/internal/new/render"
	"github.com/unng-lab/endless/internal/new/tilemap"
	"github.com/unng-lab/endless/internal/new/unit"
)

// Config controls the behaviour of the prototype game loop.
type Config struct {
	ScreenWidth  int
	ScreenHeight int
	TileColumns  int
	TileRows     int
	TileSize     float64
	MinZoom      float64
	MaxZoom      float64
	ZoomStep     float64
	PanSpeed     float64
}

const defaultTPS = 60.0

// Game implements ebiten.Game using the new camera and tile map.
type Game struct {
	cfg      Config
	cam      *camera.Camera
	tileMap  *tilemap.TileMap
	renderer *render.TileMapRenderer
	unitDraw *render.UnitRenderer
	units    *unit.Manager
}

// New creates a prototype game.
func New(cfg Config) *Game {
	if cfg.ScreenWidth <= 0 {
		cfg.ScreenWidth = 1280
	}
	if cfg.ScreenHeight <= 0 {
		cfg.ScreenHeight = 720
	}
	if cfg.ZoomStep == 0 {
		cfg.ZoomStep = 0.1
	}
	if cfg.PanSpeed == 0 {
		cfg.PanSpeed = 600
	}

	cam := camera.New(camera.Config{
		MinScale: cfg.MinZoom,
		MaxScale: cfg.MaxZoom,
	})

	tiles := tilemap.New(tilemap.Config{
		Columns:  cfg.TileColumns,
		Rows:     cfg.TileRows,
		TileSize: cfg.TileSize,
	})

	renderer := render.NewTileMapRenderer(tiles)
	unitRenderer := render.NewUnitRenderer(renderer.Atlas(), tiles.TileSize())
	navigator := pathfinding.NewNavigator(tiles, 64)
	units := unit.NewManager(tiles, navigator)
	addDefaultUnits(units, tiles)

	return &Game{cfg: cfg, cam: cam, tileMap: tiles, renderer: renderer, unitDraw: unitRenderer, units: units}
}

func addDefaultUnits(m *unit.Manager, tiles *tilemap.TileMap) {
	tileSize := tiles.TileSize()

	worker := &unit.UnitType{
		Name:  "worker",
		Speed: 140,
		Animations: map[unit.State]unit.Animation{
			unit.StateIdle:   {Frames: []int{64, 65}, FrameDuration: 0.6},
			unit.StateMoving: {Frames: []int{66, 67, 68, 69}, FrameDuration: 0.15},
		},
	}

	scout := &unit.UnitType{
		Name:  "scout",
		Speed: 220,
		Animations: map[unit.State]unit.Animation{
			unit.StateIdle:   {Frames: []int{80, 81}, FrameDuration: 0.5},
			unit.StateMoving: {Frames: []int{82, 83, 84, 85}, FrameDuration: 0.1},
		},
	}

	hauler := &unit.UnitType{
		Name:  "hauler",
		Speed: 110,
		Animations: map[unit.State]unit.Animation{
			unit.StateIdle:   {Frames: []int{96, 97}, FrameDuration: 0.7},
			unit.StateMoving: {Frames: []int{98, 99, 100, 101}, FrameDuration: 0.18},
		},
	}

	spawns := []struct {
		typ   *unit.UnitType
		start pathfinding.Point
		goal  pathfinding.Point
	}{
		{worker, pathfinding.Point{X: 40, Y: 40}, pathfinding.Point{X: 120, Y: 120}},
		{scout, pathfinding.Point{X: 42, Y: 60}, pathfinding.Point{X: 200, Y: 220}},
		{hauler, pathfinding.Point{X: 60, Y: 42}, pathfinding.Point{X: 220, Y: 160}},
	}

	for _, spawn := range spawns {
		if spawn.typ == nil {
			continue
		}
		unitPos := camera.Point{X: (float64(spawn.start.X) + 0.5) * tileSize, Y: (float64(spawn.start.Y) + 0.5) * tileSize}
		u := unit.NewUnit(spawn.typ, unitPos)
		m.Add(u)
		_ = m.CommandMove(u, spawn.goal)
	}
}

// Update advances the game state.
func (g *Game) Update() error {
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		x, y := ebiten.CursorPosition()
		g.cam.Zoom(wheelY*g.cfg.ZoomStep, camera.Point{X: float64(x), Y: float64(y)})
	}

	speed := g.cfg.PanSpeed / g.cam.Scale() / defaultTPS

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.cam.Move(-speed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.cam.Move(speed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.cam.Move(0, -speed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.cam.Move(0, speed)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		cx, cy := ebiten.CursorPosition()
		targetWorld := g.cam.ScreenToWorld(camera.Point{X: float64(cx), Y: float64(cy)})
		target := pathfinding.Point{X: int(targetWorld.X / g.tileMap.TileSize()), Y: int(targetWorld.Y / g.tileMap.TileSize())}
		for _, u := range g.units.Units() {
			if err := g.units.CommandMove(u, target); err != nil {
				log.Printf("move unit: %v", err)
			}
		}
	}

	g.units.Update(1 / defaultTPS)

	return nil
}

// Draw renders a frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 24, G: 24, B: 24, A: 255})
	g.renderer.Draw(screen, g.cam)
	g.unitDraw.Draw(screen, g.cam, g.units.Units())
}

// Layout returns the logical screen dimensions.
func (g *Game) Layout(_, _ int) (int, int) {
	return g.cfg.ScreenWidth, g.cfg.ScreenHeight
}
