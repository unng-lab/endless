package unit

import (
	"testing"
)

// Тестовый Action для использования в тестах
type TestAction struct {
	name string
}

func (a *TestAction) Execute() error {
	// Для тестов реализация может быть пустой
	return nil
}

// Функция для сбора значений sleepTicks из списка в слайс
func getSleepTicksList(l *ActionList) []int {
	var ticks []int
	current := l.head
	for current != nil {
		ticks = append(ticks, current.sleepTicks)
		current = current.next
	}
	return ticks
}

// Тестирование вставки в пустой список
func TestInsertIntoEmptyList(t *testing.T) {
	list := NewActionList()
	list.InsertSorted(5, &TestAction{name: "Action 5"})

	if list.head == nil || list.tail == nil {
		t.Fatalf("Head or tail is nil after first insertion")
	}

	if list.head != list.tail {
		t.Fatalf("Head and tail should be equal when only one element is in the list")
	}

	if list.head.sleepTicks != 5 {
		t.Errorf("Expected sleepTicks 5, got %d", list.head.sleepTicks)
	}
}

// Тестирование вставки элементов и корректности порядка
func TestInsertAndOrder(t *testing.T) {
	list := NewActionList()

	testData := []struct {
		sleepTicks int
		name       string
	}{
		{5, "Action 5"},
		{2, "Action 2"},
		{8, "Action 8"},
		{1, "Action 1"},
		{4, "Action 4"},
		{0, "Action 0"},
		{10, "Action 10"},
		{9, "Action 9"},
		{10, "Action 10-2"},
	}

	for _, data := range testData {
		list.InsertSorted(data.sleepTicks, &TestAction{name: data.name})
	}

	expectedOrder := []int{10, 10, 9, 8, 5, 4, 2, 1, 0}
	actualOrder := getSleepTicksList(list)

	if len(actualOrder) != len(expectedOrder) {
		t.Fatalf("Expected list length %d, got %d", len(expectedOrder), len(actualOrder))
	}

	for i, expectedTick := range expectedOrder {
		if actualOrder[i] != expectedTick {
			t.Errorf("At index %d, expected sleepTicks %d, got %d", i, expectedTick, actualOrder[i])
		}
	}
}

// Тестирование вставки элементов с одинаковыми sleepTicks
func TestInsertDuplicateSleepTicks(t *testing.T) {
	list := NewActionList()

	list.InsertSorted(5, &TestAction{name: "Action 5-1"})
	list.InsertSorted(5, &TestAction{name: "Action 5-2"})
	list.InsertSorted(5, &TestAction{name: "Action 5-3"})

	expectedOrder := []int{5, 5, 5}
	actualOrder := getSleepTicksList(list)

	if len(actualOrder) != len(expectedOrder) {
		t.Fatalf("Expected list length %d, got %d", len(expectedOrder), len(actualOrder))
	}

	for i, expectedTick := range expectedOrder {
		if actualOrder[i] != expectedTick {
			t.Errorf("At index %d, expected sleepTicks %d, got %d", i, expectedTick, actualOrder[i])
		}
	}
}

// Тестирование вставки элемента с sleepTicks, для которого нет sleepTicks-1 в списке
func TestInsertWithoutPreviousSleepTicks(t *testing.T) {
	list := NewActionList()

	list.InsertSorted(3, &TestAction{name: "Action 3"})
	list.InsertSorted(7, &TestAction{name: "Action 7"}) // Нет элементов с sleepTicks 6

	expectedOrder := []int{7, 3}
	actualOrder := getSleepTicksList(list)

	if len(actualOrder) != len(expectedOrder) {
		t.Fatalf("Expected list length %d, got %d", len(expectedOrder), len(actualOrder))
	}

	for i, expectedTick := range expectedOrder {
		if actualOrder[i] != expectedTick {

			t.Errorf("At index %d, expected sleepTicks %d, got %d", i, expectedTick, actualOrder[i])
		}
	}
}

// Тестирование граничных значений sleepTicks
func TestBoundarySleepTicks(t *testing.T) {
	list := NewActionList()

	list.InsertSorted(0, &TestAction{name: "Action 0"})
	list.InsertSorted(100, &TestAction{name: "Action 100"})

	//expectedOrder := []int{100, 0}
	actualOrder := getSleepTicksList(list)

	if len(actualOrder) != 2 {
		t.Fatalf("Expected list length 2, got %d", len(actualOrder))
	}

	if actualOrder[0] != 100 || actualOrder[1] != 0 {
		t.Errorf("Expected sleepTicks order [100, 0], got %v", actualOrder)
	}
}
