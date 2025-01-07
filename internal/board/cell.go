package board

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Cell struct {
	TileImage      *ebiten.Image
	TileImageSmall *ebiten.Image
	Type           int
	// Стоимость перемещения в зависимости от типа клетки
	Cost float64
	// Храним стоимость перемещения в зависимости от временного интервала
	Costs IntervalTree
}

// MoveCost Функция расчета стоимости перемещения в зависимости от временного интервала
func (c *Cell) MoveCost(start, end int64) float64 {
	// Если клетка является препятствием (гора, яма и т.д.), то возвращаем максимальную стоимость
	if c.Cost == math.Inf(1) {
		return c.Cost
	}
	// Если клетка не является препятствием, то смотрим есть ли какой-либо объект на клетке в заданном временном интервале
	interval, err := c.Costs.Add(start, end)
	tCost := 0.0
	if err != nil {
		if interval != nil {
			tCost = interval.Cost()
		} else {
			//TODO Проверить
			panic("Ошибка при добавлении интервала в дерево стоимости перемещения")
		}

	}

	// Если объект на клетке в заданном временном интервале является препятствием, то возвращаем максимальную стоимость
	if tCost != math.Inf(1) {
		return tCost
	}

	// Итоговая стоимость перемещения равна сумме стоимости перемещения в зависимости от временного интервала и типа клетки
	return tCost + c.Cost
}
