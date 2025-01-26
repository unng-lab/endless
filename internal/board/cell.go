package board

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/assets/img"
)

type CellType byte

const (
	CellTypeUndefined CellType = iota
	CellTypeGround
	CellTypeRoad
	CellTypeWood
	CellTypeSwamp
	CellTypeWater
)

const (
	RoadCost   = 100
	GroundCost = 300
	WoodCost   = 1000
	SwampCost  = 10000
)

type Cell struct {
	TileImage      *ebiten.Image
	TileImageSmall *ebiten.Image
	Type           CellType
	// Стоимость перемещения в зависимости от типа клетки
	Cost float64
	// Храним стоимость перемещения в зависимости от временного интервала
	Costs IntervalTree
}

// MoveCost Функция расчета стоимости перемещения в зависимости от временного интервала
func (c *Cell) MoveCost(start, end int64) float64 {
	// Если клетка является препятствием (гора, яма и т.д.), то возвращаем максимальную стоимость
	if math.IsInf(c.Cost, 1) {
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
	if tCost == math.Inf(1) {
		return tCost
	}

	// Итоговая стоимость перемещения равна сумме стоимости перемещения в зависимости от временного интервала и типа клетки
	return tCost + c.Cost
}

func NewCell(cellType CellType) Cell {
	var cell Cell
	cell.Type = cellType
	switch cellType {
	case CellTypeGround:
		cell.Cost = GroundCost
		cell.TileImage, cell.TileImageSmall = getRandomTileImageFromQuadrant(1)
	case CellTypeRoad:
		cell.Cost = RoadCost
		cell.TileImage, cell.TileImageSmall = getRandomTileImageFromQuadrant(3)
	case CellTypeWood:
		cell.Cost = WoodCost
		cell.TileImage, cell.TileImageSmall = getRandomTileImageFromQuadrant(4)
	case CellTypeSwamp:
		cell.Cost = SwampCost
		cell.TileImage, cell.TileImageSmall = getRandomTileImageFromQuadrant(2)
	case CellTypeWater:
		cell.Cost = math.Inf(1)
		cell.TileImage, cell.TileImageSmall = getWaterImg()
	case CellTypeUndefined:
		panic("Неизвестный тип клетки")
	default:
		panic("Неизвестный тип клетки")
	}

	return cell
}

func getRandomTileImageFromQuadrant(n int) (*ebiten.Image, *ebiten.Image) {
	if n <= 0 || n > 4 {
		panic("Неверный размер квадрата")
	}
	i, err := getRandomElementFromQuadrant(n, TileSize)
	if err != nil {
		panic(err)
	}
	return Tiles[i].Normal, Tiles[i].Small
}

func getWaterImg() (*ebiten.Image, *ebiten.Image) {
	spriteRocks, err := img.Img("boat.png", 16, 16)
	if err != nil {
		panic(err)
	}
	return spriteRocks, spriteRocks
}

// getQuadrant определяет, в какой четверти находится мелкий квадрат по его индексу.
// Параметры:
// - index: индекс мелкого квадрата в одномерном массиве (начинается с 0).
// - N: размер большой стороны квадрата (например, для 16x16 передайте 16).
// Возвращает номер четверти (1, 2, 3, 4) и ошибку, если ввод некорректен.
func getQuadrant(index, N int) (int, error) {
	if N <= 0 {
		return 0, fmt.Errorf("размер квадрата должен быть положительным, получено N=%d", N)
	}
	if index < 0 || index >= N*N {
		return 0, fmt.Errorf("индекс %d выходит за пределы массива размером %dx%d", index, N, N)
	}

	// Преобразование индекса в координаты (строка, столбец)
	row := index / N // Номер строки (от 0 до N-1)
	col := index % N // Номер столбца (от 0 до N-1)

	// Определение середины
	midRow := N / 2
	midCol := N / 2

	// Определение четверти
	if row < midRow {
		if col < midCol {
			return 2, nil // Верхняя левая четверть
		}
		return 1, nil // Верхняя правая четверть
	}
	if col < midCol {
		return 3, nil // Нижняя левая четверть
	}
	return 4, nil // Нижняя правая четверть
}

// getRandomElementFromQuadrant возвращает случайный индекс из заданной четверти.
// Параметры:
// - quadrant: номер четверти (1, 2, 3, 4).
// - N: размер стороны большого квадрата (например, для 16x16 передайте 16).
// Возвращает случайный индекс и ошибку, если ввод некорректен.
func getRandomElementFromQuadrant(quadrant, N int) (int, error) {
	if N <= 0 {
		return 0, fmt.Errorf("размер квадрата должен быть положительным, получено N=%d", N)
	}
	if quadrant < 1 || quadrant > 4 {
		return 0, fmt.Errorf("неверный номер четверти: %d. Должно быть от 1 до 4", quadrant)
	}

	// Определение середины
	midRow := N / 2
	midCol := N / 2

	var rowRangeStart, rowRangeEnd, colRangeStart, colRangeEnd int

	switch quadrant {
	case 1: // Верхняя правая четверть
		rowRangeStart = 0
		rowRangeEnd = midRow
		colRangeStart = midCol
		colRangeEnd = N
	case 2: // Верхняя левая четверть
		rowRangeStart = 0
		rowRangeEnd = midRow
		colRangeStart = 0
		colRangeEnd = midCol
	case 3: // Нижняя левая четверть
		rowRangeStart = midRow
		rowRangeEnd = N
		colRangeStart = 0
		colRangeEnd = midCol
	case 4: // Нижняя правая четверть
		rowRangeStart = midRow
		rowRangeEnd = N
		colRangeStart = midCol
		colRangeEnd = N
	}

	// Генерация случайной строки и столбца в заданных пределах
	randomRow := rand.IntN(rowRangeEnd-rowRangeStart) + rowRangeStart
	randomCol := rand.IntN(colRangeEnd-colRangeStart) + colRangeStart

	// Преобразование координат обратно в индекс
	randomIndex := randomRow*N + randomCol

	return randomIndex, nil
}
