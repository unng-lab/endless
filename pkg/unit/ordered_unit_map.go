package unit

// orderedUnitMap keeps units addressable by their stable ID while preserving insertion order
// for deterministic update, draw and collision passes. The manager uses this structure instead
// of a raw slice so lookups by ID stay O(1) without rebuilding a parallel index map.
type orderedUnitMap struct {
	order          []int64
	unitsByID      map[int64]Unit
	orderIndexByID map[int64]int
}

// newOrderedUnitMap allocates the ordered lookup once so the manager can add units without
// repeatedly growing the backing maps during scene bootstrap.
func newOrderedUnitMap(capacity int) *orderedUnitMap {
	if capacity < 0 {
		capacity = 0
	}

	return &orderedUnitMap{
		order:          make([]int64, 0, capacity),
		unitsByID:      make(map[int64]Unit, capacity),
		orderIndexByID: make(map[int64]int, capacity),
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
	if _, exists := m.orderIndexByID[unitID]; !exists {
		m.orderIndexByID[unitID] = len(m.order)
		m.order = append(m.order, unitID)
	}
	m.unitsByID[unitID] = unit
}

// Get resolves a unit by its stable ID in constant time.
func (m *orderedUnitMap) Get(unitID int64) (Unit, bool) {
	if m == nil || unitID == 0 {
		return nil, false
	}

	unit, ok := m.unitsByID[unitID]
	return unit, ok
}

// At returns the unit at the given insertion-order position. Worker batches use this method to
// keep their existing strided scheduling without depending on a raw slice.
func (m *orderedUnitMap) At(index int) (Unit, bool) {
	if m == nil || index < 0 || index >= len(m.order) {
		return nil, false
	}

	unitID := m.order[index]
	unit, ok := m.unitsByID[unitID]
	return unit, ok
}

// Range walks the ordered map in insertion order until the visitor returns false or every unit
// has been seen. This keeps manager loops concise while centralizing the order guarantee.
func (m *orderedUnitMap) Range(visitor func(Unit) bool) {
	if m == nil || visitor == nil {
		return
	}

	for _, unitID := range m.order {
		unit, ok := m.unitsByID[unitID]
		if !ok {
			continue
		}
		if !visitor(unit) {
			return
		}
	}
}

// Snapshot materializes the current ordered contents as a slice for APIs that still expect a
// linear collection, such as the renderer and some tests.
func (m *orderedUnitMap) Snapshot() []Unit {
	if m == nil || len(m.order) == 0 {
		return nil
	}

	units := make([]Unit, 0, len(m.order))
	m.Range(func(unit Unit) bool {
		units = append(units, unit)
		return true
	})
	return units
}
