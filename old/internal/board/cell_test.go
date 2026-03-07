package board

import (
	"math"
	"testing"

	"github.com/unng-lab/endless/internal/geom"
)

func TestCellRemoveUnitAllowsReAdd(t *testing.T) {
	cell := NewCell(CellTypeGround, 16, geom.Point{})

	if err := cell.AddUnit(1, 0, math.Inf(1)); err != nil {
		t.Fatalf("AddUnit failed: %v", err)
	}

	if err := cell.RemoveUnit(1); err != nil {
		t.Fatalf("RemoveUnit failed: %v", err)
	}

	if err := cell.AddUnit(1, 0, math.Inf(1)); err != nil {
		t.Fatalf("AddUnit after removal failed: %v", err)
	}
}

func TestCellCostResetsAfterRemoval(t *testing.T) {
	cell := NewCell(CellTypeGround, 16, geom.Point{})

	base := cell.BaseCost

	if err := cell.AddUnit(1, 0, math.Inf(1)); err != nil {
		t.Fatalf("AddUnit failed: %v", err)
	}

	if err := cell.RemoveUnit(1); err != nil {
		t.Fatalf("RemoveUnit failed: %v", err)
	}

	if got := cell.Cost; got != base {
		t.Fatalf("expected cost to reset to %v, got %v", base, got)
	}
}

func TestCellRecalculateCostWithFiniteUnits(t *testing.T) {
	cell := NewCell(CellTypeGround, 16, geom.Point{})

	if err := cell.AddUnit(1, 0, 10); err != nil {
		t.Fatalf("AddUnit failed: %v", err)
	}

	if err := cell.AddUnit(2, 0, 20); err != nil {
		t.Fatalf("AddUnit second failed: %v", err)
	}

	if err := cell.RemoveUnit(1); err != nil {
		t.Fatalf("RemoveUnit failed: %v", err)
	}

	expected := cell.BaseCost + 20
	if got := cell.Cost; got != expected {
		t.Fatalf("expected cost %v, got %v", expected, got)
	}
}
