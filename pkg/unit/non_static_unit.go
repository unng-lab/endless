package unit

import "github.com/unng-lab/endless/pkg/geom"

type NonStaticUnit struct {
	BaseUnit
	ID            int64
	SpawnPosition geom.Point
	Kind          Kind
	MaxHealth     int
	Health        int

	animation             Animation
	animationTicks        int
	moveSpeedPerTick      float64
	speedAt               func(geom.Point) float64
	fireCooldownRemaining int

	projectileBuilder func(*NonStaticUnit, geom.Point) (*Projectile, error)
	debugRuntimeLogf  func(string, ...any)

	queuedMove         queuedMoveCommand
	activeOrder        activeOrderState
	queuedOrder        queuedOrderState
	orderReports       []OrderReport
	preparedProjectile *Projectile
	pendingProjectiles []*Projectile
}

type travelState struct {
	from            geom.Point
	to              geom.Point
	duration        int
	remaining       int
	visualRemaining int
	active          bool
}

// queuedMoveCommand keeps at most one deferred move order for a mobile unit.
// While the unit is still interpolating into the center of the current tile, repeated move
// commands only replace this pending route instead of interrupting the active segment.
type queuedMoveCommand struct {
	path     []geom.Point
	hasRoute bool
}

// NewRunner builds one mobile unit whose movement speed is expressed directly in world
// units per simulation tick. The animation offset uses the same tick scale so callers may
// stagger the initial frame without any extra unit conversion.
func NewRunner(position geom.Point, focused bool, animationTickOffset int) *NonStaticUnit {
	kind := KindRunner
	if focused {
		kind = KindRunnerFocused
	}

	return &NonStaticUnit{
		BaseUnit: BaseUnit{
			Position: position,
		},
		SpawnPosition:    position,
		Kind:             kind,
		MaxHealth:        3,
		Health:           3,
		animation:        runnerAnimation,
		animationTicks:   normalizeAnimationOffset(animationTickOffset, runnerAnimation),
		moveSpeedPerTick: 0.8,
	}
}

func (u *NonStaticUnit) Base() *BaseUnit {
	return &u.BaseUnit
}

func (u *NonStaticUnit) UnitID() int64 {
	return u.ID
}

func (u *NonStaticUnit) SetUnitID(id int64) {
	u.ID = id
}

func (u *NonStaticUnit) UnitKind() Kind {
	return u.Kind
}

func (u *NonStaticUnit) Frame() int {
	return u.animation.frameAt(u.animationTicks)
}

func (u *NonStaticUnit) Name() string {
	switch u.Kind {
	case KindRunner:
		return "Runner"
	case KindRunnerFocused:
		return "Runner Focused"
	default:
		return string(u.Kind)
	}
}

func (u *NonStaticUnit) IsMobile() bool {
	return u.Alive() && u.moveSpeedPerTick > 0
}

func (u *NonStaticUnit) BlocksMovement() bool {
	return false
}

func (u *NonStaticUnit) CanShoot() bool {
	if !u.Alive() {
		return false
	}

	switch u.Kind {
	case KindRunner, KindRunnerFocused:
		return true
	default:
		return false
	}
}

// WeaponReady reports whether the unit may start executing a queued fire order on this tick.
// The check intentionally stays separate from CanShoot so callers may still queue a future
// fire order while the current weapon cooldown is counting down.
func (u *NonStaticUnit) WeaponReady() bool {
	return u != nil && u.CanShoot() && u.weaponReady()
}

func (u *NonStaticUnit) Alive() bool {
	return u.Health > 0
}

func (u *NonStaticUnit) CurrentHealth() int {
	return u.Health
}

func (u *NonStaticUnit) MaxHealthValue() int {
	return u.MaxHealth
}

func (u *NonStaticUnit) HealthRatio() float64 {
	if u.MaxHealth <= 0 {
		return 0
	}

	return geom.ClampFloat(float64(u.Health)/float64(u.MaxHealth), 0, 1)
}

func (u *NonStaticUnit) ApplyDamage(amount int) bool {
	if amount <= 0 || !u.Alive() {
		return false
	}

	u.Health -= amount
	if u.Health > 0 {
		return false
	}

	u.Health = 0
	u.prepareForRemovalAfterDeath()
	return true
}

func (u *NonStaticUnit) Respawn() {
	u.cancelTrackedOrders()
	u.Position = u.SpawnPosition
	u.Health = u.MaxHealth
	u.path = u.path[:0]
	u.sleepTime = 0
	u.fireCooldownRemaining = 0
	u.clearQueuedMove()
	u.clearTravel()
	u.ClearRemovalMark()
}

func (u *NonStaticUnit) Selectable() bool {
	return u.Alive()
}

func (u *NonStaticUnit) EnterTile(stack *TileStack) {
	stack.AddUnit(u.UnitID())
}

func (u *NonStaticUnit) LeaveTile(stack *TileStack) {
	stack.RemoveUnit(u.UnitID())
}

// Wake clears the external-sleep flag without touching the active travel sleep budget.
// Mobile units use sleepTime to model an in-flight move segment, so resetting it here would
// incorrectly fast-forward movement when UI or scenario code merely wants the unit selected.
func (u *NonStaticUnit) Wake() {
	u.WakeForUpdate()
}

// SetSpeedMultiplierLookup binds the terrain-speed resolver once so Tick may stay on the
// requested minimal contract and receive only the current simulation tick.
func (u *NonStaticUnit) SetSpeedMultiplierLookup(speedAt func(geom.Point) float64) {
	u.speedAt = speedAt
}

// SetProjectileBuilder binds the manager-owned projectile factory once so delayed fire orders
// can prepare their projectile exactly when execution starts without depending on manager state.
func (u *NonStaticUnit) SetProjectileBuilder(builder func(*NonStaticUnit, geom.Point) (*Projectile, error)) {
	u.projectileBuilder = builder
}

// SetDebugRuntimeLogger binds a manager-owned debug sink that the unit may call while it
// mutates its internal queue, promotes queued orders, or starts the next move segment.
func (u *NonStaticUnit) SetDebugRuntimeLogger(logger func(string, ...any)) {
	u.debugRuntimeLogf = logger
}
