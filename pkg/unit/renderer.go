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
	impacts []impactEffect,
) error {
	if len(units) == 0 && len(impacts) == 0 {
		return nil
	}

	camPos := cam.Position()
	scale := cam.Scale()

	for _, current := range units {
		if current == nil || !current.Base().OnScreen {
			continue
		}

		switch body := current.(type) {
		case *NonStaticUnit:
			if !kindUsesSprite(body.UnitKind()) {
				r.drawStatic(screen, camPos, scale, worldTileSize, body.UnitKind(), body.Base().RenderPosition())
			} else {
				if err := r.drawAnimatedUnit(screen, camPos, scale, worldTileSize, quality, body); err != nil {
					return err
				}
			}
			r.drawHealthBar(screen, bodyScreenRect(cam, worldTileSize, body.UnitKind(), body.Base().RenderPosition()), body.CurrentHealth(), body.MaxHealthValue())
		case *StaticUnit:
			r.drawStatic(screen, camPos, scale, worldTileSize, body.UnitKind(), body.Base().RenderPosition())
			r.drawHealthBar(screen, bodyScreenRect(cam, worldTileSize, body.UnitKind(), body.Base().RenderPosition()), body.CurrentHealth(), body.MaxHealthValue())
		case *Projectile:
			r.drawProjectile(screen, camPos, scale, body)
		default:
			return fmt.Errorf("unsupported unit type %T", current)
		}
	}

	for _, effect := range impacts {
		r.drawImpact(screen, camPos, scale, effect)
	}

	return nil
}

func (r *Renderer) drawAnimatedUnit(
	screen *ebiten.Image,
	camPos geom.Point,
	scale float64,
	worldTileSize float64,
	quality assets.Quality,
	body *NonStaticUnit,
) error {
	metrics := kindVisualMetrics(body.UnitKind())
	screenUnitWidth := worldTileSize * scale * metrics.widthTiles
	screenUnitHeight := worldTileSize * scale * metrics.heightTiles

	frame, err := r.frameImage(body.UnitKind(), body.Frame(), quality)
	if err != nil {
		return err
	}

	frameBounds := frame.Bounds()
	frameScale := screenUnitWidth / float64(frameBounds.Dx())
	renderPos := body.Base().RenderPosition()
	screenX := (renderPos.X - camPos.X) * scale
	screenY := (renderPos.Y - camPos.Y) * scale

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(frameScale, frameScale)
	op.GeoM.Translate(screenX-screenUnitWidth/2, screenY-screenUnitHeight*metrics.anchorY)
	screen.DrawImage(frame, &op)
	return nil
}

func ScreenRect(cam *camera.Camera, worldTileSize float64, unit Unit) geom.Rect {
	return bodyScreenRect(cam, worldTileSize, unit.UnitKind(), unit.Base().RenderPosition())
}

func bodyScreenRect(cam *camera.Camera, worldTileSize float64, kind Kind, renderPos geom.Point) geom.Rect {
	camPos := cam.Position()
	scale := cam.Scale()
	metrics := kindVisualMetrics(kind)
	screenUnitWidth := worldTileSize * scale * metrics.widthTiles
	screenUnitHeight := worldTileSize * scale * metrics.heightTiles
	screenX := (renderPos.X - camPos.X) * scale
	screenY := (renderPos.Y - camPos.Y) * scale

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

func unitScreenRect(cam *camera.Camera, worldTileSize float64, unit Unit) (geom.Rect, bool) {
	switch body := unit.(type) {
	case *NonStaticUnit, *StaticUnit:
		return ScreenRect(cam, worldTileSize, unit), true
	case *Projectile:
		return ProjectileScreenRect(cam, body), true
	default:
		return geom.Rect{}, false
	}
}

// ProjectileScreenRect converts the projectile's current interpolated world position into a
// screen-space rectangle large enough for both its core and glow. Using the glow size here
// keeps visibility consistent with the detailed rendering footprint.
func ProjectileScreenRect(cam *camera.Camera, shot *Projectile) geom.Rect {
	camPos := cam.Position()
	scale := cam.Scale()
	renderPos := shot.Base().RenderPosition()
	glowRadius := math.Max(1, shot.Radius*scale*0.9)
	screenX := (renderPos.X - camPos.X) * scale
	screenY := (renderPos.Y - camPos.Y) * scale

	return geom.Rect{
		Min: geom.Point{X: screenX - glowRadius, Y: screenY - glowRadius},
		Max: geom.Point{X: screenX + glowRadius, Y: screenY + glowRadius},
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

func (r *Renderer) drawStatic(screen *ebiten.Image, camPos geom.Point, scale, worldTileSize float64, kind Kind, renderPos geom.Point) {
	metrics := kindVisualMetrics(kind)
	screenX := (renderPos.X - camPos.X) * scale
	screenY := (renderPos.Y - camPos.Y) * scale
	width := worldTileSize * scale * metrics.widthTiles
	height := worldTileSize * scale * metrics.heightTiles
	rect := geom.Rect{
		Min: geom.Point{X: screenX - width/2, Y: screenY - height*metrics.anchorY},
		Max: geom.Point{X: screenX + width/2, Y: screenY + height*(1-metrics.anchorY)},
	}

	switch kind {
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

func (r *Renderer) drawProjectile(screen *ebiten.Image, camPos geom.Point, scale float64, shot *Projectile) {
	size := math.Max(2, shot.Radius*2*scale)
	renderPos := shot.Base().RenderPosition()
	screenX := (renderPos.X - camPos.X) * scale
	screenY := (renderPos.Y - camPos.Y) * scale
	glowSize := size * 1.8

	r.drawFilledRect(screen, screenX-glowSize/2, screenY-glowSize/2, glowSize, glowSize, color.NRGBA{R: 255, G: 176, B: 64, A: 110})
	r.drawFilledRect(screen, screenX-size/2, screenY-size/2, size, size, color.NRGBA{R: 255, G: 226, B: 168, A: 255})
}

func (r *Renderer) drawImpact(screen *ebiten.Image, camPos geom.Point, scale float64, effect impactEffect) {
	if effect.Duration <= 0 {
		return
	}

	progress := geom.ClampFloat(effect.Age/effect.Duration, 0, 1)
	alpha := uint8(math.Round((1 - progress) * 220))
	size := effect.Radius * 2 * scale * (1 + progress*0.8)
	screenX := (effect.Position.X - camPos.X) * scale
	screenY := (effect.Position.Y - camPos.Y) * scale

	r.drawFilledRect(screen, screenX-size/2, screenY-size/2, size, size, color.NRGBA{R: 255, G: 187, B: 89, A: alpha})
	r.drawFilledRect(screen, screenX-size*0.28, screenY-size*0.28, size*0.56, size*0.56, color.NRGBA{R: 255, G: 240, B: 196, A: alpha})
}

func (r *Renderer) drawHealthBar(screen *ebiten.Image, rect geom.Rect, health, maxHealth int) {
	if maxHealth <= 0 || health >= maxHealth {
		return
	}

	width := rect.Max.X - rect.Min.X
	height := math.Max(3, math.Round(width*0.08))
	top := rect.Min.Y - height - math.Max(3, height)
	ratio := geom.ClampFloat(float64(health)/float64(maxHealth), 0, 1)
	fillWidth := width * ratio
	fillColor := color.NRGBA{
		R: uint8(math.Round(255 * (1 - ratio))),
		G: uint8(math.Round(208 * ratio)),
		B: 72,
		A: 255,
	}

	r.drawFilledRect(screen, rect.Min.X, top, width, height, color.NRGBA{R: 36, G: 24, B: 24, A: 220})
	r.drawFilledRect(screen, rect.Min.X, top, fillWidth, height, fillColor)
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
