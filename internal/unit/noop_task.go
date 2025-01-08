package unit

import "github.com/unng-lab/madfarmer/internal/board"

var _ Task = new(NoopTask)

func NewNoopTask(b *board.Board, unit *Unit) NoopTask {
	return NoopTask{}
}

type NoopTask struct {
}

func (n NoopTask) Next() (int, error) {
	return 1000, nil
}

func (n NoopTask) GetName() string {
	return "noop"
}

func (n NoopTask) GetDescription() string {
	return "task that does nothing"
}

func (n NoopTask) Update(unit *Unit) error {
	return nil
}
