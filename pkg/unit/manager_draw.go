package unit

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

const (
	infoPanelHeight   = 126.0
	infoPanelMargin   = 16.0
	infoPanelMaxWidth = 480.0
)

func (m *Manager) DrawOverlay(screen *ebiten.Image, cam *camera.Camera, screenWidth, screenHeight int) {
	if !m.selectedOnScreen() {
		return
	}

	m.drawSelectedHighlight(screen, cam)
	m.drawInfoPanel(screen, screenWidth, screenHeight)
}

func (m *Manager) PanelRect(screenWidth, screenHeight int) (geom.Rect, bool) {
	if !m.selectedOnScreen() {
		return geom.Rect{}, false
	}

	width := math.Min(float64(screenWidth)-infoPanelMargin*2, infoPanelMaxWidth)
	if width <= 0 {
		return geom.Rect{}, false
	}

	left := (float64(screenWidth) - width) / 2
	top := math.Max(float64(screenHeight)-infoPanelHeight-infoPanelMargin, infoPanelMargin)
	return geom.Rect{
		Min: geom.Point{X: left, Y: top},
		Max: geom.Point{X: left + width, Y: top + infoPanelHeight},
	}, true
}

func (m *Manager) drawSelectedHighlight(screen *ebiten.Image, cam *camera.Camera) {
	selected, ok := m.selectedUnit()
	if !ok || !selected.Base().OnScreen {
		return
	}

	rect, ok := unitScreenRect(cam, m.world.TileSize(), selected)
	if !ok {
		return
	}

	padding := math.Max(2, math.Round(cam.Scale()))
	border := math.Max(2, math.Round(cam.Scale()))
	left := rect.Min.X - padding
	top := rect.Min.Y - padding
	width := (rect.Max.X - rect.Min.X) + padding*2
	height := (rect.Max.Y - rect.Min.Y) + padding*2
	right := left + width
	bottom := top + height
	highlight := color.NRGBA{R: 255, G: 214, B: 102, A: 255}

	m.drawFilledRect(screen, left, top, width, border, highlight)
	m.drawFilledRect(screen, left, bottom-border, width, border, highlight)
	m.drawFilledRect(screen, left, top+border, border, math.Max(height-border*2, 0), highlight)
	m.drawFilledRect(screen, right-border, top+border, border, math.Max(height-border*2, 0), highlight)
}

func (m *Manager) drawInfoPanel(screen *ebiten.Image, screenWidth, screenHeight int) {
	selected, ok := m.selectedUnit()
	if !ok || !selected.Base().OnScreen {
		return
	}

	rect, ok := m.PanelRect(screenWidth, screenHeight)
	if !ok {
		return
	}

	panelColor := color.NRGBA{R: 19, G: 23, B: 30, A: 230}
	borderColor := color.NRGBA{R: 255, G: 214, B: 102, A: 255}
	shadowColor := color.NRGBA{R: 0, G: 0, B: 0, A: 90}

	m.drawFilledRect(screen, rect.Min.X+4, rect.Min.Y+4, rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y, shadowColor)
	m.drawFilledRect(screen, rect.Min.X, rect.Min.Y, rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y, panelColor)
	m.drawFilledRect(screen, rect.Min.X, rect.Min.Y, rect.Max.X-rect.Min.X, 3, borderColor)

	base := selected.Base()
	tileX, tileY := base.TilePosition(m.world.TileSize())
	infoText := fmt.Sprintf(
		"Object #%d: %s\nTile: (%d, %d)  World: (%.1f, %.1f)\nKind: %s  Frame: %d\nHP: %d/%d  Terrain speed: %.0f%%  Sleep: %d\n%s",
		selected.UnitID(),
		selected.Name(),
		tileX,
		tileY,
		base.Position.X,
		base.Position.Y,
		selected.UnitKind(),
		selected.Frame(),
		selected.CurrentHealth(),
		selected.MaxHealthValue(),
		m.world.TileType(tileX, tileY).SpeedMultiplier()*100,
		base.SleepTime(),
		m.statusText(selected),
	)
	ebitenutil.DebugPrintAt(screen, infoText, int(rect.Min.X+16), int(rect.Min.Y+14))
}

func (m *Manager) statusText(selected Unit) string {
	base := selected.Base()
	if !selected.IsMobile() {
		if selected.BlocksMovement() {
			return "State: static obstacle  Blocks movement: yes"
		}
		return "State: static obstacle"
	}

	if !base.IsMoving() {
		if selected.CanShoot() {
			return "State: idle  Weapon: ready"
		}
		return "State: idle"
	}

	destination, ok := base.Destination()
	if !ok {
		return "State: idle"
	}

	targetTileX := int(math.Floor(destination.X / m.world.TileSize()))
	targetTileY := int(math.Floor(destination.Y / m.world.TileSize()))
	if selected.CanShoot() {
		return fmt.Sprintf("State: moving  Target: (%d, %d)  Waypoints: %d  Weapon: ready", targetTileX, targetTileY, base.PathLen())
	}

	return fmt.Sprintf("State: moving  Target: (%d, %d)  Waypoints: %d", targetTileX, targetTileY, base.PathLen())
}

func (m *Manager) drawFilledRect(screen *ebiten.Image, x, y, width, height float64, fill color.Color) {
	if width <= 0 || height <= 0 {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(fill)

	screen.DrawImage(m.renderer.solid, &op)
}

func pointInRect(point geom.Point, rect geom.Rect) bool {
	return point.X >= rect.Min.X &&
		point.X <= rect.Max.X &&
		point.Y >= rect.Min.Y &&
		point.Y <= rect.Max.Y
}
