package unit

import "github.com/unng-lab/endless/pkg/geom"

const (
	fireOrderWindupTicks   = 8
	fireOrderCooldownTicks = 10
)

// OrderStatus describes one lifecycle transition of a unit order. Reports keep the full
// sequence so callers can distinguish acceptance, actual execution start and the final result.
type OrderStatus uint8

const (
	OrderQueued OrderStatus = iota
	OrderStarted
	OrderCompleted
	OrderFailed
	OrderCanceled
)

func (s OrderStatus) String() string {
	switch s {
	case OrderQueued:
		return "queued"
	case OrderStarted:
		return "started"
	case OrderCompleted:
		return "completed"
	case OrderFailed:
		return "failed"
	case OrderCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// OrderKind identifies which gameplay subsystem owns the order execution rules.
type OrderKind uint8

const (
	OrderKindMove OrderKind = iota
	OrderKindFire
)

func (k OrderKind) String() string {
	switch k {
	case OrderKindMove:
		return "move"
	case OrderKindFire:
		return "fire"
	default:
		return "unknown"
	}
}

// OrderReport is the actor-facing event emitted whenever an accepted order changes state.
// Move orders fill TargetPoint, fire orders fill Direction, and consumers may use Kind to
// decide which payload field is meaningful for one concrete report.
type OrderReport struct {
	OrderID     int64
	UnitID      int64
	Kind        OrderKind
	Status      OrderStatus
	TargetPoint geom.Point
	Direction   geom.Point
}

// moveOrder carries the final world-space destination the unit should reach after pathfinding
// plus the already resolved route snapshot that execution will later consume.
type moveOrder struct {
	ID          int64
	UnitID      int64
	TargetPoint geom.Point
	Path        []geom.Point
}

// fireOrder keeps the normalized fire direction that will later be used to build a projectile
// once the order actually starts executing.
type fireOrder struct {
	ID        int64
	UnitID    int64
	Direction geom.Point
}

type orderReportingUnit interface {
	drainOrderReports() []OrderReport
}

type projectileSpawningUnit interface {
	drainPendingProjectiles() []*Projectile
}

type unitOrder struct {
	id          int64
	unitID      int64
	kind        OrderKind
	targetPoint geom.Point
	direction   geom.Point
	path        []geom.Point
}

type queuedOrderState struct {
	order    unitOrder
	hasOrder bool
}

type activeOrderState struct {
	order     unitOrder
	hasOrder  bool
	started   bool
	releasing bool
}

// queueMoveOrder accepts one move order into the unit-local lifecycle. If another order is
// already waiting, only the newest queued order is kept because gameplay input should update
// intent rather than accumulate an unbounded backlog for one unit.
func (u *NonStaticUnit) queueMoveOrder(gameTick int64, order moveOrder) {
	u.enqueueOrder(gameTick, unitOrder{
		id:          order.ID,
		unitID:      order.UnitID,
		kind:        OrderKindMove,
		targetPoint: order.TargetPoint,
		path:        append([]geom.Point(nil), order.Path...),
	})
}

// queueFireOrder accepts one fire order into the same lifecycle as movement. The direction is
// stored as a defensive copy so future callers cannot mutate the queued execution intent.
func (u *NonStaticUnit) queueFireOrder(gameTick int64, order fireOrder) {
	u.enqueueOrder(gameTick, unitOrder{
		id:        order.ID,
		unitID:    order.UnitID,
		kind:      OrderKindFire,
		direction: order.Direction,
	})
}

func (u *NonStaticUnit) enqueueOrder(gameTick int64, order unitOrder) {
	u.emitOrderReport(OrderQueued, order)
	if u.debugRuntimeLogf != nil {
		if u.queuedOrder.hasOrder {
			u.debugRuntimeLogf(
				"queue unit=%d tick=%d next_order_id=%d next_kind=%s next_target=(%.1f, %.1f) next_direction=(%.3f, %.3f) next_path_waypoints=%d replaced_order_id=%d replaced_kind=%s active_order_id=%d active_kind=%s active_sleep=%d active_path_waypoints=%d",
				u.UnitID(),
				gameTick,
				order.id,
				order.kind.String(),
				order.targetPoint.X,
				order.targetPoint.Y,
				order.direction.X,
				order.direction.Y,
				len(order.path),
				u.queuedOrder.order.id,
				u.queuedOrder.order.kind.String(),
				u.activeOrder.order.id,
				u.activeOrder.order.kind.String(),
				u.sleepTime,
				len(u.path),
			)
		} else {
			u.debugRuntimeLogf(
				"queue unit=%d tick=%d next_order_id=%d next_kind=%s next_target=(%.1f, %.1f) next_direction=(%.3f, %.3f) next_path_waypoints=%d active_order_id=%d active_kind=%s active_sleep=%d active_path_waypoints=%d",
				u.UnitID(),
				gameTick,
				order.id,
				order.kind.String(),
				order.targetPoint.X,
				order.targetPoint.Y,
				order.direction.X,
				order.direction.Y,
				len(order.path),
				u.activeOrder.order.id,
				u.activeOrder.order.kind.String(),
				u.sleepTime,
				len(u.path),
			)
		}
	}
	if u.queuedOrder.hasOrder {
		u.emitOrderReport(OrderCanceled, u.queuedOrder.order)
	}

	u.queuedOrder.order = order
	u.queuedOrder.hasOrder = true
}

// drainOrderReports hands the manager a snapshot of all statuses the unit has emitted since
// the last drain. The defensive copy keeps worker updates and manager-side aggregation isolated.
func (u *NonStaticUnit) drainOrderReports() []OrderReport {
	if len(u.orderReports) == 0 {
		return nil
	}

	reports := append([]OrderReport(nil), u.orderReports...)
	u.orderReports = u.orderReports[:0]
	return reports
}

// drainPendingProjectiles returns the short list of projectiles that finished their fire-order
// wind-up during this tick. The manager materializes them after worker updates complete so the
// parallel unit update loop never mutates shared ordered-unit storage directly.
func (u *NonStaticUnit) drainPendingProjectiles() []*Projectile {
	if len(u.pendingProjectiles) == 0 {
		return nil
	}

	projectiles := append([]*Projectile(nil), u.pendingProjectiles...)
	u.pendingProjectiles = u.pendingProjectiles[:0]
	return projectiles
}

// finishInterruptedMoveOrderAtTileBoundary cancels the active move order only at the exact
// tile-boundary handoff where the unit may legally switch to the next queued command.
func (u *NonStaticUnit) finishInterruptedMoveOrderAtTileBoundary(gameTick int64) {
	if !u.activeOrder.hasOrder || u.activeOrder.order.kind != OrderKindMove {
		return
	}
	if !u.queuedOrder.hasOrder {
		return
	}
	if len(u.path) == 0 || u.sleepTime > 0 {
		return
	}

	if u.debugRuntimeLogf != nil {
		u.debugRuntimeLogf(
			"handoff unit=%d tick=%d reached=(%.1f, %.1f) cancel_active_order_id=%d queued_order_id=%d queued_kind=%s remaining_path_waypoints=%d",
			u.UnitID(),
			gameTick,
			u.Position.X,
			u.Position.Y,
			u.activeOrder.order.id,
			u.queuedOrder.order.id,
			u.queuedOrder.order.kind.String(),
			len(u.path),
		)
	}
	u.emitOrderReport(OrderCanceled, u.activeOrder.order)
	u.path = u.path[:0]
	u.clearActiveOrder()
}

// completeMoveOrderIfFinished closes the current move order once the unit has become fully
// idle again. Waiting for the idle state ensures completion is reported only after the last
// logical segment has already ended and no more route points remain.
func (u *NonStaticUnit) completeMoveOrderIfFinished() {
	if !u.activeOrder.hasOrder || u.activeOrder.order.kind != OrderKindMove {
		return
	}
	if u.Base().IsMoving() || len(u.path) > 0 {
		return
	}

	u.emitOrderReport(OrderCompleted, u.activeOrder.order)
	u.clearActiveOrder()
}

// startQueuedOrderIfReady promotes the latest queued order into the active slot at the moment
// the unit is allowed to begin a new gameplay action.
func (u *NonStaticUnit) startQueuedOrderIfReady(gameTick int64) {
	if u.activeOrder.hasOrder || !u.queuedOrder.hasOrder || u.sleepTime > 0 {
		return
	}

	if u.queuedOrder.order.kind == OrderKindFire && !u.weaponReady() {
		return
	}

	order := u.queuedOrder.order
	u.queuedOrder = queuedOrderState{}
	u.activeOrder = activeOrderState{
		order:    order,
		hasOrder: true,
		started:  true,
	}
	if u.debugRuntimeLogf != nil {
		u.debugRuntimeLogf(
			"start unit=%d tick=%d order_id=%d kind=%s target=(%.1f, %.1f) direction=(%.3f, %.3f) queued_path_waypoints=%d position=(%.1f, %.1f)",
			u.UnitID(),
			gameTick,
			order.id,
			order.kind.String(),
			order.targetPoint.X,
			order.targetPoint.Y,
			order.direction.X,
			order.direction.Y,
			len(order.path),
			u.Position.X,
			u.Position.Y,
		)
	}
	u.emitOrderReport(OrderStarted, order)

	switch order.kind {
	case OrderKindMove:
		u.path = append(u.path[:0], order.path...)
	case OrderKindFire:
		if !u.startFireOrder(order) {
			return
		}
	}
}

// startFireOrder prepares the projectile at the moment the fire order actually starts so the
// later release step can be reduced to handing the already validated projectile back to the manager.
func (u *NonStaticUnit) startFireOrder(order unitOrder) bool {
	if u.projectileBuilder == nil {
		u.emitOrderReport(OrderFailed, order)
		u.clearActiveOrder()
		return false
	}

	projectile, err := u.projectileBuilder(u, order.direction)
	if err != nil {
		u.emitOrderReport(OrderFailed, order)
		u.clearActiveOrder()
		return false
	}

	u.preparedProjectile = projectile
	u.activeOrder.releasing = true
	u.sleepTime = fireOrderWindupTicks
	u.travel.remaining = u.sleepTime
	return true
}

// releasePreparedFireOrder finishes the fire order by handing the already built projectile to
// the manager-side spawn buffer and then publishing the completed status.
func (u *NonStaticUnit) releasePreparedFireOrder() {
	if !u.activeOrder.hasOrder || u.activeOrder.order.kind != OrderKindFire || !u.activeOrder.releasing {
		return
	}

	if u.preparedProjectile != nil {
		u.pendingProjectiles = append(u.pendingProjectiles, u.preparedProjectile)
	}
	u.preparedProjectile = nil
	u.fireCooldownRemaining = fireOrderCooldownTicks
	u.emitOrderReport(OrderCompleted, u.activeOrder.order)
	u.clearActiveOrder()
}

// cancelTrackedOrders reports cancellation for every accepted order that can no longer finish,
// such as when the unit dies, respawns, or raw legacy path APIs forcibly replace the lifecycle-managed state.
func (u *NonStaticUnit) cancelTrackedOrders() {
	if u.activeOrder.hasOrder {
		u.emitOrderReport(OrderCanceled, u.activeOrder.order)
	}
	if u.queuedOrder.hasOrder {
		u.emitOrderReport(OrderCanceled, u.queuedOrder.order)
	}

	u.clearActiveOrder()
	u.queuedOrder = queuedOrderState{}
}

func (u *NonStaticUnit) clearActiveOrder() {
	u.activeOrder = activeOrderState{}
	u.preparedProjectile = nil
}

func (u *NonStaticUnit) emitOrderReport(status OrderStatus, order unitOrder) {
	u.orderReports = append(u.orderReports, OrderReport{
		OrderID:     order.id,
		UnitID:      order.unitID,
		Kind:        order.kind,
		Status:      status,
		TargetPoint: order.targetPoint,
		Direction:   order.direction,
	})
}
