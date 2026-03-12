package unit

import (
	"log"
	"runtime"
)

func (m *Manager) Update(gameTick int64) {
	m.lastGameTick = gameTick
	if m.units.SlotsLen() == 0 {
		return
	}

	for i := range m.workers {
		m.updateWG.Add(1)
		m.workers[i] <- gameTick
	}
	m.updateWG.Wait()
	m.flushPendingSpawns()
	if _, ok := m.selectedUnit(); !ok {
		m.selectedID = 0
	}
}

func (m *Manager) startWorkers() {
	workerCount := runtime.GOMAXPROCS(0) / 4
	if workerCount < 1 {
		workerCount = 1
	}
	log.Printf("[startup] units: starting %d update workers", workerCount)

	m.workers = make([]chan int64, 0, workerCount)
	for i := range workerCount {
		ch := make(chan int64, 1)
		m.workers = append(m.workers, ch)
		go m.workerRun(i, workerCount, ch)
	}
}

func (m *Manager) workerRun(offset, workerCount int, updates <-chan int64) {
	for req := range updates {
		m.processUpdates(offset, workerCount, req)
	}
}

func (m *Manager) processUpdates(offset, workerCount int, req int64) {
	defer m.updateWG.Done()

	stride := workerCount * updateBatchSize
	for blockStart := offset * updateBatchSize; blockStart < m.units.SlotsLen(); blockStart += stride {
		for j := 0; j < updateBatchSize; j++ {
			idx := blockStart + j
			if idx >= m.units.SlotsLen() {
				break
			}

			current, ok := m.units.SlotAt(idx)
			if !ok {
				continue
			}

			m.tickUnit(current, req)
		}
	}
}

func (m *Manager) tickUnit(unit Unit, gameTick int64) {
	if unit == nil {
		return
	}

	m.advanceUnitWeaponCooldown(unit)
	if m.retireUnitIfDeleted(unit) {
		return
	}
	if m.skipSleepingUnit(unit) {
		return
	}
	if !unit.ShouldUpdate() {
		return
	}

	previousTileX, previousTileY := unit.Base().TilePosition(m.world.TileSize())
	unit.Tick(gameTick)
	m.collectUnitDeferredSpawns(unit)
	if m.retireUnitIfDeleted(unit) {
		return
	}
	if !unitUsesTileStack(unit) {
		return
	}

	m.reconcileUnitTileRegistration(unit, previousTileX, previousTileY)
}

// advanceUnitWeaponCooldown spends cooldown budget before any early returns in tickUnit so
// weapon readiness remains tied to wall-clock simulation ticks even for sleeping units.
func (m *Manager) advanceUnitWeaponCooldown(unit Unit) {
	body, ok := unit.(*NonStaticUnit)
	if !ok {
		return
	}

	body.advanceWeaponCooldown()
}

// retireUnitIfDeleted centralizes the tombstone fast path so tickUnit can short-circuit at the
// beginning and right after Tick with the same deferred cleanup rule.
func (m *Manager) retireUnitIfDeleted(unit Unit) bool {
	if unit == nil || !unit.Base().PendingRemoval() {
		return false
	}

	m.retireDeletedUnit(unit)
	return true
}

// skipSleepingUnit advances sleep counters that are managed outside concrete Tick methods and
// reports whether the unit should be skipped for the current manager update.
func (m *Manager) skipSleepingUnit(unit Unit) bool {
	if unit.Base().UpdateSleeping() {
		return true
	}
	if unit.Base().SleepTime() == 0 {
		return false
	}

	unit.Base().StepSleep()
	if projectile, ok := unit.(*Projectile); ok && !projectile.IsActive() {
		projectile.MarkForRemoval()
		m.retireDeletedUnit(projectile)
		return true
	}

	return unit.Base().SleepTime() > 0
}

// reconcileUnitTileRegistration updates tile membership only when a tick changed the logical
// tile anchor, which keeps the common no-move case cheap and explicit in the call site.
func (m *Manager) reconcileUnitTileRegistration(unit Unit, previousTileX, previousTileY int) {
	currentTileX, currentTileY := unit.Base().TilePosition(m.world.TileSize())
	if previousTileX == currentTileX && previousTileY == currentTileY {
		return
	}

	m.moveUnitToTile(
		unit,
		tileKey{x: previousTileX, y: previousTileY},
		tileKey{x: currentTileX, y: currentTileY},
	)
}

// retireDeletedUnit performs the immediate post-death bookkeeping for a unit that has already
// transitioned into the deleted state. The ordered slot stays occupied until some future AddUnit
// call reuses it, but tile registration and pending order reports must be flushed right away.
func (m *Manager) retireDeletedUnit(unit Unit) {
	if m == nil || unit == nil || !unit.Base().PendingRemoval() {
		return
	}
	if unit.Base().RemovalHandled() {
		return
	}

	if reporter, ok := unit.(orderReportingUnit); ok {
		m.appendBufferedOrderReports(unit.UnitID(), reporter.drainOrderReports())
	}

	m.tileRegistryMu.RLock()
	registeredKey, isRegistered := m.registeredTiles[unit.UnitID()]
	m.tileRegistryMu.RUnlock()
	if isRegistered {
		m.unregisterUnitFromTile(unit, registeredKey)
	}
	unit.Base().MarkRemovalHandled()
	m.units.ReleaseDeletedSlot(unit.UnitID())
}

func (m *Manager) collectUnitDeferredSpawns(unit Unit) {
	if m == nil || unit == nil {
		return
	}

	spawner, ok := unit.(projectileSpawningUnit)
	if !ok {
		return
	}

	projectiles := spawner.drainPendingProjectiles()
	if len(projectiles) == 0 {
		return
	}

	units := make([]Unit, 0, len(projectiles))
	for _, projectile := range projectiles {
		if projectile == nil {
			continue
		}
		units = append(units, projectile)
	}
	if len(units) == 0 {
		return
	}

	m.pendingSpawnsMu.Lock()
	m.pendingSpawns = append(m.pendingSpawns, units...)
	m.pendingSpawnsMu.Unlock()
}

func (m *Manager) flushPendingSpawns() {
	if m == nil {
		return
	}

	m.pendingSpawnsMu.Lock()
	pending := append([]Unit(nil), m.pendingSpawns...)
	m.pendingSpawns = m.pendingSpawns[:0]
	m.pendingSpawnsMu.Unlock()

	for _, current := range pending {
		m.AddUnit(current)
	}
}
