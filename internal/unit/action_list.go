package unit

import (
	"fmt"
)

type Action interface {
	Execute() error
}

// Узел двусвязного списка
type node struct {
	prev       *node
	next       *node
	sleepTicks int
	action     Action
}

// Двусвязный список
type ActionList struct {
	head *node
	tail *node
}

// Создание нового списка
func NewActionList() ActionList {
	return ActionList{}
}

// Вставка элемента в список в отсортированном порядке по sleepTicks (убывание)
func (l *ActionList) InsertSorted(sleepTicks int, action Action) {
	newNode := &node{
		sleepTicks: sleepTicks,
		action:     action,
	}

	// Если список пуст, новый узел становится головой и хвостом
	if l.head == nil {
		l.head = newNode
		l.tail = newNode
		return
	}

	current := l.head
	// Ищем позицию для вставки (сортировка по sleepTicks в порядке убывания)
	for current != nil && current.sleepTicks >= newNode.sleepTicks {
		// Если нашли узел с таким же sleepTicks, пропускаем его
		// чтобы новый узел вставить после узлов с sleepTicks на единицу меньше
		if current.sleepTicks == newNode.sleepTicks-1 {
			break
		}
		current = current.next
	}

	if current == nil {
		// Вставка в конец списка
		newNode.prev = l.tail
		l.tail.next = newNode
		l.tail = newNode
	} else if current.prev == nil {
		// Вставка в начало списка
		newNode.next = l.head
		l.head.prev = newNode
		l.head = newNode
	} else {
		// Вставка между узлами
		prevNode := current.prev
		prevNode.next = newNode
		newNode.prev = prevNode
		newNode.next = current
		current.prev = newNode
	}
}

// Обход списка и вывод значений sleepTicks
func (l *ActionList) Traverse() {
	current := l.head
	fmt.Println("Список в порядке убывания sleepTicks:")
	for current != nil {
		fmt.Printf("sleepTicks: %d, action: %v\n", current.sleepTicks, current.action)
		current = current.next
	}
}
