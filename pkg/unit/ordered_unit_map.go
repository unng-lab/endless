package unit

// orderedUnitMap keeps units addressable by their stable ID while preserving insertion order.
// The ordered slice stores Unit values directly so iteration walks a dense backing array instead
// of chasing an extra heap-allocated entry object per unit.
type orderedUnitMap struct {
	order []Unit
	index map[int64]int
}

// newOrderedUnitMap allocates the ordered lookup once so the manager can add units without
// repeatedly growing the backing maps during scene bootstrap.
func newOrderedUnitMap(capacity int) *orderedUnitMap {
	if capacity < 0 {
		capacity = 0
	}

	return &orderedUnitMap{
		order: make([]Unit, 0, capacity),
		index: make(map[int64]int, capacity),
	}
}

// Len reports how many non-deleted units are currently addressable through the ordered map.
func (m *orderedUnitMap) Len() int {
	if m == nil {
		return 0
	}

	count := 0
	for _, unit := range m.order {
		if orderedMapUnitDeleted(unit) {
			continue
		}
		count++
	}

	return count
}

// SlotsLen reports how many physical slots currently exist in insertion-order storage. Worker
// traversal uses this value because removed units may leave reusable holes inside the slice.
func (m *orderedUnitMap) SlotsLen() int {
	if m == nil {
		return 0
	}

	return len(m.order)
}

// HasPendingRemovalWork reports whether the backing slice still contains a deleted unit whose
// manager-side cleanup has not yet happened. Update uses this to flush tile state even when no
// live unit remains in storage.
func (m *orderedUnitMap) HasPendingRemovalWork() bool {
	if m == nil {
		return false
	}

	for _, unit := range m.order {
		if unit == nil || !unit.Base().PendingRemoval() || unit.Base().RemovalHandled() {
			continue
		}
		return true
	}

	return false
}

// Set inserts a unit into the first reusable deleted slot or appends it at the end.
// Replacements for an already known UnitID keep their established slot so external ordering
// remains stable even after older dead entries stay resident inside the backing slice.
func (m *orderedUnitMap) Set(unit Unit) {
	if m == nil || unit == nil || unit.UnitID() == 0 {
		return
	}

	unitID := unit.UnitID()
	if index, exists := m.index[unitID]; exists {
		m.order[index] = unit
		return
	}

	for index, current := range m.order {
		if !orderedMapUnitDeleted(current) {
			continue
		}

		if current != nil {
			delete(m.index, current.UnitID())
		}
		m.order[index] = unit
		m.index[unitID] = index
		return
	}

	m.index[unitID] = len(m.order)
	m.order = append(m.order, unit)
}

// Get resolves a unit by its stable ID in constant time.
func (m *orderedUnitMap) Get(unitID int64) (Unit, bool) {
	if m == nil || unitID == 0 {
		return nil, false
	}

	index, ok := m.index[unitID]
	if !ok || index < 0 || index >= len(m.order) {
		return nil, false
	}

	unit := m.order[index]
	if orderedMapUnitDeleted(unit) {
		return nil, false
	}

	return unit, true
}

// At returns the live unit at the given insertion-order slot while hiding entries that were
// already marked deleted and only await overwrite by some future insertion.
func (m *orderedUnitMap) At(index int) (Unit, bool) {
	if m == nil || index < 0 || index >= len(m.order) {
		return nil, false
	}

	unit := m.order[index]
	if orderedMapUnitDeleted(unit) {
		return nil, false
	}

	return unit, true
}

// SlotAt returns the raw unit stored in the physical slot, including already deleted entries.
// Manager worker traversal uses this lower-level accessor so it may still flush tile state for
// units that have just been marked deleted but not yet overwritten by a later insertion.
func (m *orderedUnitMap) SlotAt(index int) (Unit, bool) {
	if m == nil || index < 0 || index >= len(m.order) {
		return nil, false
	}

	unit := m.order[index]
	if unit == nil || unit.Base().RemovalHandled() {
		return nil, false
	}

	return unit, true
}

// Range walks the ordered map in insertion order until the visitor returns false or every unit
// has been seen. This keeps manager loops concise while centralizing the order guarantee.
func (m *orderedUnitMap) Range(visitor func(Unit) bool) {
	if m == nil || visitor == nil {
		return
	}

	for _, unit := range m.order {
		if orderedMapUnitDeleted(unit) {
			continue
		}
		if !visitor(unit) {
			return
		}
	}
}

func orderedMapUnitDeleted(unit Unit) bool {
	return unit == nil || unit.Base().PendingRemoval()
}
