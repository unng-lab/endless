package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

// CombatEventType names one sparse combat-side state transition that higher-level telemetry
// code may persist without reconstructing intent from raw unit snapshots later.
type CombatEventType string

const (
	CombatEventProjectileSpawned CombatEventType = "projectile_spawned"
	CombatEventProjectileHit     CombatEventType = "projectile_hit"
	CombatEventProjectileExpired CombatEventType = "projectile_expired"
	CombatEventUnitKilled        CombatEventType = "unit_killed"
)

// CombatEvent keeps only the fields required by RL-trace storage and offline reward analysis.
// Order lifecycle stays in OrderReport, while this structure captures gameplay-side outcomes.
type CombatEvent struct {
	Tick             int64
	Type             CombatEventType
	SourceUnitID     int64
	TargetUnitID     int64
	ProjectileUnitID int64
	Position         geom.Point
	Damage           int
	Killed           bool
}

// ProjectileSnapshot exposes the small immutable subset of projectile runtime state that
// observation builders need for spatial features. Keeping this separate from UnitSnapshot
// avoids polluting the generic unit view with projectile-only fields.
type ProjectileSnapshot struct {
	UnitID    int64
	OwnerID   int64
	Position  geom.Point
	Direction geom.Point
	Exploding bool
	SleepTime int
}

// BlockingUnitSnapshot exposes the immutable subset of one live movement-blocking unit that
// observation builders need to project local cover occupancy.
type BlockingUnitSnapshot struct {
	UnitID   int64
	Kind     Kind
	Position geom.Point
	TileX    int
	TileY    int
}

// UnitSnapshot exposes one compact read-only view of a world body at one simulation tick.
// The shape is intentionally narrower than the full runtime object so dataset code remains
// stable even when unit internals evolve.
type UnitSnapshot struct {
	UnitID                   int64
	Kind                     Kind
	Position                 geom.Point
	TileX                    int
	TileY                    int
	Health                   int
	MaxHealth                int
	Alive                    bool
	Selectable               bool
	BlocksMovement           bool
	IsMoving                 bool
	SleepTime                int
	WeaponReady              bool
	FireCooldownRemaining    int
	HasActiveFireOrder       bool
	HasQueuedFireOrder       bool
	HasActiveMoveOrder       bool
	HasQueuedMoveOrder       bool
	HasDestination           bool
	Destination              geom.Point
	CurrentActiveOrderKind   OrderKind
	CurrentQueuedOrderKind   OrderKind
	CurrentActiveOrderExists bool
	CurrentQueuedOrderExists bool
}

// DuelSnapshot keeps the exact pairwise state that one fire-only policy usually needs when the
// environment already owns pathfinding and movement through the internal order system.
type DuelSnapshot struct {
	Tick             int64
	Shooter          UnitSnapshot
	Target           UnitSnapshot
	RelativeTarget   geom.Point
	DistanceToTarget float64
	ProjectileCount  int
}

// UnitSnapshot returns one stable projection of the requested runtime object. Callers may use
// this to persist per-step traces without coupling storage to mutable unit internals.
func (m *Manager) UnitSnapshot(unitID int64) (UnitSnapshot, bool) {
	if m == nil || unitID == 0 {
		return UnitSnapshot{}, false
	}

	current, ok := m.unitByID(unitID)
	if !ok || current == nil {
		return UnitSnapshot{}, false
	}

	base := current.Base()
	reachedPosition := base.ReachedPosition()
	tileX, tileY := base.ReachedTilePosition(m.world.TileSize())
	destination, hasDestination := base.Destination()
	snapshot := UnitSnapshot{
		UnitID:         current.UnitID(),
		Kind:           current.UnitKind(),
		Position:       reachedPosition,
		TileX:          tileX,
		TileY:          tileY,
		Health:         current.CurrentHealth(),
		MaxHealth:      current.MaxHealthValue(),
		Alive:          current.Alive(),
		Selectable:     current.Selectable(),
		BlocksMovement: current.BlocksMovement(),
		IsMoving:       base.IsMoving(),
		SleepTime:      base.SleepTime(),
		HasDestination: hasDestination,
		Destination:    destination,
	}

	body, ok := current.(*NonStaticUnit)
	if !ok {
		return snapshot, true
	}

	snapshot.WeaponReady = body.WeaponReady()
	snapshot.FireCooldownRemaining = body.fireCooldownRemaining
	if body.activeOrder.hasOrder {
		snapshot.CurrentActiveOrderKind = body.activeOrder.order.kind
		snapshot.CurrentActiveOrderExists = true
		snapshot.HasActiveFireOrder = body.activeOrder.order.kind == OrderKindFire
		snapshot.HasActiveMoveOrder = body.activeOrder.order.kind == OrderKindMove
	}
	if body.queuedOrder.hasOrder {
		snapshot.CurrentQueuedOrderKind = body.queuedOrder.order.kind
		snapshot.CurrentQueuedOrderExists = true
		snapshot.HasQueuedFireOrder = body.queuedOrder.order.kind == OrderKindFire
		snapshot.HasQueuedMoveOrder = body.queuedOrder.order.kind == OrderKindMove
	}

	return snapshot, true
}

// DuelSnapshot returns the compact shooter-target pair that training code may write every tick.
// The method centralizes lookup and derived metric calculation so individual actors do not
// need to duplicate pairwise bookkeeping.
func (m *Manager) DuelSnapshot(shooterID, targetID int64) (DuelSnapshot, bool) {
	if m == nil || shooterID == 0 || targetID == 0 {
		return DuelSnapshot{}, false
	}

	shooter, ok := m.UnitSnapshot(shooterID)
	if !ok {
		return DuelSnapshot{}, false
	}
	target, ok := m.UnitSnapshot(targetID)
	if !ok {
		return DuelSnapshot{}, false
	}

	relativeTarget := geom.Point{
		X: target.Position.X - shooter.Position.X,
		Y: target.Position.Y - shooter.Position.Y,
	}
	return DuelSnapshot{
		Tick:             m.lastGameTick,
		Shooter:          shooter,
		Target:           target,
		RelativeTarget:   relativeTarget,
		DistanceToTarget: math.Hypot(relativeTarget.X, relativeTarget.Y),
		ProjectileCount:  m.projectileCount(),
	}, true
}

// DrainCombatEvents hands callers every buffered combat-side event in emission order and clears
// the manager-owned tail so later reads only see newly generated outcomes.
func (m *Manager) DrainCombatEvents() []CombatEvent {
	if m == nil {
		return nil
	}

	m.combatEventsMu.Lock()
	defer m.combatEventsMu.Unlock()

	if len(m.combatEvents) == 0 {
		return nil
	}

	events := append([]CombatEvent(nil), m.combatEvents...)
	m.combatEvents = m.combatEvents[:0]
	return events
}

func (m *Manager) appendCombatEvent(event CombatEvent) {
	if m == nil || event.Type == "" {
		return
	}

	m.combatEventsMu.Lock()
	m.combatEvents = append(m.combatEvents, event)
	m.combatEventsMu.Unlock()
}

func (m *Manager) projectileCount() int {
	if m == nil {
		return 0
	}

	count := 0
	m.units.Range(func(current Unit) bool {
		if current != nil && current.UnitKind() == KindProjectile {
			count++
		}
		return true
	})
	return count
}

// ProjectileSnapshots returns one defensive snapshot for every projectile currently tracked by
// the manager. RL observation code uses this to build nearest-projectile features without
// depending on mutable runtime objects or package-private manager storage.
func (m *Manager) ProjectileSnapshots() []ProjectileSnapshot {
	if m == nil {
		return nil
	}

	projectiles := make([]ProjectileSnapshot, 0)
	m.units.Range(func(current Unit) bool {
		projectile, ok := current.(*Projectile)
		if !ok || projectile == nil {
			return true
		}

		projectiles = append(projectiles, ProjectileSnapshot{
			UnitID:    projectile.UnitID(),
			OwnerID:   projectile.OwnerID,
			Position:  projectile.Position,
			Direction: projectile.Direction,
			Exploding: projectile.exploding,
			SleepTime: projectile.SleepTime(),
		})
		return true
	})
	if len(projectiles) == 0 {
		return nil
	}

	return projectiles
}

// BlockingUnitSnapshots returns one defensive snapshot for every live unit that currently
// blocks movement. RL observation code uses this to represent cover and impassable objects in
// local occupancy patches without depending on mutable manager internals.
func (m *Manager) BlockingUnitSnapshots() []BlockingUnitSnapshot {
	if m == nil {
		return nil
	}

	blockers := make([]BlockingUnitSnapshot, 0)
	m.units.Range(func(current Unit) bool {
		if current == nil || !current.Alive() || !current.BlocksMovement() {
			return true
		}

		base := current.Base()
		tileX, tileY := base.TilePosition(m.world.TileSize())
		blockers = append(blockers, BlockingUnitSnapshot{
			UnitID:   current.UnitID(),
			Kind:     current.UnitKind(),
			Position: base.Position,
			TileX:    tileX,
			TileY:    tileY,
		})
		return true
	})
	if len(blockers) == 0 {
		return nil
	}

	return blockers
}
