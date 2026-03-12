package unit

import (
	"image"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

func (m *Manager) tileSpeedMultiplierAt(position geom.Point) float64 {
	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	if !m.world.InBounds(tileX, tileY) {
		return 0
	}

	return m.world.TileType(tileX, tileY).SpeedMultiplier()
}

func (m *Manager) ensureTileStackLocked(key tileKey) *TileStack {
	stack, ok := m.tileStacks[key]
	if ok {
		return stack
	}

	stack = &TileStack{}
	m.tileStacks[key] = stack
	return stack
}

// registerUnitInCurrentTile binds a unit to the stack of the tile that matches its current
// logical position. The helper is used for initial seeding and for units added at runtime.
func (m *Manager) registerUnitInCurrentTile(unit Unit) {
	if !unitUsesTileStack(unit) {
		return
	}

	key := m.tileKeyForUnit(unit)
	m.tileRegistryMu.Lock()
	stack := m.ensureTileStackLocked(key)
	unit.EnterTile(stack)
	m.registeredTiles[unit.UnitID()] = key
	m.tileRegistryMu.Unlock()
}

// bindUnitRuntimeDependencies installs manager-owned resolvers once at registration time so
// the hot Tick path can stay on the minimal tick-only contract requested for units.
func (m *Manager) bindUnitRuntimeDependencies(unit Unit) {
	if m == nil || unit == nil {
		return
	}

	body, ok := unit.(*NonStaticUnit)
	if !ok {
		return
	}

	body.SetSpeedMultiplierLookup(m.tileSpeedMultiplierAt)
	body.SetProjectileBuilder(func(owner *NonStaticUnit, direction geom.Point) (*Projectile, error) {
		return newProjectile(owner, direction, m.world)
	})
}

func (m *Manager) unregisterUnitFromTile(unit Unit, key tileKey) {
	m.tileRegistryMu.Lock()
	defer m.tileRegistryMu.Unlock()

	m.unregisterUnitFromTileLocked(unit, key)
}

func (m *Manager) unregisterUnitFromTileLocked(unit Unit, key tileKey) {
	stack := m.tileStacks[key]
	if stack == nil {
		return
	}

	unit.LeaveTile(stack)
	m.dropEmptyTileStackLocked(key, stack)
	delete(m.registeredTiles, unit.UnitID())
}

func (m *Manager) dropEmptyTileStackLocked(key tileKey, stack *TileStack) {
	if stack == nil || !stack.Empty() {
		return
	}

	delete(m.tileStacks, key)
}

func (m *Manager) tileKeyForUnit(unit Unit) tileKey {
	tileX, tileY := unit.Base().TilePosition(m.world.TileSize())
	return tileKey{x: tileX, y: tileY}
}

// moveUnitToTile applies the explicit leave/enter sequence at the moment the logical tile
// changes. TileStack methods serialize membership edits per tile, while the registry mutex
// protects the sparse tile map and the unit-to-tile lookup table.
func (m *Manager) moveUnitToTile(unit Unit, from tileKey, to tileKey) {
	if unit == nil || from == to || !unitUsesTileStack(unit) {
		return
	}

	var currentStack *TileStack
	m.tileRegistryMu.Lock()
	if registeredKey, isRegistered := m.registeredTiles[unit.UnitID()]; isRegistered {
		from = registeredKey
	}
	if from == to {
		m.tileRegistryMu.Unlock()
		return
	}

	if previousStack := m.tileStacks[from]; previousStack != nil {
		unit.LeaveTile(previousStack)
		m.dropEmptyTileStackLocked(from, previousStack)
	}

	currentStack = m.ensureTileStackLocked(to)
	unit.EnterTile(currentStack)
	m.registeredTiles[unit.UnitID()] = to
	m.tileRegistryMu.Unlock()

	body, ok := unit.(tileEntryReactiveUnit)
	if !ok {
		return
	}

	body.ReactToEnteredTile(m, currentStack)
}

func (m *Manager) stackAtWorldPoint(position geom.Point) *TileStack {
	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	return m.tileStackAtKey(tileKey{x: tileX, y: tileY})
}

func (m *Manager) unitsFromStack(stack *TileStack) []Unit {
	if stack == nil {
		return nil
	}

	unitIDs := stack.UnitIDs()
	units := make([]Unit, 0, len(unitIDs))
	for _, unitID := range unitIDs {
		current, ok := m.unitByID(unitID)
		if !ok || !unitUsesTileStack(current) {
			continue
		}
		units = append(units, current)
	}

	return units
}

// selectableUnitsFromStack keeps click selection tied to gameplay bodies the player may
// meaningfully inspect or command even though the same tile stack can also contain projectiles
// for rendering and collision purposes.
func (m *Manager) selectableUnitsFromStack(stack *TileStack) []Unit {
	units := m.unitsFromStack(stack)
	if len(units) == 0 {
		return nil
	}

	selectable := units[:0]
	for _, current := range units {
		if !current.Selectable() {
			continue
		}
		selectable = append(selectable, current)
	}

	return selectable
}

// visibleTileUnits collects every tile-registered unit by iterating the already computed
// visible tile rectangle in row-major order. Each tile contributes units in its local
// TileStack order so draw order remains deterministic both across tiles and within one tile.
// When requested, the same pass also advances draw-only visible state exactly once for the
// current game tick so visible interpolation stays coupled to the tile traversal.
func (m *Manager) visibleTileUnits(visible image.Rectangle, updateVisibleUnits bool) []Unit {
	if m == nil || visible.Empty() {
		return nil
	}

	visibleUnits := make([]Unit, 0)
	for tileY := visible.Min.Y; tileY < visible.Max.Y; tileY++ {
		for tileX := visible.Min.X; tileX < visible.Max.X; tileX++ {
			stack := m.tileStackAtKey(tileKey{x: tileX, y: tileY})
			if stack == nil {
				continue
			}

			for _, current := range m.unitsFromStack(stack) {
				if updateVisibleUnits {
					m.updateVisibleUnit(current)
				}
				visibleUnits = append(visibleUnits, current)
			}
		}
	}

	return visibleUnits
}

func (m *Manager) updateVisibleUnit(unit Unit) {
	if unit == nil {
		return
	}

	body, ok := unit.(visibleUpdatingUnit)
	if !ok {
		return
	}

	body.UpdateVisible(m.lastGameTick)
}

// firstProjectileOccupant resolves hits through the tile stack the projectile has just entered.
// Iterating the stack snapshot keeps the old "check every unit in that tile" behavior while
// moving the broad-phase lookup away from a full scan over every unit in the scene.
func (m *Manager) firstProjectileOccupant(stack *TileStack, ownerID int64) (Unit, bool) {
	if stack == nil {
		return nil, false
	}

	for _, unitID := range stack.UnitIDs() {
		if unitID == ownerID {
			continue
		}

		currentUnit, ok := m.unitByID(unitID)
		if !ok {
			continue
		}

		if !currentUnit.Alive() || !currentUnit.Selectable() {
			continue
		}

		return currentUnit, true
	}

	return nil, false
}

// unitUsesTileStack centralizes which bodies participate in tile-local ordering. Projectiles
// now opt in here as non-selectable stack members so rendering and collision can reuse the
// same per-tile structure without making shots clickable in the UI.
func unitUsesTileStack(unit Unit) bool {
	return unit != nil && unit.Alive() && (unit.Selectable() || unit.UnitKind() == KindProjectile)
}

func (m *Manager) tileStackAtKey(key tileKey) *TileStack {
	m.tileRegistryMu.RLock()
	stack := m.tileStacks[key]
	m.tileRegistryMu.RUnlock()
	return stack
}

// tileBlockedForMovement reports whether the specified tile currently contains a live unit that
// blocks movement and is not the ignored unit. Movement pathfinding calls this to keep routed
// destinations from crossing static cover and other blocking world bodies.
func (m *Manager) tileBlockedForMovement(tileX, tileY int, ignoredUnitID int64) bool {
	if m == nil {
		return false
	}

	stack := m.tileStackAtKey(tileKey{x: tileX, y: tileY})
	if stack == nil {
		return false
	}

	for _, unitID := range stack.UnitIDs() {
		if unitID == ignoredUnitID {
			continue
		}

		current, ok := m.unitByID(unitID)
		if !ok || current == nil || !current.Alive() {
			continue
		}
		if !current.BlocksMovement() {
			continue
		}

		return true
	}

	return false
}
