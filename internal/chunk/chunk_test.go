// chunk/chunk_test.go
package chunk

import (
	"testing"

	"github.com/unng-lab/endless/internal/geom"
)

func TestGrid(t *testing.T) {
	g := NewGrid(10, 10)

	// Test boundaries
	if g.InBounds(-1, 0) {
		t.Error("InBounds failed: (-1,0) should be out of bounds")
	}
	if g.InBounds(0, -1) {
		t.Error("InBounds failed: (0,-1) should be out of bounds")
	}
	if !g.InBounds(9, 9) {
		t.Error("InBounds failed: (9,9) should be in bounds")
	}

	// Test blocking
	g.SetBlocked(5, 5, true)
	if !g.Blocked(5, 5) {
		t.Error("Blocked failed: (5,5) should be blocked")
	}

	g.SetBlocked(5, 5, false)
	if g.Blocked(5, 5) {
		t.Error("Blocked failed: (5,5) should not be blocked")
	}
}

func TestChunkManager(t *testing.T) {
	cm := NewChunkManager()

	// Test loading a chunk
	chunkID := geom.ChunkID{1, 2}
	c := cm.EnsureLoaded(chunkID)
	if c == nil {
		t.Fatal("EnsureLoaded failed: returned nil chunk")
	}
	if c.ID != chunkID {
		t.Errorf("EnsureLoaded failed: expected chunk ID %v, got %v", chunkID, c.ID)
	}

	// Test getting the same chunk
	c2 := cm.Get(chunkID)
	if c2 == nil {
		t.Fatal("Get failed: returned nil chunk")
	}
	if c != c2 {
		t.Error("Get failed: should return the same chunk instance")
	}
}
