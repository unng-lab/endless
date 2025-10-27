package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/render"
	"github.com/unng-lab/endless/internal/new/tilemap"
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

	return &Game{cfg: cfg, cam: cam, tileMap: tiles, renderer: renderer}
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

	return nil
}

// Draw renders a frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 24, G: 24, B: 24, A: 255})
	g.renderer.Draw(screen, g.cam)
}

// Layout returns the logical screen dimensions.
func (g *Game) Layout(_, _ int) (int, int) {
	return g.cfg.ScreenWidth, g.cfg.ScreenHeight
}
