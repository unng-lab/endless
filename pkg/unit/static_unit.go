package unit

import (
	"github.com/unng-lab/endless/pkg/geom"
)

// StaticUnit represents an immobile world object such as a wall, barricade, tree or bush.
// It still embeds BaseUnit so selection, health bars and visibility stay aligned with the
// rest of the gameplay entities.
type StaticUnit struct {
	BaseUnit
	ID            int64
	SpawnPosition geom.Point
	Kind          Kind
	MaxHealth     int
	Health        int

	blocksMovement bool
}

func NewWall(position geom.Point) *StaticUnit {
	return newStaticUnit(KindWall, position, 5, true)
}

func NewBarricade(position geom.Point) *StaticUnit {
	return newStaticUnit(KindBarricade, position, 4, true)
}

func newStaticUnit(kind Kind, position geom.Point, maxHealth int, blocksMovement bool) *StaticUnit {
	return &StaticUnit{
		BaseUnit: BaseUnit{
			Position: position,
		},
		SpawnPosition:  position,
		Kind:           kind,
		MaxHealth:      maxHealth,
		Health:         maxHealth,
		blocksMovement: blocksMovement,
	}
}

func (s *StaticUnit) Base() *BaseUnit {
	return &s.BaseUnit
}

func (s *StaticUnit) UnitID() int64 {
	return s.ID
}

func (s *StaticUnit) SetUnitID(id int64) {
	s.ID = id
}

func (s *StaticUnit) UnitKind() Kind {
	return s.Kind
}

func (s *StaticUnit) Name() string {
	switch s.Kind {
	case KindWall:
		return "Wall"
	case KindBarricade:
		return "Barricade"
	default:
		return string(s.Kind)
	}
}

func (s *StaticUnit) Alive() bool {
	return s.Health > 0
}

func (s *StaticUnit) IsMobile() bool {
	return false
}

func (s *StaticUnit) BlocksMovement() bool {
	return s.Alive() && s.blocksMovement
}

func (s *StaticUnit) CanShoot() bool {
	return false
}

func (s *StaticUnit) Frame() int {
	return 0
}

func (s *StaticUnit) CurrentHealth() int {
	return s.Health
}

func (s *StaticUnit) MaxHealthValue() int {
	return s.MaxHealth
}

func (s *StaticUnit) HealthRatio() float64 {
	if s.MaxHealth <= 0 {
		return 0
	}

	return geom.ClampFloat(float64(s.Health)/float64(s.MaxHealth), 0, 1)
}

func (s *StaticUnit) ApplyDamage(amount int) bool {
	if amount <= 0 || !s.Alive() {
		return false
	}

	s.Health -= amount
	return s.Health <= 0
}

func (s *StaticUnit) Respawn() {
	s.Position = s.SpawnPosition
	s.Health = s.MaxHealth
	s.clearTravel()
}

func (s *StaticUnit) Selectable() bool {
	return s.Alive()
}
