package dstar

import (
	"reflect"
	"testing"
)

// Тестирование метода Push и Pop
func TestPushPop(t *testing.T) {
	ds := NewDStar(nil)

	nodes := []*Node{
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{5, 2}},
		{Key: [2]float64{5, 1}},
		{Key: [2]float64{20, 3}},
		{Key: [2]float64{15, 0}},
	}

	// Ожидаемый порядок при извлечении: наименьший Key первым
	expectedOrder := []*Node{
		{Key: [2]float64{5, 1}},
		{Key: [2]float64{5, 2}},
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{15, 0}},
		{Key: [2]float64{20, 3}},
	}

	for _, node := range nodes {
		ds.Push(node)
	}

	for i, expected := range expectedOrder {
		popped := ds.Pop()
		if !reflect.DeepEqual(popped.Key, expected.Key) {
			t.Errorf("Pop[%d] = %v; want %v", i, popped.Key, expected.Key)
		}
	}

	if ds.Len() != 0 {
		t.Errorf("After popping all elements, Len() = %d; want 0", ds.Len())
	}
}

// Тестирование метода Remove
func TestRemove(t *testing.T) {
	ds := NewDStar(nil)

	nodes := []*Node{
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{5, 2}},
		{Key: [2]float64{5, 1}},
		{Key: [2]float64{20, 3}},
		{Key: [2]float64{15, 0}},
	}

	for _, node := range nodes {
		ds.Push(node)
	}

	// Предположим, мы хотим удалить узел с Key {5,2}, который должен быть на втором месте
	var removeIndex int = -1
	for i, node := range ds.nodes {
		if reflect.DeepEqual(node.Key, [2]float64{5, 2}) {
			removeIndex = i
			break
		}
	}

	if removeIndex == -1 {
		t.Fatal("Node with Key {5,2} not found")
	}

	removedNode := ds.Remove(removeIndex)
	if !reflect.DeepEqual(removedNode.Key, [2]float64{5, 2}) {
		t.Errorf("Removed node Key = %v; want {5,2}", removedNode.Key)
	}

	// Ожидаемый порядок после удаления
	expectedOrder := []*Node{
		{Key: [2]float64{5, 1}},
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{15, 0}},
		{Key: [2]float64{20, 3}},
	}

	// Проверяем оставшиеся элементы
	for i, expected := range expectedOrder {
		popped := ds.Pop()
		if !reflect.DeepEqual(popped.Key, expected.Key) {
			t.Errorf("After Remove, Pop[%d] = %v; want %v", i, popped.Key, expected.Key)
		}
	}

	if ds.Len() != 0 {
		t.Errorf("After popping all elements, Len() = %d; want 0", ds.Len())
	}
}

// Тестирование метода Fix
func TestFix(t *testing.T) {
	ds := NewDStar(nil)

	nodes := []*Node{
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{20, 2}},
		{Key: [2]float64{30, 3}},
	}

	for _, node := range nodes {
		ds.Push(node)
	}

	// Изменим ключ узла с Index 1 с {20,2} на {5,0}, что должно переместить его на вершину
	ds.nodes[1].Key = [2]float64{5, 0}
	ds.Fix(1)

	// Ожидаемый порядок после изменения
	expectedOrder := []*Node{
		{Key: [2]float64{5, 0}},
		{Key: [2]float64{10, 1}},
		{Key: [2]float64{30, 3}},
	}

	for i, expected := range expectedOrder {
		popped := ds.Pop()
		if !reflect.DeepEqual(popped.Key, expected.Key) {
			t.Errorf("After Fix, Pop[%d] = %v; want %v", i, popped.Key, expected.Key)
		}
	}

	if ds.Len() != 0 {
		t.Errorf("After popping all elements, Len() = %d; want 0", ds.Len())
	}
}

// Тестирование работы с пустой кучей
func TestEmptyHeap(t *testing.T) {
	ds := NewDStar(nil)

	if ds.Len() != 0 {
		t.Errorf("Initial Len() = %d; want 0", ds.Len())
	}

	node := ds.Pop()
	if node != nil {
		t.Errorf("Pop() on empty heap = %v; want nil", node)
	}

	removed := ds.Remove(0)
	if removed != nil {
		t.Errorf("Remove() on empty heap = %v; want nil", removed)
	}
}

// Тестирование кучи с одним элементом
func TestSingleElement(t *testing.T) {
	ds := NewDStar(nil)

	node := &Node{Key: [2]float64{10, 1}}
	ds.Push(node)

	if ds.Len() != 1 {
		t.Errorf("After Push, Len() = %d; want 1", ds.Len())
	}

	popped := ds.Pop()
	if !reflect.DeepEqual(popped.Key, node.Key) {
		t.Errorf("Pop() = %v; want %v", popped.Key, node.Key)
	}

	if ds.Len() != 0 {
		t.Errorf("After Pop, Len() = %d; want 0", ds.Len())
	}
}

// Тестирование последовательного добавления и удаления элементов
func TestSequentialPushPop(t *testing.T) {
	ds := NewDStar(nil)

	for i := 10; i >= 1; i-- {
		node := &Node{Key: [2]float64{float64(i), float64(i * 10)}}
		ds.Push(node)
	}

	expectedOrder := []*Node{
		{Key: [2]float64{1, 10}},
		{Key: [2]float64{2, 20}},
		{Key: [2]float64{3, 30}},
		{Key: [2]float64{4, 40}},
		{Key: [2]float64{5, 50}},
		{Key: [2]float64{6, 60}},
		{Key: [2]float64{7, 70}},
		{Key: [2]float64{8, 80}},
		{Key: [2]float64{9, 90}},
		{Key: [2]float64{10, 100}},
	}

	for i, expected := range expectedOrder {
		popped := ds.Pop()
		if !reflect.DeepEqual(popped.Key, expected.Key) {
			t.Errorf("Sequential Pop[%d] = %v; want %v", i, popped.Key, expected.Key)
		}
	}

	if ds.Len() != 0 {
		t.Errorf("After popping all elements, Len() = %d; want 0", ds.Len())
	}
}
