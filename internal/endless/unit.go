package endless

type Unit[T any] struct {
	Body *T
	Name string
	Type string
}
