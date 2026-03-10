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

// Len reports how many units are currently stored in insertion order.
func (m *orderedUnitMap) Len() int {
	if m == nil {
		return 0
	}

	return len(m.order)
}

// Set inserts a unit at the end of the ordered map or replaces the stored value for an already
// known ID without disturbing the established iteration order.
func (m *orderedUnitMap) Set(unit Unit) {
	if m == nil || unit == nil || unit.UnitID() == 0 {
		return
	}

	unitID := unit.UnitID()
	if index, exists := m.index[unitID]; exists {
		m.order[index] = unit
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
	if unit == nil {
		return nil, false
	}

	return unit, true
}

// At returns the unit at the given insertion-order position. Worker batches use this method to
// keep their existing strided scheduling without depending on a raw slice.
func (m *orderedUnitMap) At(index int) (Unit, bool) {
	if m == nil || index < 0 || index >= len(m.order) {
		return nil, false
	}

	unit := m.order[index]
	if unit == nil {
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
		if unit == nil {
			continue
		}
		if !visitor(unit) {
			return
		}
	}
}
