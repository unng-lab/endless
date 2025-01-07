package board

import (
	"fmt"
	"sort"
	"sync"
)

// Interval представляет собой интервал [Start, End)
type Interval struct {
	Start, End int64
	Cost       float64
	m          sync.Mutex
}

var pool = sync.Pool{
	New: func() interface{} {
		return make([]Interval, 0, 8)
	},
}

// IntervalTree для хранения интервалов
type IntervalTree struct {
	m    sync.RWMutex
	data []Interval
}

// Add добавляет новый интервал в дерево
func (tree *IntervalTree) Add(interval Interval) (*Interval, error) {
	tree.m.Lock()
	defer tree.m.Unlock()
	if len(tree.data) == 0 {
		tree.data = pool.Get().([]Interval)
	}
	// Проверяем, что интервал уникален и не пересекается с существующими
	index := sort.Search(len(tree.data), func(i int) bool {
		return tree.data[i].End > interval.Start
	})

	if index < len(tree.data) && tree.data[index].Start < interval.End {
		return nil, fmt.Errorf("пересечение с интервалом [%d, %d)", tree.data[index].Start, tree.data[index].End)
	}

	// Вставляем интервал в отсортированный список
	tree.data = append(tree.data[:index], append([]Interval{interval}, tree.data[index:]...)...)
	return nil, nil
}

// Find ищет интервал, который пересекается с точкой
func (tree *IntervalTree) Find(point int64) *Interval {
	tree.m.RLock()
	defer tree.m.RUnlock()
	if len(tree.data) == 0 {
		return nil
	}
	index := sort.Search(len(tree.data), func(i int) bool {
		return tree.data[i].End > point
	})
	if index < len(tree.data) && tree.data[index].Start <= point {
		return &tree.data[index]
	}
	return nil
}

// Remove удаляет интервал из дерева, если он существует
func (tree *IntervalTree) Remove(interval Interval) error {
	tree.m.Lock()
	defer tree.m.Unlock()
	// Находим индекс удаляемого интервала
	index := sort.Search(len(tree.data), func(i int) bool {
		return tree.data[i].Start >= interval.Start
	})

	// Проверяем существование и точное совпадение
	if index < len(tree.data) && tree.data[index] == interval {
		// Удаляем интервал
		tree.data = append(tree.data[:index], tree.data[index+1:]...)
		if len(tree.data) == 0 {
			pool.Put(tree.data)
			tree.data = nil
		}
		return nil
	}

	return fmt.Errorf("интервал [%d, %d) не найден", interval.Start, interval.End)
}
