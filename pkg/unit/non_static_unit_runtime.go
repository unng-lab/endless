package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

// Tick advances unit state by one game tick.
// Movement is intentionally split into two layers:
//   - logical movement jumps between tile anchors and is scheduled through sleepTime;
//   - visual movement is reconstructed later from visible-travel state during draw.
//
// This keeps path traversal deterministic while avoiding visible teleportation.
func (u *NonStaticUnit) Tick(gameTick int64) {
	if !u.Alive() {
		u.cancelTrackedOrders()
		u.clearQueuedMove()
		u.clearTravel()
		return
	}

	if u.sleepTime > 0 {
		return
	}

	u.lastUpdateTick = gameTick
	u.finishInterruptedMoveOrderAtTileBoundary()
	u.startQueuedOrderIfReady()
	if u.activeOrder.hasOrder && u.activeOrder.order.kind == OrderKindFire && u.activeOrder.releasing {
		if u.sleepTime == 0 {
			u.releasePreparedFireOrder()
		}
		u.travel.remaining = u.sleepTime
		return
	}

	u.promoteQueuedMoveIfReady()
	u.sleepTime = u.advance()
	u.completeMoveOrderIfFinished()
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

// SetPath replaces the current route with a copy of the provided path.
// Copying here prevents external code from mutating the active route after the command
// has been accepted, and resetting sleepTime lets the unit react on the next update.
func (u *NonStaticUnit) SetPath(path []geom.Point) {
	if !u.IsMobile() {
		u.cancelTrackedOrders()
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	u.cancelTrackedOrders()
	u.setPathWithoutOrderReset(path)
}

// QueueMoveCommand applies a new move order immediately only when the unit is already at the
// center of its current logical tile. If the unit is still finishing the current segment, the
// route is stored as the single pending command that will replace the old path later.
func (u *NonStaticUnit) QueueMoveCommand(path []geom.Point) {
	if !u.IsMobile() {
		u.cancelTrackedOrders()
		u.path = u.path[:0]
		u.clearQueuedMove()
		return
	}

	u.cancelTrackedOrders()
	if u.sleepTime > 0 {
		u.queueNextMove(path)
		return
	}

	u.setPathWithoutOrderReset(path)
}

// setPathWithoutOrderReset updates the immediate route without touching the tracked order
// lifecycle. Order-driven code uses this helper so it can install a path first and report the
// final status only when the path actually completes or gets interrupted later.
func (u *NonStaticUnit) setPathWithoutOrderReset(path []geom.Point) {
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
	u.cancelTrackedOrders()
	u.path = u.path[:0]
	u.sleepTime = 0
	u.fireCooldownRemaining = 0
	u.clearQueuedMove()
	u.clearTravel()
	u.MarkForRemoval()
}

// advanceWeaponCooldown spends one simulation tick from the remaining fire cooldown even when
// the unit is otherwise asleep between path steps. This keeps weapon readiness tied to real
// game ticks instead of only to active Tick callbacks.
func (u *NonStaticUnit) advanceWeaponCooldown() {
	if u == nil || u.fireCooldownRemaining <= 0 {
		return
	}

	u.fireCooldownRemaining--
}

func (u *NonStaticUnit) weaponReady() bool {
	return u != nil && u.fireCooldownRemaining == 0
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
