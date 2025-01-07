package dstar

import (
	"testing"

	"github/unng-lab/madfarmer/internal/geom"
)

func TestNode_to(t *testing.T) {
	tests := []struct {
		start, target Node
		expected      byte
	}{
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{0, -1}}, DirUp},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{0, 1}}, DirDown},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{-1, 0}}, DirLeft},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{1, 0}}, DirRight},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{-1, -1}}, DirUpLeft},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{-1, 1}}, DirDownLeft},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{1, -1}}, DirUpRight},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{1, 1}}, DirDownRight},
		{Node{Position: geom.Point{0, 0}}, Node{Position: geom.Point{2, 2}}, DirNone},
	}

	for _, test := range tests {
		result := test.start.to(test.target)
		if result != test.expected {
			t.Errorf("Expected %v, got %v", test.expected, result)
		}
	}
}
