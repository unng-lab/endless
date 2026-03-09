package unit

// orderedUnitMap keeps units addressable by their stable ID while preserving insertion order
// for deterministic update, draw and collision passes. The map points to entry objects and the
// ordered slice stores pointers to those same entries, so both views stay connected without
// duplicating the unit payload or resolving an intermediate ID on every ordered lookup.
type orderedUnitEntry struct {
	unit Unit
}

// orderedUnitMap keeps the pointer-linked ordered view and direct-ID lookup in one structure.
// The manager uses this structure instead of a raw slice so lookups by ID stay O(1) without
// rebuilding a parallel index map.
type orderedUnitMap struct {
	order   []*orderedUnitEntry
	entries map[int64]*orderedUnitEntry
}

// newOrderedUnitMap allocates the ordered lookup once so the manager can add units without
// repeatedly growing the backing maps during scene bootstrap.
func newOrderedUnitMap(capacity int) *orderedUnitMap {
	if capacity < 0 {
		capacity = 0
	}

	return &orderedUnitMap{
		order:   make([]*orderedUnitEntry, 0, capacity),
		entries: make(map[int64]*orderedUnitEntry, capacity),
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
	entry, exists := m.entries[unitID]
	if !exists {
		entry = &orderedUnitEntry{}
		m.entries[unitID] = entry
		m.order = append(m.order, entry)
	}
	entry.unit = unit
}

// Get resolves a unit by its stable ID in constant time.
func (m *orderedUnitMap) Get(unitID int64) (Unit, bool) {
	if m == nil || unitID == 0 {
		return nil, false
	}

	entry, ok := m.entries[unitID]
	if !ok || entry == nil || entry.unit == nil {
		return nil, false
	}

	return entry.unit, true
}

// At returns the unit at the given insertion-order position. Worker batches use this method to
// keep their existing strided scheduling without depending on a raw slice.
func (m *orderedUnitMap) At(index int) (Unit, bool) {
	if m == nil || index < 0 || index >= len(m.order) {
		return nil, false
	}

	entry := m.order[index]
	if entry == nil || entry.unit == nil {
		return nil, false
	}

	return entry.unit, true
}

// Range walks the ordered map in insertion order until the visitor returns false or every unit
// has been seen. This keeps manager loops concise while centralizing the order guarantee.
func (m *orderedUnitMap) Range(visitor func(Unit) bool) {
	if m == nil || visitor == nil {
		return
	}

	for _, entry := range m.order {
		if entry == nil || entry.unit == nil {
			continue
		}
		if !visitor(entry.unit) {
			return
		}
	}
}
