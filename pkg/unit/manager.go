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

	units       []Unit
	projectiles []projectile
	impacts     []impactEffect
	selected    int
	nextUnitID  int64

	workers  []chan float64
	updateWG sync.WaitGroup
}

func NewManager(gameWorld world.World, units []Unit) *Manager {
	m := &Manager{
		world:    gameWorld,
		renderer: NewRenderer(),
		units:    append([]Unit(nil), units...),
		selected: -1,
	}
	m.assignUnitIDs()
	m.startWorkers()
	return m
}

func (m *Manager) Update(delta float64) {
	if len(m.units) > 0 {
		if len(m.workers) == 0 {
			for i := range m.units {
				m.units[i].Update(delta, m.tileSpeedMultiplierAt)
			}
		} else {
			for i := range m.workers {
				m.updateWG.Add(1)
				m.workers[i] <- delta
			}
			m.updateWG.Wait()
		}
	}

	m.updateProjectiles(delta)
	m.updateImpacts(delta)
}

func (m *Manager) SyncVisibility(cam *camera.Camera, screenWidth, screenHeight int) {
	for i := range m.units {
		UpdateOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, &m.units[i])
	}
}

func (m *Manager) Draw(screen *ebiten.Image, cam *camera.Camera, quality assets.Quality) error {
	return m.renderer.Draw(screen, cam, m.world.TileSize(), quality, m.units, m.projectiles, m.impacts)
}

func (m *Manager) HasSelected() bool {
	_, ok := m.selectedUnit()
	return ok
}

func (m *Manager) SelectAtScreen(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) {
	if m.PointInPanel(cursor, screenWidth, screenHeight) {
		return
	}

	for i := len(m.units) - 1; i >= 0; i-- {
		if !m.units[i].OnScreen {
			continue
		}
		if pointInRect(cursor, ScreenRect(cam, m.world.TileSize(), m.units[i])) {
			m.selected = i
			return
		}
	}

	m.selected = -1
}

func (m *Manager) PointInPanel(cursor geom.Point, screenWidth, screenHeight int) bool {
	rect, ok := m.PanelRect(screenWidth, screenHeight)
	return ok && pointInRect(cursor, rect)
}

func (m *Manager) CommandSelectedMove(targetTileX, targetTileY int) error {
	selected, ok := m.selectedUnit()
	if !ok {
		return nil
	}
	if !selected.IsMobile() {
		return fmt.Errorf("unit %q is immobile", selected.Name())
	}

	startTileX, startTileY := selected.TilePosition(m.world.TileSize())
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
	selected, ok := m.selectedUnit()
	if !ok {
		return nil
	}
	if !selected.CanShoot() {
		return fmt.Errorf("unit %q cannot shoot", selected.Name())
	}

	shot, err := newProjectile(*selected, target, m.world.TileSize())
	if err != nil {
		return err
	}

	m.projectiles = append(m.projectiles, shot)
	return nil
}

func (m *Manager) Selected() (Unit, bool) {
	selected, ok := m.selectedUnit()
	if !ok {
		return Unit{}, false
	}
	return *selected, true
}

func (m *Manager) selectedUnit() (*Unit, bool) {
	if m.selected < 0 || m.selected >= len(m.units) {
		return nil, false
	}
	return &m.units[m.selected], true
}

func (m *Manager) selectedOnScreen() bool {
	selected, ok := m.selectedUnit()
	return ok && selected.OnScreen
}

func (m *Manager) assignUnitIDs() {
	for i := range m.units {
		if m.units[i].ID == 0 {
			m.nextUnitID++
			m.units[i].ID = m.nextUnitID
			continue
		}
		if m.units[i].ID > m.nextUnitID {
			m.nextUnitID = m.units[i].ID
		}
	}
}

func (m *Manager) startWorkers() {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}

	m.workers = make([]chan float64, 0, workerCount)
	for i := range workerCount {
		ch := make(chan float64, 1)
		m.workers = append(m.workers, ch)
		go m.workerRun(i, workerCount, ch)
	}
}

func (m *Manager) workerRun(offset, workerCount int, updates <-chan float64) {
	for delta := range updates {
		m.processUpdates(offset, workerCount, delta)
	}
}

func (m *Manager) processUpdates(offset, workerCount int, delta float64) {
	defer m.updateWG.Done()

	stride := workerCount * updateBatchSize
	for blockStart := offset * updateBatchSize; blockStart < len(m.units); blockStart += stride {
		for j := 0; j < updateBatchSize; j++ {
			idx := blockStart + j
			if idx >= len(m.units) {
				break
			}
			m.units[idx].Update(delta, m.tileSpeedMultiplierAt)
		}
	}
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
		if index == excludedUnit || !currentUnit.BlocksMovement() {
			continue
		}

		tileX, tileY := currentUnit.TilePosition(m.world.TileSize())
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

func (m *Manager) updateProjectiles(delta float64) {
	if delta <= 0 || len(m.projectiles) == 0 {
		return
	}

	active := m.projectiles[:0]
	for _, shot := range m.projectiles {
		if shot.RemainingRange <= 0 {
			continue
		}

		start := shot.Position
		stepDistance := math.Hypot(shot.Velocity.X, shot.Velocity.Y) * delta
		if stepDistance <= 0 {
			continue
		}
		if stepDistance > shot.RemainingRange {
			stepDistance = shot.RemainingRange
		}

		direction := geom.Point{
			X: shot.Velocity.X / math.Hypot(shot.Velocity.X, shot.Velocity.Y),
			Y: shot.Velocity.Y / math.Hypot(shot.Velocity.X, shot.Velocity.Y),
		}
		end := geom.Point{
			X: shot.Position.X + direction.X*stepDistance,
			Y: shot.Position.Y + direction.Y*stepDistance,
		}

		if unitIndex, impactPos, ok := m.firstProjectileCollision(start, end, shot.OwnerID); ok {
			if m.units[unitIndex].ApplyDamage(shot.Damage) {
				m.units[unitIndex].Respawn()
			}
			m.impacts = append(m.impacts, newImpactEffect(impactPos, m.world.TileSize()))
			continue
		}

		if !m.pointInWorld(end) {
			continue
		}

		shot.Position = end
		shot.RemainingRange -= stepDistance
		if shot.RemainingRange <= 0 {
			continue
		}
		active = append(active, shot)
	}

	m.projectiles = active
}

func (m *Manager) firstProjectileCollision(start, end geom.Point, ownerID int64) (int, geom.Point, bool) {
	bestIndex := -1
	bestT := math.Inf(1)
	hitRadius := m.world.TileSize() * unitHitRadiusScale

	for i := range m.units {
		currentUnit := m.units[i]
		if !currentUnit.Alive() || currentUnit.ID == ownerID {
			continue
		}

		t, hit := segmentPointIntersection(start, end, currentUnit.Position, hitRadius)
		if !hit || t >= bestT {
			continue
		}

		bestIndex = i
		bestT = t
	}

	if bestIndex < 0 {
		return 0, geom.Point{}, false
	}

	return bestIndex, pointAlongSegment(start, end, bestT), true
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
