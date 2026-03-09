package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

type Kind string

const (
	KindRunner        Kind = "runner"
	KindRunnerFocused Kind = "runnerfocused"
	KindWall          Kind = "wall"
	KindBarricade     Kind = "barricade"
	KindProjectile    Kind = "projectile"
)

var runnerAnimation = Animation{
	FrameCount:    8,
	FrameDuration: 0.1,
}

type NonStaticUnit struct {
	BaseUnit
	ID            int64
	SpawnPosition geom.Point
	Kind          Kind
	MaxHealth     int
	Health        int

	animation Animation
	elapsed   float64
	moveSpeed float64

	queuedMove queuedMoveCommand
}

type travelState struct {
	from      geom.Point
	to        geom.Point
	duration  int
	remaining int
	active    bool
}

// queuedMoveCommand keeps at most one deferred move order for a mobile unit.
// While the unit is still interpolating into the center of the current tile, repeated move
// commands only replace this pending route instead of interrupting the active segment.
type queuedMoveCommand struct {
	path     []geom.Point
	hasRoute bool
}

func NewRunner(position geom.Point, focused bool, phase float64) *NonStaticUnit {
	kind := KindRunner
	if focused {
		kind = KindRunnerFocused
	}

	return &NonStaticUnit{
		BaseUnit: BaseUnit{
			Position: position,
		},
		SpawnPosition: position,
		Kind:          kind,
		MaxHealth:     3,
		Health:        3,
		animation:     runnerAnimation,
		elapsed:       phase,
		moveSpeed:     48,
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

// Tick advances unit state by one game tick.
// Movement is intentionally split into two layers:
//   - logical movement jumps between tile anchors and is scheduled through sleepTime;
//   - visual movement is reconstructed from travel so rendering can interpolate smoothly.
//
// This keeps path traversal deterministic while avoiding visible teleportation.
func (u *NonStaticUnit) Tick(gameTick int64, delta float64, speedMultiplier func(geom.Point) float64) {
	if !u.Alive() {
		u.clearQueuedMove()
		u.clearTravel()
		return
	}

	u.elapsed += delta
	if u.sleepTime > 0 {
		u.sleepTime--
		u.travel.remaining = u.sleepTime
		if u.sleepTime == 0 {
			u.travel.remaining = 0
		}
		return
	}

	u.lastUpdateTick = gameTick
	u.promoteQueuedMoveIfReady()
	u.sleepTime = u.advance(delta, speedMultiplier)
	u.travel.remaining = u.sleepTime
}

// ShouldUpdate keeps mobile bodies inside the regular update loop every frame so movement,
// animation timers and queued commands continue progressing deterministically.
func (u *NonStaticUnit) ShouldUpdate() bool {
	return true
}

func (u *NonStaticUnit) Frame() int {
	return u.animation.frameAt(u.elapsed)
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

// SetPath replaces the current route with a copy of the provided path.
// Copying here prevents external code from mutating the active route after the command
// has been accepted, and resetting sleepTime lets the unit react on the next update.
func (u *NonStaticUnit) SetPath(path []geom.Point) {
	if !u.IsMobile() {
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	u.path = append(u.path[:0], path...)
	u.clearQueuedMove()
	u.sleepTime = 0
}

// QueueMoveCommand applies a new move order immediately only when the unit is already at the
// center of its current logical tile. If the unit is still finishing the current segment, the
// route is stored as the single pending command that will replace the old path later.
func (u *NonStaticUnit) QueueMoveCommand(path []geom.Point) {
	if !u.IsMobile() {
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	if u.sleepTime > 0 {
		u.queueNextMove(path)
		return
	}

	u.SetPath(path)
}

func (u *NonStaticUnit) IsMobile() bool {
	return u.Alive() && u.moveSpeed > 0
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
	return u.Health <= 0
}

func (u *NonStaticUnit) Respawn() {
	u.Position = u.SpawnPosition
	u.Health = u.MaxHealth
	u.path = u.path[:0]
	u.sleepTime = 0
	u.clearQueuedMove()
	u.clearTravel()
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

func (u *NonStaticUnit) Wake() {
	u.WakeForUpdate()
	u.sleepTime = 0
}

// promoteQueuedMoveIfReady swaps in the latest deferred move order once the previous travel
// segment has fully completed and the unit is allowed to accept a new movement command.
func (u *NonStaticUnit) promoteQueuedMoveIfReady() {
	if u.sleepTime > 0 || !u.queuedMove.hasRoute {
		return
	}

	u.path = append(u.path[:0], u.queuedMove.path...)
	u.clearQueuedMove()
}

// queueNextMove stores a defensive copy of the next move order. Keeping only one pending
// route matches the input rule that repeated clicks during travel should update, not stack,
// the upcoming command.
func (u *NonStaticUnit) queueNextMove(path []geom.Point) {
	u.queuedMove.path = append(u.queuedMove.path[:0], path...)
	u.queuedMove.hasRoute = true
}

func (u *NonStaticUnit) clearQueuedMove() {
	u.queuedMove.path = u.queuedMove.path[:0]
	u.queuedMove.hasRoute = false
}

// advance schedules movement to the next reachable waypoint and returns how many ticks the
// unit should stay asleep before the next logical update. Returning a sleep budget instead
// of applying continuous movement each frame keeps all units aligned to the fixed game tick.
func (u *NonStaticUnit) advance(delta float64, speedMultiplier func(geom.Point) float64) int {
	if delta <= 0 || len(u.path) == 0 || u.moveSpeed <= 0 {
		u.clearTravel()
		return 0
	}

	for len(u.path) > 0 {
		target := u.path[0]
		if u.consumeReachedWaypoint(target) {
			continue
		}

		currentSpeed, ok := u.moveSpeedAtCurrentTile(speedMultiplier)
		if !ok {
			u.clearTravel()
			return 0
		}

		return u.startTravel(target, currentSpeed, delta)
	}

	u.clearTravel()
	return 0
}

// moveSpeedAtCurrentTile resolves the effective movement speed for the tile the unit is
// currently standing on. Terrain modifiers are applied here once so advance can stay focused
// on route progression and travel scheduling.
func (u *NonStaticUnit) moveSpeedAtCurrentTile(speedMultiplier func(geom.Point) float64) (float64, bool) {
	currentSpeed := u.moveSpeed
	if speedMultiplier == nil {
		return currentSpeed, currentSpeed > 0
	}

	multiplier := speedMultiplier(u.Position)
	if multiplier <= 0 {
		return 0, false
	}

	currentSpeed *= multiplier
	return currentSpeed, currentSpeed > 0
}

// startTravel snapshots the segment that render interpolation should visualize, then moves
// the logical position directly to the next waypoint. This split lets pathfinding and tile
// occupancy observe the new cell immediately while drawing still shows continuous motion.
func (u *NonStaticUnit) startTravel(target geom.Point, currentSpeed, delta float64) int {
	dx := target.X - u.Position.X
	dy := target.Y - u.Position.Y
	distance := math.Hypot(dx, dy)
	travelTicks := sleepTicks(distance/currentSpeed, delta)

	u.travel = travelState{
		from:      u.RenderPosition(),
		to:        target,
		duration:  travelTicks,
		remaining: travelTicks,
		active:    true,
	}
	u.Position = target
	u.path = u.path[1:]

	return travelTicks
}

// sleepTicks converts a continuous duration into a minimum number of simulation ticks.
// Ceil is important here: when travel does not divide evenly by delta, we must reserve the
// extra partial tick so render interpolation never finishes before the logical move does.
func sleepTicks(duration, delta float64) int {
	if duration <= 0 {
		return 0
	}
	if delta <= 0 {
		return 1
	}

	ticks := int(math.Ceil(duration / delta))
	if ticks < 1 {
		return 1
	}
	return ticks
}
