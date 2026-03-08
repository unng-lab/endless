package img

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAssetName(t *testing.T) {
	t.Parallel()

	got, err := normalizeAssetName(`nested\runner.png`)
	if err != nil {
		t.Fatalf("normalizeAssetName returned error: %v", err)
	}
	if got != "runner.png" {
		t.Fatalf("normalizeAssetName = %q, want %q", got, "runner.png")
	}
}

func TestDimensionFileName(t *testing.T) {
	t.Parallel()

	if got := dimensionFileName(128, 64); got != "128x64.png" {
		t.Fatalf("dimensionFileName = %q, want %q", got, "128x64.png")
	}
}

func TestAssetDirName(t *testing.T) {
	t.Parallel()

	if got := assetDirName("normal.png"); got != "normal" {
		t.Fatalf("assetDirName = %q, want %q", got, "normal")
	}
}

func TestCacheKeyUsesStableSeparators(t *testing.T) {
	t.Parallel()

	if got := cacheKey("normal.png", 512, 512); got != "normal.png/512x512.png" {
		t.Fatalf("cacheKey = %q, want %q", got, "normal.png/512x512.png")
	}
}

func TestCacheFilePathUsesPerAssetDirectory(t *testing.T) {
	t.Setenv("ENDLESS_ASSET_CACHE_DIR", t.TempDir())

	got, err := cacheFilePath("normal.png", 512, 512)
	if err != nil {
		t.Fatalf("cacheFilePath returned error: %v", err)
	}

	want := filepath.Join(os.Getenv("ENDLESS_ASSET_CACHE_DIR"), "normal", "512x512.png")
	if got != want {
		t.Fatalf("cacheFilePath = %q, want %q", got, want)
	}
}
