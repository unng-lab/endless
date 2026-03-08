package assets

import (
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets/img"
)

type Quality int

const (
	QualityLow Quality = iota
	QualityMedium
	QualityHigh
)

const tilesPerRow = 16

type AtlasConfig struct {
	FileName string
	TileSize int
}

type TileAtlas struct {
	mu       sync.Mutex
	configs  map[Quality]AtlasConfig
	atlases  map[Quality]*ebiten.Image
	tileRefs map[Quality]map[int]*ebiten.Image
}

func NewTileAtlas() *TileAtlas {
	return &TileAtlas{
		configs: map[Quality]AtlasConfig{
			QualityLow: {
				FileName: "small.png",
				TileSize: 16,
			},
			QualityMedium: {
				FileName: "normal.png",
				TileSize: 32,
			},
			QualityHigh: {
				FileName: "normal.png",
				TileSize: 64,
			},
		},
		atlases:  make(map[Quality]*ebiten.Image),
		tileRefs: make(map[Quality]map[int]*ebiten.Image),
	}
}

func (a *TileAtlas) QualityForScreenSize(screenTileSize float64) Quality {
	switch {
	case screenTileSize >= 96:
		return QualityHigh
	case screenTileSize >= 48:
		return QualityMedium
	default:
		return QualityLow
	}
}

func (a *TileAtlas) TileSize(quality Quality) int {
	cfg, ok := a.configs[quality]
	if !ok {
		return 0
	}

	return cfg.TileSize
}

func (a *TileAtlas) TileImage(index int, quality Quality) (*ebiten.Image, error) {
	if index < 0 || index >= tilesPerRow*tilesPerRow {
		return nil, fmt.Errorf("tile index %d is out of range", index)
	}

	atlas, err := a.ensureAtlas(quality)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.tileRefs[quality]; !ok {
		a.tileRefs[quality] = make(map[int]*ebiten.Image)
	}
	if tile := a.tileRefs[quality][index]; tile != nil {
		return tile, nil
	}

	tileSize := a.configs[quality].TileSize
	x := (index % tilesPerRow) * tileSize
	y := (index / tilesPerRow) * tileSize

	sub := atlas.SubImage(image.Rect(x, y, x+tileSize, y+tileSize)).(*ebiten.Image)
	a.tileRefs[quality][index] = sub
	return sub, nil
}

func (a *TileAtlas) ensureAtlas(quality Quality) (*ebiten.Image, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if atlas := a.atlases[quality]; atlas != nil {
		return atlas, nil
	}

	cfg, ok := a.configs[quality]
	if !ok {
		return nil, fmt.Errorf("no atlas config for quality %d", quality)
	}

	size := cfg.TileSize * tilesPerRow
	atlas, err := img.Img(cfg.FileName, uint64(size), uint64(size))
	if err != nil {
		return nil, fmt.Errorf("load atlas %q: %w", cfg.FileName, err)
	}

	a.atlases[quality] = atlas
	return atlas, nil
}
