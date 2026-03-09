package unit

import "sync"

// TileStack stores the deterministic unit order for one logical tile. The stack keeps only
// stable unit identifiers so membership survives manager-side storage rebuilds without tying
// tile occupancy to any particular in-memory collection layout.
type TileStack struct {
	mu      sync.RWMutex
	unitIDs []int64
}

// AddUnit registers the unit in the tile-local render and interaction order if it is not
// already present. Deduplication here prevents repeated enter calls from corrupting the
// per-tile stack when reconciliation sees the same unit multiple times.
func (s *TileStack) AddUnit(unitID int64) {
	if s == nil || unitID == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, currentID := range s.unitIDs {
		if currentID == unitID {
			return
		}
	}

	s.unitIDs = append(s.unitIDs, unitID)
}

// RemoveUnit unregisters the unit from the tile-local order while preserving the relative
// order of all remaining units on that tile.
func (s *TileStack) RemoveUnit(unitID int64) {
	if s == nil || unitID == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for index, currentID := range s.unitIDs {
		if currentID != unitID {
			continue
		}

		copy(s.unitIDs[index:], s.unitIDs[index+1:])
		s.unitIDs = s.unitIDs[:len(s.unitIDs)-1]
		return
	}
}

// UnitIDs returns a snapshot so callers can iterate without holding the tile lock across
// manager lookups, rendering or combat callbacks.
func (s *TileStack) UnitIDs() []int64 {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]int64(nil), s.unitIDs...)
}

// Empty reports whether the tile stack currently has no registered units and can therefore be
// removed from the manager's sparse tile map.
func (s *TileStack) Empty() bool {
	if s == nil {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.unitIDs) == 0
}
