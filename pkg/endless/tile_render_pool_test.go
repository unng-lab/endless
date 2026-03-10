package endless

import (
	"errors"
	"image"
	"testing"
)

func TestTileCoordinatesForVisibleIndexFollowsRowMajorOrder(t *testing.T) {
	visible := image.Rect(3, 5, 6, 7)
	want := [][2]int{
		{3, 5},
		{4, 5},
		{5, 5},
		{3, 6},
		{4, 6},
		{5, 6},
	}

	for index, expected := range want {
		tileX, tileY := tileCoordinatesForVisibleIndex(visible, visible.Dx(), index)
		if tileX != expected[0] || tileY != expected[1] {
			t.Fatalf("tileCoordinatesForVisibleIndex(..., %d) = (%d, %d), want (%d, %d)", index, tileX, tileY, expected[0], expected[1])
		}
	}
}

func TestTileRenderErrorSinkKeepsFirstError(t *testing.T) {
	firstErr := errors.New("first")
	secondErr := errors.New("second")
	var sink tileRenderErrorSink

	sink.store(firstErr)
	sink.store(secondErr)

	if got := sink.load(); !errors.Is(got, firstErr) {
		t.Fatalf("load() = %v, want first error %v", got, firstErr)
	}
}
