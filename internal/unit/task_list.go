package unit

import (
	"errors"
)

type Task interface {
	Next() (int, error)
	GetName() string
	GetDescription() string
	// Other common methods
}

type TaskList struct {
	Tasks []Task
}

func NewTaskList() TaskList {
	return TaskList{}
}

func (tl *TaskList) Add(task Task) {
	tl.Tasks = append(tl.Tasks, task)
}

func (tl *TaskList) Run() int {
	if len(tl.Tasks) == 0 {
		return 0
	}
	sleepTicks, err := tl.Tasks[0].Next()
	if err != nil {
		if errors.Is(err, ErrTaskFinished) {
			tl.Tasks = nil
		}
		//if len(tl.Tasks) > 1 {
		//	tl.Tasks = tl.Tasks[1:]
		//	tl.Run()
		//}
	}
	return sleepTicks
}

var ErrTaskFinished = errors.New("Task finished")
