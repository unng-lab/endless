package unit

import (
	"math"

	"github.com/unng-lab/endless/internal/new/camera"
)

// State identifies unit animation state.
type State string

const (
	// StateIdle is used when the unit has no movement orders.
	StateIdle State = "idle"
	// StateMoving is used when the unit follows a path.
	StateMoving State = "moving"
)

// UnitType describes common data shared between units of the same archetype.
type UnitType struct {
	Name       string
	Speed      float64 // units per second
	Animations map[State]Animation
}

// Unit represents an entity on the map.
type Unit struct {
	Type      *UnitType
	Position  camera.Point
	state     State
	animTime  float64
	waypoints []camera.Point
}

// NewUnit creates a unit of the provided type at the given position.
func NewUnit(t *UnitType, position camera.Point) *Unit {
	return &Unit{Type: t, Position: position, state: StateIdle, waypoints: nil}
}

// State returns the current unit state.
func (u *Unit) State() State {
	return u.state
}

// SetPath sets the path the unit should follow expressed in world coordinates.
func (u *Unit) SetPath(points []camera.Point) {
	u.waypoints = append([]camera.Point(nil), points...)
	if len(u.waypoints) > 1 {
		u.state = StateMoving
	} else {
		u.state = StateIdle
	}
}

// ClearPath stops the unit immediately.
func (u *Unit) ClearPath() {
	u.waypoints = nil
	u.state = StateIdle
}

// Update advances the unit state by delta seconds.
func (u *Unit) Update(delta float64) {
	if u.Type == nil {
		return
	}

	u.animTime += delta

	if len(u.waypoints) == 0 {
		u.state = StateIdle
		return
	}

	target := u.waypoints[0]
	dx := target.X - u.Position.X
	dy := target.Y - u.Position.Y
	dist := distance(dx, dy)
	if dist == 0 {
		u.Position = target
		u.waypoints = u.waypoints[1:]
		if len(u.waypoints) == 0 {
			u.state = StateIdle
		}
		return
	}

	step := u.Type.Speed * delta
	if step >= dist {
		u.Position = target
		u.waypoints = u.waypoints[1:]
		if len(u.waypoints) == 0 {
			u.state = StateIdle
		}
		return
	}

	u.state = StateMoving
	factor := step / dist
	u.Position.X += dx * factor
	u.Position.Y += dy * factor
}

// FrameIndex returns the atlas tile index for the current animation state.
func (u *Unit) FrameIndex() int {
	if u.Type == nil {
		return 0
	}
	anim, ok := u.Type.Animations[u.state]
	if !ok {
		anim = u.Type.Animations[StateIdle]
	}
	frames := anim.Frames
	if len(frames) == 0 {
		return 0
	}
	return frames[anim.frameAt(u.animTime)]
}

func distance(dx, dy float64) float64 {
	return math.Hypot(dx, dy)
}
