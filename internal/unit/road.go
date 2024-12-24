package unit

import (
	"log/slog"

	"github/unng-lab/madfarmer/internal/astar"
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

var _ Task = new(Road)

type Road struct {
	astar.Astar

	unit               *Unit
	nextMove           func() error
	position           geom.Point
	timeToWalkOnePoint int
	nextPoint          geom.Point
}

func (r *Road) Update(unit *Unit) error {
	part := float64(r.timeToWalkOnePoint-unit.SleepTicks) / float64(r.timeToWalkOnePoint)
	unit.PositionShiftModX = (r.nextPoint.X - r.position.X) * part
	unit.PositionShiftModY = (r.nextPoint.Y - r.position.Y) * part
	//slog.Info("shift mod", "x", unit.PositionShiftModX, "y", unit.PositionShiftModY)
	return nil
}

func NewRoad(b *board.Board, unit *Unit) Road {
	return Road{
		unit:  unit,
		Astar: astar.NewAstar(b),
	}
}

func (r *Road) Next() (int, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("error recover", r)
		}
	}()

	if r.nextMove != nil {
		if r.position == r.unit.Position {
			if err := r.nextMove(); err == nil {
				if len(r.Astar.Path) == 0 {
					slog.Info("task finished")
					return 0, ErrTaskFinished
				}
			} else {
				slog.Error("nextMove", err)
			}
		} else {
			// если передвинулся то надо логику обработать
			slog.Error("юнит переместился в другую точку", r.unit)
		}

	}

	dir := r.unit.Position.To(r.Astar.Path[len(r.Astar.Path)-1])

	nextPoint, err := r.unit.Position.GetNeighbor(dir)

	if err != nil {
		slog.Error("GetNeighbor", "error", err, "unitType", r.unit.Type, "dir", dir)
		return 0, err
	}

	if nextPoint == r.Astar.Path[len(r.Astar.Path)-1] {
		r.Astar.Path = r.Astar.Path[:len(r.Astar.Path)-1]
	}

	walkOnePoint := timeToWalkOnePoint(r.unit, r.B, nextPoint)
	r.nextMove = func() error {
		r.unit.Relocate(r.unit.Position, nextPoint)
		return nil
	}
	r.position = r.unit.Position
	r.timeToWalkOnePoint = walkOnePoint
	r.nextPoint = nextPoint

	return walkOnePoint, nil
}

func timeToWalkOnePoint(unit *Unit, b *board.Board, nextPoint geom.Point) int {
	firstCellMoveCost := b.Cells[int(unit.Position.X)][int(unit.Position.Y)].MoveCost()
	secondCellMoveCost := b.Cells[int(nextPoint.X)][int(nextPoint.Y)].MoveCost()
	averageMoveCost := (firstCellMoveCost + secondCellMoveCost) / 2
	return int(1 / (unit.Speed * averageMoveCost))
}

func (r *Road) GetName() string {
	return "Walk Task"
}

func (r *Road) GetDescription() string {
	return "Task to walk"
}

func (r *Road) Path(to geom.Point) error {
	err := r.BuildPath(r.unit.Position.X, r.unit.Position.Y, to.X, to.Y)
	if err != nil {
		return err
	}
	return nil
}
