package unit

import (
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/assets/img"
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

const (
	sheetWidth     = 256
	sheetHeight    = 96
	frameWidth     = 32
	frameHeight    = 32
	frameStartY    = 32
	frameCount     = 8
	unitScaleTiles = 2.0
	unitAnchorY    = 0.85
)

type Renderer struct {
	mu     sync.Mutex
	sheets map[assets.Quality]map[Kind]*ebiten.Image
	frames map[assets.Quality]map[Kind]map[int]*ebiten.Image
}

func NewRenderer() *Renderer {
	return &Renderer{
		sheets: make(map[assets.Quality]map[Kind]*ebiten.Image),
		frames: make(map[assets.Quality]map[Kind]map[int]*ebiten.Image),
	}
}

func (r *Renderer) Draw(
	screen *ebiten.Image,
	cam *camera.Camera,
	worldTileSize float64,
	quality assets.Quality,
	units []Unit,
) error {
	if len(units) == 0 {
		return nil
	}

	camPos := cam.Position()
	scale := cam.Scale()
	screenUnitSize := worldTileSize * scale * unitScaleTiles

	for _, u := range units {
		frame, err := r.frameImage(u.Kind, u.Frame(), quality)
		if err != nil {
			return err
		}

		frameBounds := frame.Bounds()
		frameScale := screenUnitSize / float64(frameBounds.Dx())
		screenX := (u.Position.X - camPos.X) * scale
		screenY := (u.Position.Y - camPos.Y) * scale

		var op ebiten.DrawImageOptions
		op.GeoM.Scale(frameScale, frameScale)
		op.GeoM.Translate(screenX-screenUnitSize/2, screenY-screenUnitSize*unitAnchorY)

		screen.DrawImage(frame, &op)
	}

	return nil
}

func ScreenRect(cam *camera.Camera, worldTileSize float64, u Unit) geom.Rect {
	camPos := cam.Position()
	scale := cam.Scale()
	screenUnitSize := worldTileSize * scale * unitScaleTiles
	screenX := (u.Position.X - camPos.X) * scale
	screenY := (u.Position.Y - camPos.Y) * scale

	return geom.Rect{
		Min: geom.Point{
			X: screenX - screenUnitSize/2,
			Y: screenY - screenUnitSize*unitAnchorY,
		},
		Max: geom.Point{
			X: screenX + screenUnitSize/2,
			Y: screenY + screenUnitSize*(1-unitAnchorY),
		},
	}
}

func (r *Renderer) frameImage(kind Kind, frameIndex int, quality assets.Quality) (*ebiten.Image, error) {
	if frameIndex < 0 || frameIndex >= frameCount {
		return nil, fmt.Errorf("unit frame %d is out of range", frameIndex)
	}

	sheet, cfg, err := r.ensureSheet(kind, quality)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.frames[quality]; !ok {
		r.frames[quality] = make(map[Kind]map[int]*ebiten.Image)
	}
	if _, ok := r.frames[quality][kind]; !ok {
		r.frames[quality][kind] = make(map[int]*ebiten.Image)
	}
	if frame := r.frames[quality][kind][frameIndex]; frame != nil {
		return frame, nil
	}

	x := frameIndex * cfg.frameWidth
	rect := image.Rect(x, cfg.frameStartY, x+cfg.frameWidth, cfg.frameStartY+cfg.frameHeight)
	if !rect.In(sheet.Bounds()) {
		return nil, fmt.Errorf("unit frame %d exceeds sprite sheet bounds", frameIndex)
	}

	frame := sheet.SubImage(rect).(*ebiten.Image)
	r.frames[quality][kind][frameIndex] = frame
	return frame, nil
}

type sheetConfig struct {
	fileName    string
	width       int
	height      int
	frameWidth  int
	frameHeight int
	frameStartY int
}

func (r *Renderer) ensureSheet(kind Kind, quality assets.Quality) (*ebiten.Image, sheetConfig, error) {
	cfg, err := spriteSheetConfig(kind, quality)
	if err != nil {
		return nil, sheetConfig{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sheets[quality]; !ok {
		r.sheets[quality] = make(map[Kind]*ebiten.Image)
	}
	if sheet := r.sheets[quality][kind]; sheet != nil {
		return sheet, cfg, nil
	}

	sheet, err := img.Img(cfg.fileName, uint64(cfg.width), uint64(cfg.height))
	if err != nil {
		return nil, sheetConfig{}, fmt.Errorf("load unit sprite sheet %q: %w", cfg.fileName, err)
	}

	r.sheets[quality][kind] = sheet
	return sheet, cfg, nil
}

func spriteSheetConfig(kind Kind, quality assets.Quality) (sheetConfig, error) {
	fileName := string(kind) + ".png"

	switch quality {
	case assets.QualityLow:
		return sheetConfig{
			fileName:    fileName,
			width:       sheetWidth,
			height:      sheetHeight,
			frameWidth:  frameWidth,
			frameHeight: frameHeight,
			frameStartY: frameStartY,
		}, nil
	case assets.QualityMedium:
		return sheetConfig{
			fileName:    fileName,
			width:       sheetWidth * 2,
			height:      sheetHeight * 2,
			frameWidth:  frameWidth * 2,
			frameHeight: frameHeight * 2,
			frameStartY: frameStartY * 2,
		}, nil
	case assets.QualityHigh:
		return sheetConfig{
			fileName:    fileName,
			width:       sheetWidth * 4,
			height:      sheetHeight * 4,
			frameWidth:  frameWidth * 4,
			frameHeight: frameHeight * 4,
			frameStartY: frameStartY * 4,
		}, nil
	default:
		return sheetConfig{}, fmt.Errorf("unsupported unit sprite quality %d", quality)
	}
}
