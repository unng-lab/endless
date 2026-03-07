// flowfield/flowfield.go
package flowfield

import (
	"errors"

	"github.com/unng-lab/endless/internal/chunk"
	"github.com/unng-lab/endless/internal/geom"
)

// BuildFlowField строит поле потока для прямоугольной области
func BuildFlowField(cm *chunk.ChunkManager, min, max geom.Vec2) (map[geom.Vec2]geom.Vec2, error) {
	w := max.X - min.X + 1
	h := max.Y - min.Y + 1
	if w <= 0 || h <= 0 {
		return nil, errors.New("bad region")
	}

	field := make(map[geom.Vec2]geom.Vec2, w*h)
	target := geom.Vec2{max.X, max.Y}

	type cell struct {
		pos  geom.Vec2
		cost int
	}

	pq := []cell{{target, 0}}
	visited := make(map[geom.Vec2]bool)

	for len(pq) > 0 {
		// Извлекаем ячейку с минимальной стоимостью
		minIndex := 0
		for i := 1; i < len(pq); i++ {
			if pq[i].cost < pq[minIndex].cost {
				minIndex = i
			}
		}
		c := pq[minIndex]
		pq = append(pq[:minIndex], pq[minIndex+1:]...)

		if visited[c.pos] {
			continue
		}
		visited[c.pos] = true

		// Проверяем соседей
		for _, d := range []geom.Vec2{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := geom.Vec2{c.pos.X + d.X, c.pos.Y + d.Y}

			// Проверяем границы области
			if n.X < min.X || n.Y < min.Y || n.X > max.X || n.Y > max.Y {
				continue
			}

			// Проверяем проходимость
			cid, local := geom.WorldToChunk(n, chunk.ChunkSize)
			chunk := cm.EnsureLoaded(cid)
			if chunk.Grid.Blocked(local.X, local.Y) {
				continue
			}

			if visited[n] {
				continue
			}

			// Устанавливаем направление
			field[n] = geom.Vec2{-d.X, -d.Y}
			pq = append(pq, cell{n, c.cost + 1})
		}
	}

	return field, nil
}
