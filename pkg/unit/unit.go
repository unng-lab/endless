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
	FrameCount: 8,
	FrameTicks: 6,
}

type NonStaticUnit struct {
	BaseUnit
	ID            int64
	SpawnPosition geom.Point
	Kind          Kind
	MaxHealth     int
	Health        int

	animation        Animation
	animationTicks   int
	moveSpeedPerTick float64
	speedAt          func(geom.Point) float64

	queuedMove queuedMoveCommand
	moveJob    moveJobState
	jobReports []JobReport
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

// Tick advances unit state by one game tick.
// Movement is intentionally split into two layers:
//   - logical movement jumps between tile anchors and is scheduled through sleepTime;
//   - visual movement is reconstructed later from visible-travel state during draw.
//
// This keeps path traversal deterministic while avoiding visible teleportation.
func (u *NonStaticUnit) Tick(gameTick int64) {
	if !u.Alive() {
		u.failAssignedMoveJob()
		u.clearQueuedMove()
		u.clearTravel()
		return
	}

	if u.sleepTime > 0 {
		return
	}

	u.lastUpdateTick = gameTick
	u.promoteQueuedMoveIfReady()
	u.sleepTime = u.advance()
	u.completeAssignedMoveJobIfFinished()
	u.travel.remaining = u.sleepTime
}

// UpdateVisible advances only the render-facing state for a visible unit. The manager calls
// this while iterating visible tile stacks so animation and interpolated travel progress only
// spend work on units that may actually be drawn this frame.
func (u *NonStaticUnit) UpdateVisible(gameTick int64) {
	if u == nil || u.lastVisibleTick == gameTick {
		return
	}

	u.animationTicks++
	u.AdvanceVisibleTravel(gameTick)
}

// ShouldUpdate keeps mobile bodies inside the regular update loop every frame so movement,
// animation timers and queued commands continue progressing deterministically.
func (u *NonStaticUnit) ShouldUpdate() bool {
	return true
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

// SetPath replaces the current route with a copy of the provided path.
// Copying here prevents external code from mutating the active route after the command
// has been accepted, and resetting sleepTime lets the unit react on the next update.
func (u *NonStaticUnit) SetPath(path []geom.Point) {
	if !u.IsMobile() {
		u.failAssignedMoveJob()
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	u.failAssignedMoveJob()
	u.setPathWithoutJobCancel(path)
}

// QueueMoveCommand applies a new move order immediately only when the unit is already at the
// center of its current logical tile. If the unit is still finishing the current segment, the
// route is stored as the single pending command that will replace the old path later.
func (u *NonStaticUnit) QueueMoveCommand(path []geom.Point) {
	if !u.IsMobile() {
		u.failAssignedMoveJob()
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	u.failAssignedMoveJob()
	if u.sleepTime > 0 {
		u.queueNextMove(path)
		return
	}

	u.setPathWithoutJobCancel(path)
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
	u.failAssignedMoveJob()
	u.Position = u.SpawnPosition
	u.Health = u.MaxHealth
	u.path = u.path[:0]
	u.sleepTime = 0
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

func (u *NonStaticUnit) Wake() {
	u.WakeForUpdate()
	u.sleepTime = 0
}

// SetSpeedMultiplierLookup binds the terrain-speed resolver once so Tick may stay on the
// requested minimal contract and receive only the current simulation tick.
func (u *NonStaticUnit) SetSpeedMultiplierLookup(speedAt func(geom.Point) float64) {
	u.speedAt = speedAt
}

// setPathWithoutJobCancel updates the immediate route without touching the current move-job
// bookkeeping. Job-driven code uses this helper so it can install a path first and report the
// final status only when the path actually completes or fails later.
func (u *NonStaticUnit) setPathWithoutJobCancel(path []geom.Point) {
	u.path = append(u.path[:0], path...)
	u.clearQueuedMove()
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

// prepareForRemovalAfterDeath cancels any gameplay state that should not outlive a dead unit.
// The manager later removes the unit from tiles and ordered storage during its deferred sweep.
func (u *NonStaticUnit) prepareForRemovalAfterDeath() {
	u.failAssignedMoveJob()
	u.path = u.path[:0]
	u.sleepTime = 0
	u.clearQueuedMove()
	u.clearTravel()
	u.MarkForRemoval()
}

// advance schedules movement to the next reachable waypoint and returns how many ticks the
// unit should stay asleep before the next logical update. Returning a sleep budget instead
// of applying continuous movement each frame keeps all units aligned to the fixed game tick.
func (u *NonStaticUnit) advance() int {
	if len(u.path) == 0 || u.moveSpeedPerTick <= 0 {
		u.clearTravel()
		return 0
	}

	for len(u.path) > 0 {
		target := u.path[0]
		if u.consumeReachedWaypoint(target) {
			continue
		}

		currentSpeed, ok := u.moveSpeedAtCurrentTile()
		if !ok {
			u.clearTravel()
			return 0
		}

		return u.startTravel(target, currentSpeed)
	}

	u.clearTravel()
	return 0
}

// moveSpeedAtCurrentTile resolves the effective movement speed for the tile the unit is
// currently standing on. The returned value is expressed in world units per tick so advance
// may convert distance directly into a discrete sleep budget with no additional time-unit conversion.
func (u *NonStaticUnit) moveSpeedAtCurrentTile() (float64, bool) {
	currentSpeed := u.moveSpeedPerTick
	if u.speedAt == nil {
		return currentSpeed, currentSpeed > 0
	}

	multiplier := u.speedAt(u.Position)
	if multiplier <= 0 {
		return 0, false
	}

	currentSpeed *= multiplier
	return currentSpeed, currentSpeed > 0
}

// startTravel snapshots the segment that render interpolation should visualize, then moves
// the logical position directly to the next waypoint. This split lets pathfinding and tile
// occupancy observe the new cell immediately while drawing still shows continuous motion.
func (u *NonStaticUnit) startTravel(target geom.Point, currentSpeed float64) int {
	dx := target.X - u.Position.X
	dy := target.Y - u.Position.Y
	distance := math.Hypot(dx, dy)
	travelTicks := travelTicksForDistance(distance, currentSpeed)

	u.travel = travelState{
		from:            u.RenderPosition(),
		to:              target,
		duration:        travelTicks,
		remaining:       travelTicks,
		visualRemaining: travelTicks,
		active:          true,
	}
	u.Position = target
	u.path = u.path[1:]

	return travelTicks
}

// travelTicksForDistance converts a segment length and a tick-based speed into the minimum
// number of update ticks required to complete the segment. Ceil is important here: when the
// distance does not divide evenly by the per-tick speed, the extra partial tick keeps visual
// interpolation from finishing before the logical travel budget has been exhausted.
func travelTicksForDistance(distance, speedPerTick float64) int {
	if distance <= 0 || speedPerTick <= 0 {
		return 0
	}

	ticks := int(math.Ceil(distance / speedPerTick))
	if ticks < 1 {
		return 1
	}
	return ticks
}

// normalizeAnimationOffset folds an arbitrary initial tick offset into the animation cycle so
// callers may stagger spawned runners without having to manage the cycle length themselves.
func normalizeAnimationOffset(animationTickOffset int, animation Animation) int {
	if animation.FrameCount <= 0 || animation.FrameTicks <= 0 {
		return 0
	}

	cycleTicks := animation.FrameCount * animation.FrameTicks
	if cycleTicks <= 0 {
		return 0
	}

	offset := animationTickOffset % cycleTicks
	if offset < 0 {
		offset += cycleTicks
	}

	return offset
}
