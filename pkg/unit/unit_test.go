package unit

import (
	"testing"

	"github.com/unng-lab/endless/pkg/geom"
)

func TestUnitFollowsPath(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	u.Update(1)

	if u.Position != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 40, Y: 8})
	}
	if u.HasPath() {
		t.Fatalf("expected path to be consumed, got %d waypoints", u.PathLen())
	}
}
