package unit

import (
	"errors"
)

type Task interface {
	Next() (int, error)
	GetName() string
	GetDescription() string
	// Update вызывается во время нахождения на экране пользователя, то есть может вызваться в любой момент
	Update(*Unit) error
	// Other common methods
}

var (
	taskBuffer = 8
)

type TaskList struct {
	Tasks []Task
}

func NewTaskList() TaskList {
	return TaskList{
		Tasks: make([]Task, 0, 8),
	}
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
			tl.Tasks = tl.Tasks[:0]
		}
		//if len(tl.Tasks) > 1 {
		//	tl.Tasks = tl.Tasks[1:]
		//	tl.Run()
		//}
	}
	return sleepTicks
}

func (tl *TaskList) Finish() {
	if cap(tl.Tasks) < taskBuffer*4 {
		tl.Tasks = tl.Tasks[:0]
		return
	}
	tl.Tasks = make([]Task, 0, taskBuffer)
}

func (tl *TaskList) Current() Task {
	if len(tl.Tasks) == 0 {
		return nil
	}
	return tl.Tasks[0]
}

var ErrTaskFinished = errors.New("Task finished")
