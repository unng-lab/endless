package unit

import (
	"fmt"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	projectileSpeedPerTick = 320.0 / 60.0
	projectileDamage       = 1
	projectileRadiusScale  = 0.16
	projectileRangeTiles   = 14.0
	projectileEntryOffset  = 0.05
	impactDurationTicks    = 11
	impactRadiusScale      = 0.55
)

type Projectile struct {
	BaseUnit
	ID      int64
	OwnerID int64
	Radius  float64
	Damage  int

	impactRadius        float64
	impactTicks         int
	impactDurationTicks int
	exploding           bool
}

// newProjectile builds a discrete trajectory that advances from tile to tile in the cursor
// direction. The projectile keeps the same sleepTime-based cadence as units, so collision is
// checked only when the logical position enters the next tile on the route.
func newProjectile(owner *NonStaticUnit, target geom.Point, gameWorld world.World) (*Projectile, error) {
	dx := target.X - owner.Position.X
	dy := target.Y - owner.Position.Y
	length := math.Hypot(dx, dy)
	if length <= 1e-6 {
		return nil, fmt.Errorf("cursor is too close to the unit")
	}

	direction := geom.Point{
		X: dx / length,
		Y: dy / length,
	}
	path := buildProjectilePath(owner.Position, direction, gameWorld, gameWorld.TileSize()*projectileRangeTiles)
	if len(path) == 0 {
		return nil, fmt.Errorf("shot leaves the world immediately")
	}

	return &Projectile{
		BaseUnit: BaseUnit{
			Position: owner.Position,
			path:     path,
		},
		OwnerID:             owner.ID,
		Radius:              gameWorld.TileSize() * projectileRadiusScale,
		Damage:              projectileDamage,
		impactRadius:        gameWorld.TileSize() * impactRadiusScale,
		impactDurationTicks: impactDurationTicks,
	}, nil
}

func (p *Projectile) Base() *BaseUnit {
	return &p.BaseUnit
}

func (p *Projectile) UnitID() int64 {
	return p.ID
}

func (p *Projectile) SetUnitID(id int64) {
	p.ID = id
}

func (p *Projectile) UnitKind() Kind {
	return KindProjectile
}

func (p *Projectile) Name() string {
	return "Projectile"
}

func (p *Projectile) Frame() int {
	return 0
}

func (p *Projectile) Alive() bool {
	return true
}

func (p *Projectile) IsMobile() bool {
	return false
}

func (p *Projectile) BlocksMovement() bool {
	return false
}

func (p *Projectile) CanShoot() bool {
	return false
}

func (p *Projectile) CurrentHealth() int {
	return 0
}

func (p *Projectile) MaxHealthValue() int {
	return 0
}

func (p *Projectile) HealthRatio() float64 {
	return 0
}

func (p *Projectile) ApplyDamage(_ int) bool {
	return false
}

func (p *Projectile) Respawn() {
	p.exploding = false
	p.impactTicks = 0
	p.clearTravel()
	p.ClearRemovalMark()
}

func (p *Projectile) Selectable() bool {
	return false
}

func (p *Projectile) EnterTile(stack *TileStack) {
	stack.AddUnit(p.UnitID())
}

func (p *Projectile) LeaveTile(stack *TileStack) {
	stack.RemoveUnit(p.UnitID())
}

// Tick advances the projectile through the same manager-driven update contract as every other
// tickable unit. Projectile-specific side effects are resolved immediately when the manager
// moves the projectile into the next tile, so this method only advances movement or impact age.
func (p *Projectile) Tick(gameTick int64) {
	p.lastUpdateTick = gameTick

	if p.exploding {
		p.impactTicks++
		if !p.IsActive() {
			p.MarkForRemoval()
		}
		return
	}

	if p.sleepTime > 0 {
		return
	}

	p.sleepTime = p.advance()
	p.travel.remaining = p.sleepTime
	if !p.IsActive() {
		p.MarkForRemoval()
	}
}

// UpdateVisible keeps the projectile's render interpolation in sync with the visible draw pass.
// Logical movement still progresses only through Tick and manager-managed sleep countdowns.
func (p *Projectile) UpdateVisible(gameTick int64) {
	if p == nil {
		return
	}

	p.AdvanceVisibleTravel(gameTick)
}

// ShouldUpdate keeps the projectile inside the regular tick loop while it is either flying
// towards the next tile or still finishing its short impact animation.
func (p *Projectile) ShouldUpdate() bool {
	return p.IsActive()
}

// ReactToEnteredTile resolves projectile impacts at the exact point where the manager has
// already registered the projectile inside the newly entered tile. Running the hit test here
// keeps the projectile lifecycle local to the projectile while reusing the same tile-entry
// event that every moving unit already goes through.
func (p *Projectile) ReactToEnteredTile(m *Manager, stack *TileStack) {
	if p == nil || p.exploding || m == nil {
		return
	}

	target, hit := m.firstProjectileOccupant(stack, p.OwnerID)
	if !hit {
		return
	}

	if target.ApplyDamage(p.Damage) {
		m.retireDeletedUnit(target)
	}

	p.StartExplosion()
}

// IsActive reports whether the projectile still has either a future waypoint to traverse or
// a currently interpolated segment that should remain visible.
func (p *Projectile) IsActive() bool {
	if p.exploding {
		return p.impactTicks < p.impactDurationTicks
	}

	return len(p.path) > 0 || p.sleepTime > 0
}

// StartExplosion freezes the projectile at the impact point and reuses the same runtime object
// for the short-lived hit animation so the manager does not need a separate impact collection.
func (p *Projectile) StartExplosion() {
	if p == nil || p.exploding {
		return
	}

	p.exploding = true
	p.impactTicks = 0
	p.path = p.path[:0]
	p.sleepTime = 0
	p.clearTravel()
}

func (p *Projectile) advance() int {
	if len(p.path) == 0 {
		p.clearTravel()
		return 0
	}

	for len(p.path) > 0 {
		target := p.path[0]
		if p.consumeReachedWaypoint(target) {
			continue
		}

		return p.startTravel(target)
	}

	p.clearTravel()
	return 0
}

func (p *Projectile) startTravel(target geom.Point) int {
	dx := target.X - p.Position.X
	dy := target.Y - p.Position.Y
	distance := math.Hypot(dx, dy)
	travelTicks := travelTicksForDistance(distance, projectileSpeedPerTick)

	p.travel = travelState{
		from:            p.RenderPosition(),
		to:              target,
		duration:        travelTicks,
		remaining:       travelTicks,
		visualRemaining: travelTicks,
		active:          true,
	}
	p.Position = target
	p.path = p.path[1:]

	return travelTicks
}

// buildProjectilePath performs a grid traversal in the normalized fire direction and emits
// points just inside each newly entered tile. This keeps the trajectory perfectly straight
// while still guaranteeing that every crossed tile produces a logical interaction point.
func buildProjectilePath(start geom.Point, direction geom.Point, gameWorld world.World, maxDistance float64) []geom.Point {
	tileSize := gameWorld.TileSize()
	if tileSize <= 0 || maxDistance <= 0 {
		return nil
	}

	currentTileX := int(math.Floor(start.X / tileSize))
	currentTileY := int(math.Floor(start.Y / tileSize))
	if !gameWorld.InBounds(currentTileX, currentTileY) {
		return nil
	}

	stepX, tMaxX, tDeltaX := projectileAxisTraversal(start.X, direction.X, tileSize, currentTileX)
	stepY, tMaxY, tDeltaY := projectileAxisTraversal(start.Y, direction.Y, tileSize, currentTileY)
	entryOffset := math.Min(tileSize*projectileEntryOffset, tileSize*0.25)
	path := make([]geom.Point, 0, int(math.Ceil(projectileRangeTiles)))

	for {
		boundaryDistance := 0.0
		moveX := false
		moveY := false

		switch {
		case tMaxX < tMaxY:
			boundaryDistance = tMaxX
			moveX = true
		case tMaxY < tMaxX:
			boundaryDistance = tMaxY
			moveY = true
		default:
			boundaryDistance = tMaxX
			moveX = true
			moveY = true
		}

		sampleDistance := boundaryDistance + entryOffset
		if sampleDistance > maxDistance+1e-6 {
			break
		}

		if moveX {
			currentTileX += stepX
			tMaxX += tDeltaX
		}
		if moveY {
			currentTileY += stepY
			tMaxY += tDeltaY
		}

		if !gameWorld.InBounds(currentTileX, currentTileY) {
			break
		}

		path = append(path, pointAlongRay(start, direction, sampleDistance))
	}

	finalPoint := pointAlongRay(start, direction, maxDistance)
	if pointInsideWorld(finalPoint, gameWorld) {
		if len(path) == 0 || math.Hypot(path[len(path)-1].X-finalPoint.X, path[len(path)-1].Y-finalPoint.Y) > 1e-6 {
			path = append(path, finalPoint)
		}
	}

	return path
}

func projectileAxisTraversal(startCoord, directionCoord, tileSize float64, currentTile int) (int, float64, float64) {
	if math.Abs(directionCoord) <= 1e-9 {
		return 0, math.Inf(1), math.Inf(1)
	}

	if directionCoord > 0 {
		nextBoundary := float64(currentTile+1) * tileSize
		return 1, (nextBoundary - startCoord) / directionCoord, tileSize / directionCoord
	}

	nextBoundary := float64(currentTile) * tileSize
	return -1, (nextBoundary - startCoord) / directionCoord, -tileSize / directionCoord
}

func pointAlongRay(start, direction geom.Point, distance float64) geom.Point {
	return geom.Point{
		X: start.X + direction.X*distance,
		Y: start.Y + direction.Y*distance,
	}
}

func pointInsideWorld(point geom.Point, gameWorld world.World) bool {
	return point.X >= 0 &&
		point.Y >= 0 &&
		point.X < gameWorld.Width() &&
		point.Y < gameWorld.Height()
}
