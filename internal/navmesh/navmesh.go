package navmesh

import (
	"github.com/unng-lab/endless/internal/chunk"
	"github.com/unng-lab/endless/internal/geom"
)

// RectPoly — прямоугольный полигон, описывающий область прохода
type RectPoly struct {
	Min       geom.Vec2 // включительные координаты в мировом пространстве
	Max       geom.Vec2 // включительные координаты в мировом пространстве
	Centroid  geom.Vec2
	ID        int
	Neighbors []int // ID смежных полигонов
}

// NavMesh — контейнер полигонов
type NavMesh struct {
	Polys []*RectPoly
}

// BuildNavMeshFromGrid строит NavMesh из occupancy grid
func BuildNavMeshFromGrid(g *chunk.Grid, id geom.ChunkID) *NavMesh {
	// Преобразуем локальные тайлы в мировые координаты
	ox := id.X * chunk.ChunkSize
	oy := id.Y * chunk.ChunkSize
	W, H := g.W, g.H
	used := make([]bool, W*H)
	polys := []*RectPoly{}

	for y := 0; y < H; y++ {
		x := 0
		for x < W {
			if used[x+y*W] || g.Blocked(x, y) {
				x++
				continue
			}
			// Находим ширину
			w := 1
			for x+w < W && !used[(x+w)+y*W] && !g.Blocked(x+w, y) {
				w++
			}
			// Находим высоту
			h := 1
		outer:
			for y+h < H {
				for xi := 0; xi < w; xi++ {
					if used[(x+xi)+(y+h)*W] || g.Blocked(x+xi, y+h) {
						break outer
					}
				}
				h++
			}
			// Помечаем использованные тайлы
			for yy := 0; yy < h; yy++ {
				for xx := 0; xx < w; xx++ {
					used[(x+xx)+(y+yy)*W] = true
				}
			}
			p := &RectPoly{
				Min:      geom.Vec2{ox + x, oy + y},
				Max:      geom.Vec2{ox + x + w - 1, oy + y + h - 1},
				Centroid: geom.Vec2{(ox + x + ox + x + w - 1) / 2, (oy + y + oy + y + h - 1) / 2},
				ID:       len(polys),
			}
			polys = append(polys, p)
			x += w
		}
	}

	mesh := &NavMesh{Polys: polys}
	// Строим смежность
	for i, a := range mesh.Polys {
		for j, b := range mesh.Polys {
			if i == j {
				continue
			}
			if RectsTouch(a, b) {
				a.Neighbors = append(a.Neighbors, b.ID)
			}
		}
	}
	return mesh
}

// RectsTouch проверяет, соприкасаются ли два прямоугольника
func RectsTouch(a, b *RectPoly) bool {
	// Проверяем, что прямоугольники пересекаются или касаются
	if a.Max.X+1 < b.Min.X || b.Max.X+1 < a.Min.X || a.Max.Y+1 < b.Min.Y || b.Max.Y+1 < a.Min.Y {
		return false
	}
	// Они касаются или пересекаются
	if a.Max.X < b.Min.X || b.Max.X < a.Min.X { // вертикальные соседи
		return IntervalsOverlap(a.Min.Y, a.Max.Y, b.Min.Y, b.Max.Y)
	}
	if a.Max.Y < b.Min.Y || b.Max.Y < a.Min.Y { // горизонтальные соседи
		return IntervalsOverlap(a.Min.X, a.Max.X, b.Min.X, b.Max.X)
	}
	// Перекрываются, считаем их соседями
	return true
}

// IntervalsOverlap проверяет пересечение отрезков
func IntervalsOverlap(a1, a2, b1, b2 int) bool {
	if a2 < b1 || b2 < a1 {
		return false
	}
	low := geom.Max(a1, b1)
	high := geom.Min(a2, b2)
	return high-low >= 0
}
