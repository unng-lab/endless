package board

import "sync"

var pool = sync.Pool{
	New: func() interface{} {
		return new(Interval)
	},
}

// Interval представляет узел AVL-дерева
type Interval struct {
	Left, Right *Interval
	Height      int64
	Start, End  int64
	rwLock      sync.RWMutex
}

// IntervalTree представляет AVL-дерево интервалов
type IntervalTree struct {
	root   *Interval
	rwLock sync.RWMutex
}

func (tree *IntervalTree) Add(start, end int64) (*Interval, error) {
	tree.rwLock.Lock()
	defer tree.rwLock.Unlock()
	return &Interval{
		Start: start,
		End:   end,
	}, nil
}

func (tree *IntervalTree) Remove(start, end int64) (*Interval, error) {
	tree.rwLock.Lock()
	defer tree.rwLock.Unlock()
	return &Interval{
		Start: start,
		End:   end,
	}, nil
}

func (i *Interval) Cost() float64 {
	return 1.0
}
