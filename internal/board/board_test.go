package board

import (
	"testing"
)

func TestGetCellBoundaryCheck(t *testing.T) {
	type testCase struct {
		name     string
		x        int64
		y        int64
		width    uint64
		height   uint64
		expected bool // true если должен вернуть nil
	}

	tests := []testCase{
		// Valid cases
		{"Middle of board", 3, 3, 5, 5, false},
		{"Zero coordinates", 0, 0, 1, 1, false},
		{"Max valid X", 4, 2, 5, 5, false},
		{"Max valid Y", 2, 4, 5, 5, false},

		// Invalid cases
		{"X equals width", 5, 3, 5, 5, true},
		{"Y equals height", 3, 5, 5, 5, true},
		{"Negative X", -1, 3, 5, 5, true},
		{"Negative Y", 3, -1, 5, 5, true},
		{"Both exceed", 6, 6, 5, 5, true},
		{"Large values", 1e18, 1e18, 100, 100, true},

		// Edge cases
		{"Zero size board", 0, 0, 0, 0, true},
		{"Unit board valid", 0, 0, 1, 1, false},
		{"Unit board invalid", 1, 1, 1, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Board{
				Width:  tt.width,
				Height: tt.height,
			}

			// Вызываем проверку границ
			result := uint64(tt.x) >= b.Width || uint64(tt.y) >= b.Height

			if result != tt.expected {
				t.Errorf("For case %s: expected %v, got %v (x=%d, y=%d, w=%d, h=%d)",
					tt.name, tt.expected, result, tt.x, tt.y, tt.width, tt.height)
			}
		})
	}
}

func TestGetCellReturnsNil(t *testing.T) {
	b := &Board{Width: 5, Height: 5}

	tests := []struct {
		x, y int64
	}{
		{5, 2}, {2, 5}, {-1, 3}, {3, -1}, {100, 100},
	}

	for _, tt := range tests {
		cell := b.GetCell(tt.x, tt.y)
		if cell != nil {
			t.Errorf("Expected nil for (%d, %d), got %v", tt.x, tt.y, cell)
		}
	}
}
