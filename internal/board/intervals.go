package board

import (
	"fmt"
	"sort"
)

// Interval представляет собой интервал [Start, End)
type Interval struct {
	Start, End int64
	Cost       float64
}

// IntervalTree для хранения интервалов
type IntervalTree struct {
	data []Interval
}

// Add добавляет новый интервал в дерево
func (tree *IntervalTree) Add(interval Interval) error {
	// Проверяем, что интервал уникален и не пересекается с существующими
	index := sort.Search(len(tree.data), func(i int) bool {
		return tree.data[i].End > interval.Start
	})

	if index < len(tree.data) && tree.data[index].Start < interval.End {
		return fmt.Errorf("пересечение с интервалом [%d, %d)", tree.data[index].Start, tree.data[index].End)
	}

	// Вставляем интервал в отсортированный список
	tree.data = append(tree.data[:index], append([]Interval{interval}, tree.data[index:]...)...)
	return nil
}

// Find ищет интервал, который пересекается с точкой
func (tree *IntervalTree) Find(point int64) *Interval {
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
	// Находим индекс удаляемого интервала
	index := sort.Search(len(tree.data), func(i int) bool {
		return tree.data[i].Start >= interval.Start
	})

	// Проверяем существование и точное совпадение
	if index < len(tree.data) && tree.data[index] == interval {
		// Удаляем интервал
		tree.data = append(tree.data[:index], tree.data[index+1:]...)
		return nil
	}

	return fmt.Errorf("интервал [%d, %d) не найден", interval.Start, interval.End)
}
