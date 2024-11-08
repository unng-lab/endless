package unit

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
	sleepTicks, err := tl.Tasks[0].Next()
	if err != nil {
		tl.Tasks = tl.Tasks[1:]
	}
	return sleepTicks
}
