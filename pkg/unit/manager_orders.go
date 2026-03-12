package unit

import (
	"fmt"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/pathfinding"
	"github.com/unng-lab/endless/pkg/world"
)

// IssueMoveOrder accepts a movement order for one concrete unit. The manager resolves the
// route immediately so later execution can start from a stable path snapshot even if callers
// issue another order before the unit reaches its next tile-boundary handoff point.
func (m *Manager) IssueMoveOrder(unitID int64, targetPoint geom.Point) error {
	current, ok := m.unitByID(unitID)
	if !ok {
		report := m.failedMoveOrderReport(unitID, targetPoint)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("unit %d not found", unitID)
	}

	body, ok := current.(*NonStaticUnit)
	if !ok || !body.IsMobile() {
		report := m.failedMoveOrderReport(unitID, targetPoint)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("unit %d is immobile", unitID)
	}

	targetTileX, targetTileY, ok := m.worldPointToTile(targetPoint)
	if !ok {
		report := m.failedMoveOrderReport(unitID, targetPoint)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("target point %+v is outside the world", targetPoint)
	}

	canonicalTarget := m.tileAnchor(targetTileX, targetTileY)
	startTileX, startTileY := body.Base().TilePosition(m.world.TileSize())
	grid := worldGrid{world: m.world}
	path, err := pathfinding.FindPath(
		grid,
		pathfinding.Step{X: startTileX, Y: startTileY},
		pathfinding.Step{X: targetTileX, Y: targetTileY},
	)
	if err != nil {
		m.appendBufferedOrderReport(m.failedMoveOrderReport(unitID, canonicalTarget))
		return err
	}

	body.queueMoveOrder(moveOrder{
		ID:          m.nextIssuedOrderID(),
		UnitID:      unitID,
		TargetPoint: canonicalTarget,
		Path:        m.worldPath(path),
	})
	return nil
}

// IssueFireOrder accepts one delayed fire command. The direction is normalized up front so
// queued reports and later execution use the same canonical direction vector.
func (m *Manager) IssueFireOrder(unitID int64, direction geom.Point) error {
	current, ok := m.unitByID(unitID)
	if !ok {
		report := m.failedFireOrderReport(unitID, direction)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("unit %d not found", unitID)
	}

	body, ok := current.(*NonStaticUnit)
	if !ok || !body.CanShoot() {
		report := m.failedFireOrderReport(unitID, direction)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("unit %d cannot shoot", unitID)
	}

	normalizedDirection, ok := normalizeDirection(direction)
	if !ok {
		report := m.failedFireOrderReport(unitID, direction)
		m.appendBufferedOrderReport(report)
		return fmt.Errorf("direction %+v is too small", direction)
	}

	body.queueFireOrder(fireOrder{
		ID:        m.nextIssuedOrderID(),
		UnitID:    unitID,
		Direction: normalizedDirection,
	})
	return nil
}

// DrainUnitOrderReports returns every order lifecycle event currently associated with one
// concrete unit. Reports normally stay owned by the unit itself, but the manager keeps a
// buffered tail for acceptance-time failures or for units that were already removed before
// external code had a chance to read their last reported state.
func (m *Manager) DrainUnitOrderReports(unitID int64) []OrderReport {
	if m == nil || unitID == 0 {
		return nil
	}

	reports := m.drainBufferedOrderReports(unitID)
	current, ok := m.unitByID(unitID)
	if !ok {
		if len(reports) == 0 {
			return nil
		}
		return reports
	}

	reporter, ok := current.(orderReportingUnit)
	if !ok {
		if len(reports) == 0 {
			return nil
		}
		return reports
	}

	reports = append(reports, reporter.drainOrderReports()...)
	if len(reports) == 0 {
		return nil
	}

	return reports
}

func (m *Manager) failedMoveOrderReport(unitID int64, targetPoint geom.Point) OrderReport {
	return OrderReport{
		OrderID:     m.nextIssuedOrderID(),
		UnitID:      unitID,
		Kind:        OrderKindMove,
		Status:      OrderFailed,
		TargetPoint: targetPoint,
	}
}

func (m *Manager) failedFireOrderReport(unitID int64, direction geom.Point) OrderReport {
	return OrderReport{
		OrderID:   m.nextIssuedOrderID(),
		UnitID:    unitID,
		Kind:      OrderKindFire,
		Status:    OrderFailed,
		Direction: direction,
	}
}

func (m *Manager) nextIssuedOrderID() int64 {
	if m == nil {
		return 0
	}

	m.nextOrderID++
	return m.nextOrderID
}

func (m *Manager) worldPointToTile(position geom.Point) (int, int, bool) {
	if !m.pointInWorld(position) {
		return 0, 0, false
	}

	tileX := int(math.Floor(position.X / m.world.TileSize()))
	tileY := int(math.Floor(position.Y / m.world.TileSize()))
	if !m.world.InBounds(tileX, tileY) {
		return 0, 0, false
	}

	return tileX, tileY, true
}

func (m *Manager) tileAnchor(tileX, tileY int) geom.Point {
	return geom.Point{
		X: (float64(tileX) + 0.5) * m.world.TileSize(),
		Y: (float64(tileY) + 0.5) * m.world.TileSize(),
	}
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

// appendBufferedOrderReports stores reports in manager-owned memory for cases where no live
// unit can serve them directly anymore, such as acceptance failures or unit removal.
func (m *Manager) appendBufferedOrderReports(unitID int64, reports []OrderReport) {
	if m == nil || unitID == 0 || len(reports) == 0 {
		return
	}

	m.orderReportsMu.Lock()
	m.bufferedOrderReports[unitID] = append(m.bufferedOrderReports[unitID], reports...)
	m.orderReportsMu.Unlock()
}

func (m *Manager) appendBufferedOrderReport(report OrderReport) {
	if m == nil || report.UnitID == 0 {
		return
	}

	m.appendBufferedOrderReports(report.UnitID, []OrderReport{report})
}

// drainBufferedOrderReports returns and clears the manager-owned report tail for one unit.
// Keeping this logic separate from DrainUnitOrderReports makes the lock scope explicit and small.
func (m *Manager) drainBufferedOrderReports(unitID int64) []OrderReport {
	if m == nil || unitID == 0 {
		return nil
	}

	m.orderReportsMu.Lock()
	defer m.orderReportsMu.Unlock()

	reports := m.bufferedOrderReports[unitID]
	if len(reports) == 0 {
		return nil
	}

	delete(m.bufferedOrderReports, unitID)
	return append([]OrderReport(nil), reports...)
}

func normalizeDirection(direction geom.Point) (geom.Point, bool) {
	length := math.Hypot(direction.X, direction.Y)
	if length <= 1e-6 {
		return geom.Point{}, false
	}

	return geom.Point{
		X: direction.X / length,
		Y: direction.Y / length,
	}, true
}

type worldGrid struct {
	world world.World
}

func (g worldGrid) InBounds(x, y int) bool {
	return g.world.InBounds(x, y)
}

func (g worldGrid) Cost(x, y int) float64 {
	if !g.InBounds(x, y) {
		return 0
	}

	return g.world.TileType(x, y).MovementCost()
}
