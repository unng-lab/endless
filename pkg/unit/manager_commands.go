package unit

import (
	"fmt"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

func (m *Manager) HasSelected() bool {
	_, ok := m.selectedUnit()
	return ok
}

// SelectUnitByID pins one concrete runtime object into the existing overlay and command-target
// flow so autonomous scenarios may keep the same actor selected without requiring a mouse click.
func (m *Manager) SelectUnitByID(unitID int64) bool {
	if m == nil || unitID == 0 {
		m.selectedID = 0
		return false
	}

	selected, ok := m.unitByID(unitID)
	if !ok || selected == nil {
		m.selectedID = 0
		return false
	}

	m.selectedID = unitID
	if body, ok := selected.(*NonStaticUnit); ok {
		body.Wake()
	}
	return true
}

func (m *Manager) SelectAtScreen(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) {
	if m.PointInPanel(cam, cursor, screenWidth, screenHeight) {
		return
	}
	if cam == nil {
		m.selectedID = 0
		return
	}

	worldPos := cam.ScreenToWorld(cursor)
	if !m.pointInWorld(worldPos) {
		m.selectedID = 0
		return
	}

	stack := m.stackAtWorldPoint(worldPos)
	candidates := m.selectableUnitsFromStack(stack)
	if len(candidates) == 0 {
		m.selectedID = 0
		return
	}

	selected := candidates[len(candidates)-1]
	m.selectedID = selected.UnitID()
	if body, ok := selected.(*NonStaticUnit); ok {
		body.Wake()
	}
}

func (m *Manager) PointInPanel(cam *camera.Camera, cursor geom.Point, screenWidth, screenHeight int) bool {
	rect, ok := m.PanelRect(cam, screenWidth, screenHeight)
	return ok && pointInRect(cursor, rect)
}

func (m *Manager) CommandSelectedMove(targetTileX, targetTileY int) error {
	selected, ok := m.selectedUnit()
	if !ok {
		if m.HasSelected() {
			return fmt.Errorf("selected object is immobile")
		}
		return nil
	}

	return m.IssueMoveOrder(selected.UnitID(), m.tileAnchor(targetTileX, targetTileY))
}

func (m *Manager) CommandSelectedFire(target geom.Point) error {
	selected, ok := m.selectedNonStatic()
	if !ok {
		if m.HasSelected() {
			return fmt.Errorf("selected object cannot shoot")
		}
		return nil
	}
	if !selected.CanShoot() {
		return fmt.Errorf("unit %q cannot shoot", selected.Name())
	}

	return m.IssueFireOrder(selected.UnitID(), geom.Point{
		X: target.X - selected.Position.X,
		Y: target.Y - selected.Position.Y,
	})
}

func (m *Manager) Selected() (Unit, bool) {
	return m.selectedUnit()
}

func (m *Manager) selectedUnit() (Unit, bool) {
	if m.selectedID == 0 {
		return nil, false
	}

	return m.unitByID(m.selectedID)
}

func (m *Manager) selectedNonStatic() (*NonStaticUnit, bool) {
	selected, ok := m.selectedUnit()
	if !ok {
		return nil, false
	}

	body, ok := selected.(*NonStaticUnit)
	return body, ok
}

func (m *Manager) selectedVisible(cam *camera.Camera, screenWidth, screenHeight int) bool {
	selected, ok := m.selectedUnit()
	if !ok {
		return false
	}

	return unitVisibleOnScreen(cam, m.world.TileSize(), screenWidth, screenHeight, selected)
}

func (m *Manager) unitByID(unitID int64) (Unit, bool) {
	return m.units.Get(unitID)
}

func (m *Manager) pointInWorld(position geom.Point) bool {
	return position.X >= 0 &&
		position.Y >= 0 &&
		position.X <= m.world.Width() &&
		position.Y <= m.world.Height()
}
