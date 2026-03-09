package unit

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/assets/img"
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

const (
	sheetWidth  = 256
	sheetHeight = 96
	frameWidth  = 32
	frameHeight = 32
	frameStartY = 32
	frameCount  = 8
)

type Renderer struct {
	mu     sync.Mutex
	sheets map[assets.Quality]map[Kind]*ebiten.Image
	frames map[assets.Quality]map[Kind]map[int]*ebiten.Image
	solid  *ebiten.Image
}

func NewRenderer() *Renderer {
	solid := ebiten.NewImage(1, 1)
	solid.Fill(color.White)

	return &Renderer{
		sheets: make(map[assets.Quality]map[Kind]*ebiten.Image),
		frames: make(map[assets.Quality]map[Kind]map[int]*ebiten.Image),
		solid:  solid,
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

	for _, u := range units {
		if !u.OnScreen {
			continue
		}
		if !kindUsesSprite(u.Kind) {
			r.drawStatic(screen, camPos, scale, worldTileSize, u)
			continue
		}

		metrics := kindVisualMetrics(u.Kind)
		screenUnitWidth := worldTileSize * scale * metrics.widthTiles
		screenUnitHeight := worldTileSize * scale * metrics.heightTiles

		frame, err := r.frameImage(u.Kind, u.Frame(), quality)
		if err != nil {
			return err
		}

		frameBounds := frame.Bounds()
		frameScale := screenUnitWidth / float64(frameBounds.Dx())
		screenX := (u.Position.X - camPos.X) * scale
		screenY := (u.Position.Y - camPos.Y) * scale

		var op ebiten.DrawImageOptions
		op.GeoM.Scale(frameScale, frameScale)
		op.GeoM.Translate(screenX-screenUnitWidth/2, screenY-screenUnitHeight*metrics.anchorY)

		screen.DrawImage(frame, &op)
	}

	return nil
}

func ScreenRect(cam *camera.Camera, worldTileSize float64, u Unit) geom.Rect {
	camPos := cam.Position()
	scale := cam.Scale()
	metrics := kindVisualMetrics(u.Kind)
	screenUnitWidth := worldTileSize * scale * metrics.widthTiles
	screenUnitHeight := worldTileSize * scale * metrics.heightTiles
	screenX := (u.Position.X - camPos.X) * scale
	screenY := (u.Position.Y - camPos.Y) * scale

	return geom.Rect{
		Min: geom.Point{
			X: screenX - screenUnitWidth/2,
			Y: screenY - screenUnitHeight*metrics.anchorY,
		},
		Max: geom.Point{
			X: screenX + screenUnitWidth/2,
			Y: screenY + screenUnitHeight*(1-metrics.anchorY),
		},
	}
}

type visualMetrics struct {
	widthTiles  float64
	heightTiles float64
	anchorY     float64
}

func kindVisualMetrics(kind Kind) visualMetrics {
	switch kind {
	case KindWall:
		return visualMetrics{widthTiles: 1.15, heightTiles: 1.25, anchorY: 0.95}
	case KindBarricade:
		return visualMetrics{widthTiles: 1.3, heightTiles: 0.85, anchorY: 0.86}
	case KindRunner, KindRunnerFocused:
		fallthrough
	default:
		return visualMetrics{widthTiles: 2.0, heightTiles: 2.0, anchorY: 0.85}
	}
}

func kindUsesSprite(kind Kind) bool {
	switch kind {
	case KindRunner, KindRunnerFocused:
		return true
	default:
		return false
	}
}

func (r *Renderer) drawStatic(screen *ebiten.Image, camPos geom.Point, scale, worldTileSize float64, u Unit) {
	metrics := kindVisualMetrics(u.Kind)
	screenX := (u.Position.X - camPos.X) * scale
	screenY := (u.Position.Y - camPos.Y) * scale
	width := worldTileSize * scale * metrics.widthTiles
	height := worldTileSize * scale * metrics.heightTiles
	rect := geom.Rect{
		Min: geom.Point{X: screenX - width/2, Y: screenY - height*metrics.anchorY},
		Max: geom.Point{X: screenX + width/2, Y: screenY + height*(1-metrics.anchorY)},
	}

	switch u.Kind {
	case KindWall:
		base := color.NRGBA{R: 106, G: 112, B: 122, A: 255}
		top := color.NRGBA{R: 147, G: 154, B: 167, A: 255}
		accent := color.NRGBA{R: 78, G: 82, B: 91, A: 255}
		r.drawFilledRect(screen, rect.Min.X, rect.Min.Y, rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y, base)
		r.drawFilledRect(screen, rect.Min.X, rect.Min.Y, rect.Max.X-rect.Min.X, (rect.Max.Y-rect.Min.Y)*0.22, top)
		segmentWidth := (rect.Max.X - rect.Min.X) / 5
		for index := 1; index <= 3; index++ {
			x := rect.Min.X + segmentWidth*float64(index)
			r.drawFilledRect(screen, x, rect.Min.Y, scale, rect.Max.Y-rect.Min.Y, accent)
		}
	case KindBarricade:
		wood := color.NRGBA{R: 123, G: 88, B: 56, A: 255}
		highlight := color.NRGBA{R: 162, G: 120, B: 78, A: 255}
		shadow := color.NRGBA{R: 89, G: 62, B: 39, A: 255}
		width := rect.Max.X - rect.Min.X
		height := rect.Max.Y - rect.Min.Y
		postWidth := math.Max(scale, width*0.12)
		r.drawFilledRect(screen, rect.Min.X+width*0.15, rect.Min.Y+height*0.1, postWidth, height*0.9, shadow)
		r.drawFilledRect(screen, rect.Max.X-width*0.15-postWidth, rect.Min.Y+height*0.1, postWidth, height*0.9, shadow)
		r.drawFilledRect(screen, rect.Min.X+width*0.1, rect.Min.Y+height*0.2, width*0.8, height*0.22, wood)
		r.drawFilledRect(screen, rect.Min.X+width*0.05, rect.Min.Y+height*0.48, width*0.75, height*0.18, highlight)
		r.drawFilledRect(screen, rect.Min.X+width*0.18, rect.Min.Y+height*0.68, width*0.72, height*0.16, wood)
	}
}

func (r *Renderer) drawFilledRect(screen *ebiten.Image, x, y, width, height float64, fill color.Color) {
	if width <= 0 || height <= 0 {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(fill)
	screen.DrawImage(r.solid, &op)
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
