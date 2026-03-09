package unit

import (
	"fmt"
	"math"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/pathfinding"
	"github.com/unng-lab/endless/pkg/world"
)

const updateBatchSize = 16

type Manager struct {
	world    world.World
	renderer *Renderer

	units *orderedUnitMap

	impacts         []impactEffect
	jobReports      []JobReport
	tileStacks      map[tileKey]*TileStack
	registeredTiles map[int64]tileKey
	selectedID      int64
	nextID          int64

	workers        []chan updateRequest
	updateWG       sync.WaitGroup
	tileRegistryMu sync.RWMutex
}

type updateRequest struct {
	tick  int64
	delta float64
}

// tickableUnit describes gameplay bodies that participate in the manager's per-frame update
// loop. Static obstacles implement it too, but stay excluded while their eternal-sleep flag
// is set.
type tickableUnit interface {
	Unit
	Tick(int64, float64, func(geom.Point) float64)
	ShouldUpdate() bool
}

type tileKey struct {
	x int
	y int
}

func NewManager(gameWorld world.World, units []Unit) *Manager {
	m := &Manager{
		world:           gameWorld,
		renderer:        NewRenderer(),
		units:           newOrderedUnitMap(len(units)),
		tileStacks:      make(map[tileKey]*TileStack),
		registeredTiles: make(map[int64]tileKey),
	}
	m.assignUnitIDs(units)
	for _, current := range units {
		m.units.Set(current)
	}
	m.registerExistingTileUnits()
	m.startWorkers()
	return m
}

func (m *Manager) Update(gameTick int64, delta float64) {
	if m.units.Len() > 0 {
		if len(m.workers) == 0 {
			for i := 0; i < m.units.Len(); i++ {
				current, ok := m.units.At(i)
				if !ok {
					continue
				}
				m.tickUnit(current, gameTick, delta)
			}
		} else {
			for i := range m.workers {
				m.updateWG.Add(1)
				m.workers[i] <- updateRequest{tick: gameTick, delta: delta}
			}
			m.updateWG.Wait()
		}
	}

	m.updateProjectiles(delta)
	m.updateImpacts(delta)
	m.collectUnitJobReports()
}

func (m *Manager) Draw(screen *ebiten.Image, cam *camera.Camera, quality assets.Quality) error {
	bounds := screen.Bounds()
	return m.renderer.Draw(
		screen,
		cam,
		m.world.TileSize(),
		quality,
		m.visibleUnits(cam, bounds.Dx(), bounds.Dy()),
		m.impacts,
	)
}

func (m *Manager) HasSelected() bool {
	_, ok := m.selectedUnit()
	return ok
}

func (m *Manager) SelectAtScreen(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) {
	if m.PointInPanel(cam, cursor, screenWidth, screenHeight) {
		return
	}
	if cam == nil {
		m.selectedID = 0
		return
	}

	worldPos := cam.ScreenToWorld(cursor)
	if !m.pointInWorld(worldPos) {
		m.selectedID = 0
		return
	}

	stack := m.stackAtWorldPoint(worldPos)
	candidates := m.unitsFromStack(stack)
	if len(candidates) == 0 {
		m.selectedID = 0
		return
	}

	selected := candidates[len(candidates)-1]
	m.selectedID = selected.UnitID()
	if body, ok := selected.(*NonStaticUnit); ok {
		body.Wake()
	}
}

func (m *Manager) PointInPanel(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) bool {
	rect, ok := m.PanelRect(cam, screenWidth, screenHeight)
	return ok && pointInRect(cursor, rect)
}

func (m *Manager) CommandSelectedMove(targetTileX, targetTileY int) error {
	selected, ok := m.selectedNonStatic()
	if !ok {
		if m.HasSelected() {
			return fmt.Errorf("selected object is immobile")
		}
		return nil
	}
	if !selected.IsMobile() {
		return fmt.Errorf("unit %q is immobile", selected.Name())
	}

	startTileX, startTileY := selected.Base().TilePosition(m.world.TileSize())
	grid := worldGrid{world: m.world, blocked: m.blockedTiles(selected.UnitID())}
	path, err := pathfinding.FindPath(
		grid,
		pathfinding.Step{X: startTileX, Y: startTileY},
		pathfinding.Step{X: targetTileX, Y: targetTileY},
	)
	if err != nil {
		return err
	}

	selected.QueueMoveCommand(m.worldPath(path))
	return nil
}

// AssignMoveJob resolves a path for the actor-issued job and binds that job to the unit so
// completion or cancellation can later be reported back through DrainJobReports.
func (m *Manager) AssignMoveJob(job MoveJob) error {
	current, ok := m.unitByID(job.UnitID)
	if !ok {
		err := fmt.Errorf("unit %d not found", job.UnitID)
		m.appendJobFailure(job)
		return err
	}

	body, ok := current.(*NonStaticUnit)
	if !ok || !body.IsMobile() {
		err := fmt.Errorf("unit %d is immobile", job.UnitID)
		m.appendJobFailure(job)
		return err
	}

	startTileX, startTileY := body.Base().TilePosition(m.world.TileSize())
	grid := worldGrid{world: m.world, blocked: m.blockedTiles(body.UnitID())}
	path, err := pathfinding.FindPath(
		grid,
		pathfinding.Step{X: startTileX, Y: startTileY},
		pathfinding.Step{X: job.TargetTileX, Y: job.TargetTileY},
	)
	if err != nil {
		m.appendJobFailure(job)
		return err
	}

	body.AssignMoveJob(job, m.worldPath(path))
	return nil
}

func (m *Manager) CommandSelectedFire(target geom.Point) error {
	selected, ok := m.selectedNonStatic()
	if !ok {
		if m.HasSelected() {
			return fmt.Errorf("selected object cannot shoot")
		}
		return nil
	}
	if !selected.CanShoot() {
		return fmt.Errorf("unit %q cannot shoot", selected.Name())
	}

	shot, err := newProjectile(selected, target, m.world)
	if err != nil {
		return err
	}

	m.nextID++
	shot.SetUnitID(m.nextID)
	m.units.Set(shot)
	return nil
}

func (m *Manager) Selected() (Unit, bool) {
	return m.selectedUnit()
}

// AddUnit registers a freshly spawned unit in the manager and returns the persistent ID that
// the caller should use for later commands, selections or job ownership tracking.
func (m *Manager) AddUnit(body Unit) int64 {
	if body == nil {
		return 0
	}

	if body.UnitID() == 0 {
		m.nextID++
		body.SetUnitID(m.nextID)
	} else if body.UnitID() > m.nextID {
		m.nextID = body.UnitID()
	}

	m.units.Set(body)
	m.registerUnitInCurrentTile(body)

	return body.UnitID()
}

// DrainJobReports returns all actor-facing status changes emitted since the previous drain.
// The manager aggregates reports both from job assignment failures and from units that finish
// or lose ownership of their active move job during simulation.
func (m *Manager) DrainJobReports() []JobReport {
	m.collectUnitJobReports()
	if len(m.jobReports) == 0 {
		return nil
	}

	reports := append([]JobReport(nil), m.jobReports...)
	m.jobReports = m.jobReports[:0]
	return reports
}

func (m *Manager) selectedUnit() (Unit, bool) {
	if m.selectedID == 0 {
		return nil, false
	}

	return m.unitByID(m.selectedID)
}

func (m *Manager) selectedNonStatic() (*NonStaticUnit, bool) {
	selected, ok := m.selectedUnit()
	if !ok {
		return nil, false
	}
	body, ok := selected.(*NonStaticUnit)
	return body, ok
}

func (m *Manager) selectedVisible(cam *camera.Camera, screenWidth, screenHeight int) bool {
	selected, ok := m.selectedUnit()
	if !ok {
		return false
	}

	return unitVisibleOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, selected)
}

func (m *Manager) assignUnitIDs(units []Unit) {
	for i := range units {
		if units[i].UnitID() == 0 {
			m.nextID++
			units[i].SetUnitID(m.nextID)
			continue
		}
		if units[i].UnitID() > m.nextID {
			m.nextID = units[i].UnitID()
		}
	}
}

func (m *Manager) unitByID(unitID int64) (Unit, bool) {
	return m.units.Get(unitID)
}

func (m *Manager) startWorkers() {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}

	m.workers = make([]chan updateRequest, 0, workerCount)
	for i := range workerCount {
		ch := make(chan updateRequest, 1)
		m.workers = append(m.workers, ch)
		go m.workerRun(i, workerCount, ch)
	}
}

func (m *Manager) workerRun(offset, workerCount int, updates <-chan updateRequest) {
	for req := range updates {
		m.processUpdates(offset, workerCount, req)
	}
}

func (m *Manager) processUpdates(offset, workerCount int, req updateRequest) {
	defer m.updateWG.Done()

	stride := workerCount * updateBatchSize
	for blockStart := offset * updateBatchSize; blockStart < m.units.Len(); blockStart += stride {
		for j := 0; j < updateBatchSize; j++ {
			idx := blockStart + j
			if idx >= m.units.Len() {
				break
			}
			current, ok := m.units.At(idx)
			if !ok {
				continue
			}
			m.tickUnit(current, req.tick, req.delta)
		}
	}
}

func (m *Manager) tickUnit(unit Unit, gameTick int64, delta float64) {
	body, ok := unit.(tickableUnit)
	if !ok || !body.ShouldUpdate() {
		return
	}

	previousTileX, previousTileY := body.Base().TilePosition(m.world.TileSize())
	body.Tick(gameTick, delta, m.tileSpeedMultiplierAt)
	if !unitUsesTileStack(body) {
		return
	}

	currentTileX, currentTileY := body.Base().TilePosition(m.world.TileSize())
	if previousTileX == currentTileX && previousTileY == currentTileY {
		return
	}

	m.moveUnitToTile(
		body,
		tileKey{x: previousTileX, y: previousTileY},
		tileKey{x: currentTileX, y: currentTileY},
	)
}

func (m *Manager) tileSpeedMultiplierAt(position geom.Point) float64 {
	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	if !m.world.InBounds(tileX, tileY) {
		return 0
	}

	return m.world.TileType(tileX, tileY).SpeedMultiplier()
}

func (m *Manager) blockedTiles(excludedUnitID int64) map[pathfinding.Step]struct{} {
	blocked := make(map[pathfinding.Step]struct{})
	m.units.Range(func(currentUnit Unit) bool {
		if currentUnit.UnitID() == excludedUnitID || !currentUnit.BlocksMovement() || !currentUnit.Alive() {
			return true
		}

		tileX, tileY := currentUnit.Base().TilePosition(m.world.TileSize())
		blocked[pathfinding.Step{X: tileX, Y: tileY}] = struct{}{}
		return true
	})

	return blocked
}

func (m *Manager) worldPath(path []pathfinding.Step) []geom.Point {
	if len(path) == 0 {
		return nil
	}

	worldPath := make([]geom.Point, 0, len(path))
	for _, step := range path {
		worldPath = append(worldPath, geom.Point{
			X: (float64(step.X) + 0.5) * m.world.TileSize(),
			Y: (float64(step.Y) + 0.5) * m.world.TileSize(),
		})
	}

	return worldPath
}

// registerExistingTileUnits seeds the sparse tile map from the initial unit list or from
// bulk-created units loaded before the simulation starts ticking.
func (m *Manager) registerExistingTileUnits() {
	m.units.Range(func(current Unit) bool {
		m.registerUnitInCurrentTile(current)
		return true
	})
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

	m.tileRegistryMu.Lock()
	defer m.tileRegistryMu.Unlock()

	if registeredKey, isRegistered := m.registeredTiles[unit.UnitID()]; isRegistered {
		from = registeredKey
	}
	if from == to {
		return
	}

	if previousStack := m.tileStacks[from]; previousStack != nil {
		unit.LeaveTile(previousStack)
		m.dropEmptyTileStackLocked(from, previousStack)
	}

	currentStack := m.ensureTileStackLocked(to)
	unit.EnterTile(currentStack)
	m.registeredTiles[unit.UnitID()] = to
}

func (m *Manager) stackAtWorldPoint(position geom.Point) *TileStack {
	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	m.tileRegistryMu.RLock()
	stack := m.tileStacks[tileKey{x: tileX, y: tileY}]
	m.tileRegistryMu.RUnlock()
	return stack
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

// visibleUnits builds the frame-local draw order from tile stacks inside the visible tile
// band plus a one-tile margin for sprites that visually overhang their anchor tile.
func (m *Manager) visibleUnits(cam *camera.Camera, screenWidth, screenHeight int) []Unit {
	if cam == nil || screenWidth <= 0 || screenHeight <= 0 {
		return m.units.Snapshot()
	}

	visibleTiles := m.world.VisibleRange(cam.ViewRect(float64(screenWidth), float64(screenHeight)))
	minX := geom.ClampInt(visibleTiles.Min.X-1, 0, m.world.Columns())
	minY := geom.ClampInt(visibleTiles.Min.Y-1, 0, m.world.Rows())
	maxX := geom.ClampInt(visibleTiles.Max.X+1, 0, m.world.Columns())
	maxY := geom.ClampInt(visibleTiles.Max.Y+1, 0, m.world.Rows())

	seen := make(map[int64]struct{})
	visible := make([]Unit, 0)
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			m.tileRegistryMu.RLock()
			stack := m.tileStacks[tileKey{x: x, y: y}]
			m.tileRegistryMu.RUnlock()
			for _, current := range m.unitsFromStack(stack) {
				if _, alreadyAdded := seen[current.UnitID()]; alreadyAdded {
					continue
				}
				if !unitVisibleOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, current) {
					continue
				}

				seen[current.UnitID()] = struct{}{}
				visible = append(visible, current)
			}
		}
	}

	m.units.Range(func(current Unit) bool {
		if unitUsesTileStack(current) || !unitVisibleOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, current) {
			return true
		}
		visible = append(visible, current)
		return true
	})

	return visible
}

func (m *Manager) updateProjectiles(delta float64) {
	if delta <= 0 || m.units.Len() == 0 {
		return
	}

	active := newOrderedUnitMap(m.units.Len())
	m.units.Range(func(current Unit) bool {
		shot, ok := current.(*Projectile)
		if !ok {
			active.Set(current)
			return true
		}

		previousTileX, previousTileY := shot.Base().TilePosition(m.world.TileSize())
		previousKey := tileKey{x: previousTileX, y: previousTileY}
		moved := shot.Tick(delta)
		if moved {
			currentTileX, currentTileY := shot.Base().TilePosition(m.world.TileSize())
			currentKey := tileKey{x: currentTileX, y: currentTileY}
			if previousKey != currentKey {
				m.tileRegistryMu.RLock()
				previousStack := m.tileStacks[previousKey]
				currentStack := m.tileStacks[currentKey]
				m.tileRegistryMu.RUnlock()

				if previousStack != nil {
					shot.LeaveTile(previousStack)
				}
				shot.EnterTile(currentStack)
				if target, hit := m.firstProjectileOccupant(currentStack, shot.OwnerID); hit {
					impactPos := shot.Position
					m.tileRegistryMu.RLock()
					targetPreviousKey, targetWasRegistered := m.registeredTiles[target.UnitID()]
					m.tileRegistryMu.RUnlock()
					if target.ApplyDamage(shot.Damage) {
						target.Respawn()
						if unitUsesTileStack(target) {
							if targetWasRegistered {
								m.unregisterUnitFromTile(target, targetPreviousKey)
							}
							m.registerUnitInCurrentTile(target)
						}
					}
					m.impacts = append(m.impacts, newImpactEffect(impactPos, m.world.TileSize()))
					return true
				}
			}
		}

		if !shot.IsActive() {
			return true
		}

		active.Set(shot)
		return true
	})

	m.units = active
	if _, ok := m.selectedUnit(); !ok {
		m.selectedID = 0
	}
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

func (m *Manager) updateImpacts(delta float64) {
	if delta <= 0 || len(m.impacts) == 0 {
		return
	}

	active := m.impacts[:0]
	for _, effect := range m.impacts {
		effect.Age += delta
		if effect.Age >= effect.Duration {
			continue
		}
		active = append(active, effect)
	}
	m.impacts = active
}

// collectUnitJobReports pulls per-unit job events into the manager-owned queue so the game
// loop can drain them without reaching into individual unit internals.
func (m *Manager) collectUnitJobReports() {
	m.units.Range(func(current Unit) bool {
		reporter, ok := current.(jobReportingUnit)
		if !ok {
			return true
		}

		m.jobReports = append(m.jobReports, reporter.drainJobReports()...)
		return true
	})
}

func (m *Manager) appendJobFailure(job MoveJob) {
	m.jobReports = append(m.jobReports, JobReport{
		JobID:       job.ID,
		ActorID:     job.ActorID,
		UnitID:      job.UnitID,
		Status:      JobStatusFailed,
		TargetTileX: job.TargetTileX,
		TargetTileY: job.TargetTileY,
	})
}

func (m *Manager) pointInWorld(position geom.Point) bool {
	return position.X >= 0 &&
		position.Y >= 0 &&
		position.X <= m.world.Width() &&
		position.Y <= m.world.Height()
}

func unitUsesTileStack(unit Unit) bool {
	return unit != nil && unit.Alive() && unit.Selectable()
}

type worldGrid struct {
	world   world.World
	blocked map[pathfinding.Step]struct{}
}

func (g worldGrid) InBounds(x, y int) bool {
	return g.world.InBounds(x, y)
}

func (g worldGrid) Cost(x, y int) float64 {
	if !g.InBounds(x, y) {
		return 0
	}
	if _, blocked := g.blocked[pathfinding.Step{X: x, Y: y}]; blocked {
		return 0
	}

	return g.world.TileType(x, y).MovementCost()
}
