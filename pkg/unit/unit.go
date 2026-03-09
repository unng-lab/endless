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
)

var runnerAnimation = Animation{
	FrameCount:    8,
	FrameDuration: 0.1,
}

type Unit struct {
	ID            int64
	Position      geom.Point
	SpawnPosition geom.Point
	Kind          Kind
	OnScreen      bool
	MaxHealth     int
	Health        int

	animation Animation
	elapsed   float64
	moveSpeed float64
	path      []geom.Point

	sleepTime      int
	lastUpdateTick int64
	travel         travelState
}

type travelState struct {
	from      geom.Point
	to        geom.Point
	duration  int
	remaining int
	active    bool
}

func NewRunner(position geom.Point, focused bool, phase float64) Unit {
	kind := KindRunner
	if focused {
		kind = KindRunnerFocused
	}

	return Unit{
		Position:      position,
		SpawnPosition: position,
		Kind:          kind,
		MaxHealth:     3,
		Health:        3,
		animation:     runnerAnimation,
		elapsed:       phase,
		moveSpeed:     48,
	}
}

func NewWall(position geom.Point) Unit {
	return Unit{
		Position:      position,
		SpawnPosition: position,
		Kind:          KindWall,
		MaxHealth:     5,
		Health:        5,
		animation:     Animation{FrameCount: 1, FrameDuration: 1},
	}
}

func NewBarricade(position geom.Point) Unit {
	return Unit{
		Position:      position,
		SpawnPosition: position,
		Kind:          KindBarricade,
		MaxHealth:     4,
		Health:        4,
		animation:     Animation{FrameCount: 1, FrameDuration: 1},
	}
}

func (u *Unit) Tick(gameTick int64, delta float64, speedMultiplier func(geom.Point) float64) {
	if !u.Alive() {
		u.travel = travelState{}
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
	u.sleepTime = u.advance(gameTick, delta, speedMultiplier)
	u.travel.remaining = u.sleepTime
}

func (u Unit) Frame() int {
	return u.animation.frameAt(u.elapsed)
}

func (u Unit) Name() string {
	switch u.Kind {
	case KindRunner:
		return "Runner"
	case KindRunnerFocused:
		return "Runner Focused"
	case KindWall:
		return "Wall"
	case KindBarricade:
		return "Barricade"
	default:
		return string(u.Kind)
	}
}

func (u Unit) TilePosition(tileSize float64) (int, int) {
	if tileSize <= 0 {
		return 0, 0
	}

	return int(math.Floor(u.Position.X / tileSize)), int(math.Floor(u.Position.Y / tileSize))
}

func (u *Unit) SetPath(path []geom.Point) {
	if !u.IsMobile() {
		u.path = u.path[:0]
		return
	}

	u.path = append(u.path[:0], path...)
	u.sleepTime = 0
}

func (u Unit) IsMobile() bool {
	return u.Alive() && u.moveSpeed > 0
}

func (u Unit) BlocksMovement() bool {
	if !u.Alive() {
		return false
	}

	switch u.Kind {
	case KindWall, KindBarricade:
		return true
	default:
		return false
	}
}

func (u Unit) CanShoot() bool {
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

func (u Unit) Alive() bool {
	return u.Health > 0
}

func (u Unit) HealthRatio() float64 {
	if u.MaxHealth <= 0 {
		return 0
	}

	return geom.ClampFloat(float64(u.Health)/float64(u.MaxHealth), 0, 1)
}

func (u *Unit) ApplyDamage(amount int) bool {
	if amount <= 0 || !u.Alive() {
		return false
	}

	u.Health -= amount
	return u.Health <= 0
}

func (u *Unit) Respawn() {
	u.Position = u.SpawnPosition
	u.Health = u.MaxHealth
	u.path = u.path[:0]
	u.sleepTime = 0
	u.travel = travelState{}
}

func (u Unit) HasPath() bool {
	return len(u.path) > 0
}

func (u Unit) IsMoving() bool {
	return len(u.path) > 0 || (u.travel.active && u.travel.remaining > 0)
}

func (u Unit) PathLen() int {
	return len(u.path)
}

func (u Unit) SleepTime() int {
	return u.sleepTime
}

func (u Unit) LastUpdateTick() int64 {
	return u.lastUpdateTick
}

func (u Unit) Destination() (geom.Point, bool) {
	if len(u.path) == 0 {
		if u.travel.active {
			return u.travel.to, true
		}
		return geom.Point{}, false
	}

	return u.path[len(u.path)-1], true
}

func (u Unit) RenderPosition() geom.Point {
	if !u.travel.active || u.travel.duration <= 0 {
		return u.Position
	}

	progress := 1 - float64(u.travel.remaining)/float64(u.travel.duration)
	progress = geom.ClampFloat(progress, 0, 1)
	return geom.Point{
		X: u.travel.from.X + (u.travel.to.X-u.travel.from.X)*progress,
		Y: u.travel.from.Y + (u.travel.to.Y-u.travel.from.Y)*progress,
	}
}

func (u *Unit) Wake() {
	u.sleepTime = 0
}

func (u *Unit) advance(_ int64, delta float64, speedMultiplier func(geom.Point) float64) int {
	if delta <= 0 || len(u.path) == 0 || u.moveSpeed <= 0 {
		u.travel = travelState{}
		return 0
	}

	for len(u.path) > 0 {
		target := u.path[0]
		dx := target.X - u.Position.X
		dy := target.Y - u.Position.Y
		distance := math.Hypot(dx, dy)
		if distance <= 1e-6 {
			u.Position = target
			u.path = u.path[1:]
			continue
		}

		currentSpeed := u.moveSpeed
		if speedMultiplier != nil {
			multiplier := speedMultiplier(u.Position)
			if multiplier <= 0 {
				u.travel = travelState{}
				return 0
			}
			currentSpeed *= multiplier
		}
		if currentSpeed <= 0 {
			u.travel = travelState{}
			return 0
		}

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

	u.travel = travelState{}
	return 0
}

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
