package board

import (
	"testing"
)

func TestIntervalTree_Add(t *testing.T) {
	tree := &IntervalTree{}
	if _, err := tree.Add(Interval{Start: 1, End: 5}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := tree.Add(Interval{Start: 10, End: 15}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := tree.Add(Interval{Start: 3, End: 7}); err == nil {
		t.Errorf("expected error due to overlap, got none")
	}
}

func TestIntervalTree_Find(t *testing.T) {
	tree := &IntervalTree{}
	tree.Add(Interval{Start: 1, End: 5})
	tree.Add(Interval{Start: 10, End: 15})

	tests := []struct {
		point    int64
		expected *Interval
	}{
		{point: 3, expected: &Interval{Start: 1, End: 5}},
		{point: 12, expected: &Interval{Start: 10, End: 15}},
		{point: 6, expected: nil},
	}

	for _, test := range tests {
		result := tree.Find(test.point)
		if result == nil && test.expected != nil {
			t.Errorf("point %d: expected interval %v, got nil", test.point, test.expected)
		}
		if result != nil && (result.Start != test.expected.Start || result.End != test.expected.End) {
			t.Errorf("point %d: expected interval %v, got %v", test.point, test.expected, result)
		}
	}
}

func TestIntervalTree_Remove(t *testing.T) {
	tree := &IntervalTree{}
	tree.Add(Interval{Start: 1, End: 5})
	tree.Add(Interval{Start: 10, End: 15})

	// Удаляем существующий интервал
	err := tree.Remove(Interval{Start: 1, End: 5})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Проверяем, что интервал был удален
	interval := tree.Find(3)
	if interval != nil {
		t.Errorf("expected no interval for point 3, got %v", interval)
	}

	// Пытаемся удалить несуществующий интервал
	err = tree.Remove(Interval{Start: 1, End: 5})
	if err == nil {
		t.Errorf("expected error for deleting non-existing interval, got none")
	}
}

func TestIntervalTree_RemoveOne(t *testing.T) {
	tree := &IntervalTree{}
	tree.Add(Interval{Start: 1, End: 5})

	// Удаляем существующий интервал
	err := tree.Remove(Interval{Start: 1, End: 5})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Проверяем, что интервал был удален
	interval := tree.Find(3)
	if interval != nil {
		t.Errorf("expected no interval for point 3, got %v", interval)
	}

	// Пытаемся удалить несуществующий интервал
	err = tree.Remove(Interval{Start: 1, End: 5})
	if err == nil {
		t.Errorf("expected error for deleting non-existing interval, got none")
	}
}

func TestIntervalTree_Remove_NonIntersecting(t *testing.T) {
	tree := &IntervalTree{}
	tree.Add(Interval{Start: 1, End: 5})
	tree.Add(Interval{Start: 10, End: 15})

	// Пытаемся удалить интервал, которого нет
	err := tree.Remove(Interval{Start: 2, End: 6})
	if err == nil {
		t.Errorf("expected error for deleting non-existing interval, got none")
	}
}
