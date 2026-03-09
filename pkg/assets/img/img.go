package img

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	_ "image/jpeg"
	_ "image/png"
)

var (
	//go:embed raw/*
	images embed.FS

	mu      sync.RWMutex
	storage = make(map[string]*ebiten.Image)
)

func Img(name string, w, h uint64) (*ebiten.Image, error) {
	assetName, err := normalizeAssetName(name)
	if err != nil {
		return nil, err
	}
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("asset %q dimensions must be positive", assetName)
	}

	key := cacheKey(assetName, w, h)

	mu.RLock()
	if cached := storage[key]; cached != nil {
		mu.RUnlock()
		return cached, nil
	}
	mu.RUnlock()

	cachePath, err := cacheFilePath(assetName, w, h)
	if err != nil {
		return nil, err
	}

	if cached, err := loadImage(cachePath); err == nil {
		store(key, cached)
		return cached, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("load cached asset %q: %w", assetName, err)
	}

	resized, err := buildResizedAsset(assetName, int(w), int(h))
	if err != nil {
		return nil, err
	}
	if err := persistResizedAsset(cachePath, resized); err != nil {
		return nil, err
	}

	cached := ebiten.NewImageFromImage(resized)
	store(key, cached)
	return cached, nil
}

func buildResizedAsset(assetName string, width, height int) (image.Image, error) {
	source, err := images.ReadFile(path.Join("raw", assetName))
	if err != nil {
		return nil, fmt.Errorf("read embedded asset %q: %w", assetName, err)
	}

	src, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("decode embedded asset %q: %w", assetName, err)
	}

	if src.Bounds().Dx() == width && src.Bounds().Dy() == height {
		return src, nil
	}

	return resizeNearest(src, width, height), nil
}

func persistResizedAsset(filePath string, img image.Image) (err error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create asset cache directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create cached asset %q: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close cached asset %q: %w", filePath, closeErr)
		}
	}()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encode cached asset %q: %w", filePath, err)
	}

	return nil
}

func loadImage(filePath string) (_ *ebiten.Image, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close cached asset %q: %w", filePath, closeErr)
		}
	}()

	decoded, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(decoded), nil
}

func cacheFilePath(assetName string, w, h uint64) (string, error) {
	root, err := cacheRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(root, assetDirName(assetName), dimensionFileName(w, h)), nil
}

func cacheRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("ENDLESS_ASSET_CACHE_DIR")); root != "" {
		return root, nil
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve package directory for asset cache")
	}

	return filepath.Join(filepath.Dir(file), "cache"), nil
}

func normalizeAssetName(name string) (string, error) {
	cleaned := path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	switch cleaned {
	case "", ".", "/":
		return "", fmt.Errorf("asset name %q is invalid", name)
	}

	return cleaned, nil
}

func cacheKey(assetName string, w, h uint64) string {
	return path.Join(assetName, dimensionFileName(w, h))
}

func assetDirName(assetName string) string {
	return strings.TrimSuffix(assetName, path.Ext(assetName))
}

func dimensionFileName(w, h uint64) string {
	return fmt.Sprintf("%dx%d.png", w, h)
}

func resizeNearest(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	for y := range height {
		srcY := srcBounds.Min.Y + y*srcHeight/height
		for x := range width {
			srcX := srcBounds.Min.X + x*srcWidth/width
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	return dst
}

func store(key string, img *ebiten.Image) {
	mu.Lock()
	defer mu.Unlock()
	storage[key] = img
}
