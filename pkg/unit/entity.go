package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

// Unit is the common runtime type for every gameplay body managed by the scene. Concrete
// implementations split the behavior into mobile bodies, static bodies and projectiles while
// still letting the manager keep one heterogeneous ordered collection.
type Unit interface {
	Base() *BaseUnit
	UnitID() int64
	SetUnitID(int64)
	UnitKind() Kind
	Name() string
	Frame() int
	Alive() bool
	IsMobile() bool
	BlocksMovement() bool
	CanShoot() bool
	CurrentHealth() int
	MaxHealthValue() int
	HealthRatio() float64
	ApplyDamage(int) bool
	Respawn()
	Selectable() bool
	EnterTile(*TileStack)
	LeaveTile(*TileStack)
}

// BaseUnit stores the spatial and tick-based movement state shared by all world units.
// Concrete unit kinds embed it so interpolation, tile lookup and visibility bookkeeping stay
// identical across runners, obstacles and projectiles.
type BaseUnit struct {
	Position geom.Point

	path           []geom.Point
	sleepTime      int
	lastUpdateTick int64
	travel         travelState
	updateSleeping bool
}

func (s BaseUnit) TilePosition(tileSize float64) (int, int) {
	if tileSize <= 0 {
		return 0, 0
	}

	return int(math.Floor(s.Position.X / tileSize)), int(math.Floor(s.Position.Y / tileSize))
}

func (s BaseUnit) HasPath() bool {
	return len(s.path) > 0
}

func (s BaseUnit) IsMoving() bool {
	return len(s.path) > 0 || (s.travel.active && s.travel.remaining > 0)
}

func (s BaseUnit) PathLen() int {
	return len(s.path)
}

func (s BaseUnit) SleepTime() int {
	return s.sleepTime
}

func (s BaseUnit) LastUpdateTick() int64 {
	return s.lastUpdateTick
}

// UpdateSleeping reports whether the manager must skip this unit during the main update
// pass until some external code explicitly wakes it again.
func (s BaseUnit) UpdateSleeping() bool {
	return s.updateSleeping
}

func (s BaseUnit) Destination() (geom.Point, bool) {
	if len(s.path) == 0 {
		if s.travel.active {
			return s.travel.to, true
		}
		return geom.Point{}, false
	}

	return s.path[len(s.path)-1], true
}

// RenderPosition reconstructs the in-between position for the current move segment.
// The gameplay state snaps Position straight to the next waypoint when travel starts, so
// collision, occupancy and path progression all operate on discrete tile centers. During
// rendering we interpolate back from the segment origin to the destination using the
// remaining tick budget stored in travel.
func (s BaseUnit) RenderPosition() geom.Point {
	if !s.travel.active || s.travel.duration <= 0 {
		return s.Position
	}

	progress := 1 - float64(s.travel.remaining)/float64(s.travel.duration)
	progress = geom.ClampFloat(progress, 0, 1)
	return geom.Point{
		X: s.travel.from.X + (s.travel.to.X-s.travel.from.X)*progress,
		Y: s.travel.from.Y + (s.travel.to.Y-s.travel.from.Y)*progress,
	}
}

func (s *BaseUnit) consumeReachedWaypoint(target geom.Point) bool {
	dx := target.X - s.Position.X
	dy := target.Y - s.Position.Y
	if math.Hypot(dx, dy) > 1e-6 {
		return false
	}

	s.Position = target
	s.path = s.path[1:]
	return true
}

func (s *BaseUnit) clearTravel() {
	s.travel = travelState{}
}

// WakeForUpdate leaves the eternal-sleep mode so the manager may call Tick again.
func (s *BaseUnit) WakeForUpdate() {
	s.updateSleeping = false
}

// SleepUntilExternalWake blocks future Tick calls until some external event reactivates the
// unit. Static units use this to stay completely out of the hot update loop by default.
func (s *BaseUnit) SleepUntilExternalWake() {
	s.updateSleeping = true
}
