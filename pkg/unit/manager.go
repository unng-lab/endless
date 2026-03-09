package unit

import (
	"fmt"
	"math"
	"math/rand/v2"
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

	units     []Unit
	impacts   []impactEffect
	occupants map[tileKey][]int
	selected  int
	nextID    int64

	workers  []chan updateRequest
	updateWG sync.WaitGroup
}

type updateRequest struct {
	tick  int64
	delta float64
}

type tileKey struct {
	x int
	y int
}

func NewManager(gameWorld world.World, units []Unit) *Manager {
	m := &Manager{
		world:     gameWorld,
		renderer:  NewRenderer(),
		units:     append([]Unit(nil), units...),
		occupants: make(map[tileKey][]int),
		selected:  -1,
	}
	m.assignUnitIDs()
	m.rebuildOccupants()
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
	m.rebuildOccupants()
}

func (m *Manager) SyncVisibility(cam *camera.Camera, screenWidth, screenHeight int) {
	for i := range m.units {
		UpdateOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, m.units[i])
	}
}

func (m *Manager) Draw(screen *ebiten.Image, cam *camera.Camera, quality assets.Quality) error {
	return m.renderer.Draw(screen, cam, m.world.TileSize(), quality, m.units, m.impacts)
}

func (m *Manager) HasSelected() bool {
	_, ok := m.selectedUnit()
	return ok
}

func (m *Manager) SelectAtScreen(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) {
	if m.PointInPanel(cursor, screenWidth, screenHeight) {
		return
	}
	if cam == nil {
		m.selected = -1
		return
	}

	worldPos := cam.ScreenToWorld(cursor)
	if !m.pointInWorld(worldPos) {
		m.selected = -1
		return
	}

	tileX := int(math.Floor(worldPos.X / m.world.TileSize()))
	tileY := int(math.Floor(worldPos.Y / m.world.TileSize()))
	candidates := m.occupants[tileKey{x: tileX, y: tileY}]
	if len(candidates) == 0 {
		m.selected = -1
		return
	}

	m.selected = candidates[rand.IntN(len(candidates))]
	if selected, ok := m.selectedNonStatic(); ok {
		selected.Wake()
	}
}

func (m *Manager) PointInPanel(cursor geom.Point, screenWidth, screenHeight int) bool {
	rect, ok := m.PanelRect(screenWidth, screenHeight)
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
	grid := worldGrid{world: m.world, blocked: m.blockedTiles(m.selected)}
	path, err := pathfinding.FindPath(
		grid,
		pathfinding.Step{X: startTileX, Y: startTileY},
		pathfinding.Step{X: targetTileX, Y: targetTileY},
	)
	if err != nil {
		return err
	}

	selected.SetPath(m.worldPath(path))
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
	return nil
}

func (m *Manager) Selected() (Unit, bool) {
	return m.selectedUnit()
}

func (m *Manager) selectedUnit() (Unit, bool) {
	if m.selected < 0 || m.selected >= len(m.units) {
		return nil, false
	}
	return m.units[m.selected], true
}

func (m *Manager) selectedNonStatic() (*NonStaticUnit, bool) {
	selected, ok := m.selectedUnit()
	if !ok {
		return nil, false
	}
	body, ok := selected.(*NonStaticUnit)
	return body, ok
}

func (m *Manager) selectedOnScreen() bool {
	selected, ok := m.selectedUnit()
	return ok && selected.Base().OnScreen
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
	body, ok := unit.(*NonStaticUnit)
	if !ok {
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

func (m *Manager) blockedTiles(excludedUnit int) map[pathfinding.Step]struct{} {
	blocked := make(map[pathfinding.Step]struct{})
	for index, currentUnit := range m.units {
		if index == excludedUnit || !currentUnit.BlocksMovement() || !currentUnit.Alive() {
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

func (m *Manager) rebuildOccupants() {
	clear(m.occupants)
	for i := range m.units {
		current := m.units[i]
		if !current.Alive() || !current.Selectable() {
			continue
		}

		tileX, tileY := current.Base().TilePosition(m.world.TileSize())
		key := tileKey{x: tileX, y: tileY}
		m.occupants[key] = append(m.occupants[key], i)
	}
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

		moved := shot.Tick(delta)
		if moved {
			if unitIndex, hit := m.firstProjectileOccupant(shot, shot.OwnerID); hit {
				impactPos := shot.Position
				if m.units[unitIndex].ApplyDamage(shot.Damage) {
					m.units[unitIndex].Respawn()
				}
				m.impacts = append(m.impacts, newImpactEffect(impactPos, m.world.TileSize()))
				continue
			}
		}

		if !shot.IsActive() {
			continue
		}

		active = append(active, shot)
	}

	m.units = active
	if m.selected >= len(m.units) {
		m.selected = -1
	}
}

// firstProjectileOccupant resolves which unit, if any, occupies the projectile's current
// logical tile. Scanning units directly keeps the answer consistent even when an earlier hit
// in the same tick respawns a unit and changes tile occupancy immediately.
func (m *Manager) firstProjectileOccupant(shot *Projectile, ownerID int64) (int, bool) {
	shotTileX, shotTileY := shot.Base().TilePosition(m.world.TileSize())
	for i := range m.units {
		currentUnit := m.units[i]
		if !currentUnit.Alive() || !currentUnit.Selectable() || currentUnit.UnitID() == ownerID {
			continue
		}

		tileX, tileY := currentUnit.Base().TilePosition(m.world.TileSize())
		if tileX != shotTileX || tileY != shotTileY {
			continue
		}

		return i, true
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

func (m *Manager) pointInWorld(position geom.Point) bool {
	return position.X >= 0 &&
		position.Y >= 0 &&
		position.X <= m.world.Width() &&
		position.Y <= m.world.Height()
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
