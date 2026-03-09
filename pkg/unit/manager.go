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

	units []Unit

	impacts         []impactEffect
	jobReports      []JobReport
	tileStacks      map[tileKey]*TileStack
	registeredTiles map[int64]tileKey
	unitIndexByID   map[int64]int
	selectedID      int64
	nextID          int64

	workers  []chan updateRequest
	updateWG sync.WaitGroup
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
		units:           append([]Unit(nil), units...),
		tileStacks:      make(map[tileKey]*TileStack),
		registeredTiles: make(map[int64]tileKey),
		unitIndexByID:   make(map[int64]int),
	}
	m.assignUnitIDs()
	m.rebuildUnitIndex()
	m.syncTileStacks()
	m.startWorkers()
	return m
}

func (m *Manager) Update(gameTick int64, delta float64) {
	if len(m.units) > 0 {
		if len(m.workers) == 0 {
			for i := range m.units {
				m.tickUnit(m.units[i], gameTick, delta)
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
	m.rebuildUnitIndex()
	m.syncTileStacks()
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
	m.units = append(m.units, shot)
	m.rebuildUnitIndex()
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

	m.units = append(m.units, body)
	m.rebuildUnitIndex()
	m.syncTileStacks()

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

func (m *Manager) assignUnitIDs() {
	for i := range m.units {
		if m.units[i].UnitID() == 0 {
			m.nextID++
			m.units[i].SetUnitID(m.nextID)
			continue
		}
		if m.units[i].UnitID() > m.nextID {
			m.nextID = m.units[i].UnitID()
		}
	}
}

func (m *Manager) rebuildUnitIndex() {
	clear(m.unitIndexByID)
	for index, current := range m.units {
		if current == nil || current.UnitID() == 0 {
			continue
		}
		m.unitIndexByID[current.UnitID()] = index
	}
}

func (m *Manager) unitByID(unitID int64) (Unit, bool) {
	index, ok := m.unitIndexByID[unitID]
	if !ok || index < 0 || index >= len(m.units) {
		return nil, false
	}

	return m.units[index], true
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
	for blockStart := offset * updateBatchSize; blockStart < len(m.units); blockStart += stride {
		for j := 0; j < updateBatchSize; j++ {
			idx := blockStart + j
			if idx >= len(m.units) {
				break
			}
			m.tickUnit(m.units[idx], req.tick, req.delta)
		}
	}
}

func (m *Manager) tickUnit(unit Unit, gameTick int64, delta float64) {
	body, ok := unit.(tickableUnit)
	if !ok || !body.ShouldUpdate() {
		return
	}

	body.Tick(gameTick, delta, m.tileSpeedMultiplierAt)
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
	for _, currentUnit := range m.units {
		if currentUnit.UnitID() == excludedUnitID || !currentUnit.BlocksMovement() || !currentUnit.Alive() {
			continue
		}

		tileX, tileY := currentUnit.Base().TilePosition(m.world.TileSize())
		blocked[pathfinding.Step{X: tileX, Y: tileY}] = struct{}{}
	}

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

// syncTileStacks keeps the sparse tile map aligned with the current logical tile of every
// selectable unit. The reconciliation runs after unit updates so tile enter/leave callbacks
// stay serialized even though movement ticks are processed by workers.
func (m *Manager) syncTileStacks() {
	desiredTiles := make(map[int64]tileKey, len(m.units))

	for _, current := range m.units {
		if !unitUsesTileStack(current) {
			continue
		}

		tileX, tileY := current.Base().TilePosition(m.world.TileSize())
		key := tileKey{x: tileX, y: tileY}
		desiredTiles[current.UnitID()] = key

		previous, hadPrevious := m.registeredTiles[current.UnitID()]
		if hadPrevious && previous == key {
			continue
		}

		if hadPrevious {
			m.unregisterUnitFromTile(current, previous)
		}

		stack := m.ensureTileStack(key)
		current.EnterTile(stack)
		m.registeredTiles[current.UnitID()] = key
	}

	for unitID, previous := range m.registeredTiles {
		if _, stillTracked := desiredTiles[unitID]; stillTracked {
			continue
		}

		current, ok := m.unitByID(unitID)
		if ok {
			m.unregisterUnitFromTile(current, previous)
		} else if stack := m.tileStacks[previous]; stack != nil {
			stack.RemoveUnit(unitID)
			m.dropEmptyTileStack(previous, stack)
		}

		delete(m.registeredTiles, unitID)
	}
}

func (m *Manager) ensureTileStack(key tileKey) *TileStack {
	stack, ok := m.tileStacks[key]
	if ok {
		return stack
	}

	stack = &TileStack{}
	m.tileStacks[key] = stack
	return stack
}

func (m *Manager) unregisterUnitFromTile(unit Unit, key tileKey) {
	stack := m.tileStacks[key]
	if stack == nil {
		return
	}

	unit.LeaveTile(stack)
	m.dropEmptyTileStack(key, stack)
	delete(m.registeredTiles, unit.UnitID())
}

func (m *Manager) dropEmptyTileStack(key tileKey, stack *TileStack) {
	if stack == nil || !stack.Empty() {
		return
	}

	delete(m.tileStacks, key)
}

func (m *Manager) stackAtWorldPoint(position geom.Point) *TileStack {
	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	return m.tileStacks[tileKey{x: tileX, y: tileY}]
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
		return append([]Unit(nil), m.units...)
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
			stack := m.tileStacks[tileKey{x: x, y: y}]
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

	for _, current := range m.units {
		if unitUsesTileStack(current) || !unitVisibleOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, current) {
			continue
		}
		visible = append(visible, current)
	}

	return visible
}

func (m *Manager) updateProjectiles(delta float64) {
	if delta <= 0 || len(m.units) == 0 {
		return
	}

	active := m.units[:0]
	for _, current := range m.units {
		shot, ok := current.(*Projectile)
		if !ok {
			active = append(active, current)
			continue
		}

		previousTileX, previousTileY := shot.Base().TilePosition(m.world.TileSize())
		previousKey := tileKey{x: previousTileX, y: previousTileY}
		moved := shot.Tick(delta)
		if moved {
			currentTileX, currentTileY := shot.Base().TilePosition(m.world.TileSize())
			currentKey := tileKey{x: currentTileX, y: currentTileY}
			if previousKey != currentKey {
				if previousStack := m.tileStacks[previousKey]; previousStack != nil {
					shot.LeaveTile(previousStack)
				}

				currentStack := m.tileStacks[currentKey]
				shot.EnterTile(currentStack)
				if unitIndex, hit := m.firstProjectileOccupant(currentStack, shot.OwnerID); hit {
					impactPos := shot.Position
					if m.units[unitIndex].ApplyDamage(shot.Damage) {
						m.units[unitIndex].Respawn()
					}
					m.impacts = append(m.impacts, newImpactEffect(impactPos, m.world.TileSize()))
					continue
				}
			}
		}

		if !shot.IsActive() {
			continue
		}

		active = append(active, shot)
	}

	m.units = active
	m.rebuildUnitIndex()
	if _, ok := m.selectedUnit(); !ok {
		m.selectedID = 0
	}
}

// firstProjectileOccupant resolves hits through the tile stack the projectile has just entered.
// Iterating the stack snapshot keeps the old "check every unit in that tile" behavior while
// moving the broad-phase lookup away from a full scan over every unit in the scene.
func (m *Manager) firstProjectileOccupant(stack *TileStack, ownerID int64) (int, bool) {
	if stack == nil {
		return 0, false
	}

	for _, unitID := range stack.UnitIDs() {
		if unitID == ownerID {
			continue
		}

		index, ok := m.unitIndexByID[unitID]
		if !ok || index < 0 || index >= len(m.units) {
			continue
		}

		currentUnit := m.units[index]
		if !currentUnit.Alive() || !currentUnit.Selectable() {
			continue
		}

		return index, true
	}

	return 0, false
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
	for _, current := range m.units {
		reporter, ok := current.(jobReportingUnit)
		if !ok {
			continue
		}

		m.jobReports = append(m.jobReports, reporter.drainJobReports()...)
	}
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
