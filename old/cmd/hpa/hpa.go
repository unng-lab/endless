// Production-ready (skeleton) HPA* + chunked pathfinding system for a streaming infinite world.
// Reworked: replaced grid A* inside chunks with a simple NavMesh generator (merged rectangles -> polygons)
// and A* on polygon graph, using Funnel (string pulling) smoothing across portal edges.
// Single-file demo to bootstrap integration. Split into packages in real project.

package main

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"log/slog"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ----------------------------- CONFIG -------------------------------------
const (
	ChunkSize        = 32 // tiles per chunk (square)
	ClusterChunkSize = 4  // how many chunks per cluster side
	MaxWorkers       = 0  // 0 -> auto (NumCPU - 1)
	CacheCapacity    = 4096
)

// ----------------------------- BASIC TYPES --------------------------------

// Vec2 — целочисленные 2D-координаты. Используются и для мировых координат, и для локальных
// тайловых координат внутри чанка. Методы удобны для преобразований и для funnel
// (FloatX/FloatY) — при вычислениях с плавающей точкой.
type Vec2 struct{ X, Y int }

// Add — простое сложение векторов.
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Eq — сравнение на равенство.
func (v Vec2) Eq(o Vec2) bool { return v.X == o.X && v.Y == o.Y }

// FloatX/FloatY — вспомогательные методы для преобразования в float64 (для funnel).
func (v Vec2) FloatX() float64 { return float64(v.X) }
func (v Vec2) FloatY() float64 { return float64(v.Y) }

// WorldToChunk — переводит глобальную позицию (в целых) в идентификатор чанка и
// локальную позицию внутри чанка. Возвращаемые значения:
//   - ChunkID: координаты чанка в сетке чанков (могут быть отрицательными)
//   - Vec2: локальная позиция (в диапазоне 0..ChunkSize-1 если вход в том же чанке)
//
// Важно: используется floorDiv, чтобы корректно обрабатывать отрицательные координаты.
func WorldToChunk(pos Vec2) (ChunkID, Vec2) {
	x := pos.X
	y := pos.Y
	cx := floorDiv(x, ChunkSize)
	cy := floorDiv(y, ChunkSize)
	lx := x - cx*ChunkSize
	ly := y - cy*ChunkSize
	return ChunkID{cx, cy}, Vec2{lx, ly}
}

// floorDiv — целочисленное деление с округлением вниз (floor), важно при отрицательных
// значениях координат, чтобы чанк слева/снизу от 0 имел индекс -1 и т.д.
func floorDiv(a, b int) int {
	if a >= 0 {
		return a / b
	}
	// negative safe div
	return -((-a + b - 1) / b)
}

// ----------------------------- CHUNKS -------------------------------------

// ChunkID — координаты чанка в логической сетке чанков.
type ChunkID struct{ X, Y int }

// ClusterID — идентификатор кластера (группа чанков). Метод ClusterID вычисляет
// кластер, которому принадлежит чанк, по параметру ClusterChunkSize.
func (c ChunkID) ClusterID() ClusterID {
	// cluster grid groups ClusterChunkSize chunks
	cx := floorDiv(c.X, ClusterChunkSize)
	cy := floorDiv(c.Y, ClusterChunkSize)
	return ClusterID{cx, cy}
}

// Chunk — структура, хранящая данные чанка: идентификатор, версия (инкрементируемая
// при изменениях), occupancy grid и сгенерированный NavMesh (RectPoly список).
type Chunk struct {
	ID      ChunkID
	Version uint64   // increment on change
	Grid    *Grid    // local occupancy grid
	Mesh    *NavMesh // generated navmesh for this chunk
}

// ChunkManager — отвечает за загрузку/генерацию чанков и их NavMesh.
// Поддерживает потоко-безопасный доступ через RWMutex.
type ChunkManager struct {
	mu     sync.RWMutex
	chunks map[ChunkID]*Chunk
}

// NewChunkManager — создаёт пустой менеджер чанков.
func NewChunkManager() *ChunkManager {
	return &ChunkManager{chunks: make(map[ChunkID]*Chunk)}
}

// EnsureLoaded — гарантирует, что чанк с указанным ID загружен.
// Алгоритм:
// 1) Быстрая RLock проверка — если чанк есть, вернуть.
// 2) Если нет — взять Lock, проверить ещё раз (double-check) и сгенерировать:
//   - создать Grid заданного размера
//   - заполнить препятствиями детерминированно по seed, зависящему от координат чанка
//   - построить NavMesh вызовом BuildNavMeshFromGrid
//   - сохранить чанк в map и вернуть
//
// Возвращаемый чанк готов к использованию и содержит NavMesh.
func (m *ChunkManager) EnsureLoaded(id ChunkID) *Chunk {
	m.mu.RLock()
	c := m.chunks[id]
	m.mu.RUnlock()
	if c != nil {
		return c
	}
	// load/generate
	m.mu.Lock()
	defer m.mu.Unlock()
	if c = m.chunks[id]; c != nil { // double-check
		return c
	}
	c = &Chunk{ID: id}
	c.Grid = NewGrid(ChunkSize, ChunkSize)
	// sample generation: random obstacles
	seed := int64((id.X * 73856093) ^ (id.Y * 19349663))
	r := rand.New(rand.NewSource(seed))
	for y := 0; y < ChunkSize; y++ {
		for x := 0; x < ChunkSize; x++ {
			if r.Float64() < 0.12 { // obstacle density
				c.Grid.SetBlocked(x, y, true)
			}
		}
	}
	// build navmesh from grid
	c.Mesh = BuildNavMeshFromGrid(c.Grid, c.ID)
	m.chunks[id] = c
	return c
}

// Unload — выгружает чанк из памяти (удаляет из карты). Не делает дополнительных действий
// (например, сохранения на диск) — в боевом проекте можно добавить write-back.
func (m *ChunkManager) Unload(id ChunkID) {
	m.mu.Lock()
	delete(m.chunks, id)
	m.mu.Unlock()
}

// Get — безопасный ридер для чанка; возвращает nil если чанк не загружен.
func (m *ChunkManager) Get(id ChunkID) *Chunk {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chunks[id]
}

// ----------------------------- CLUSTERS (HPA) ------------------------------

// ClusterID — идентификатор кластера.
type ClusterID struct{ X, Y int }

// Portal — упрощённая структура для описания прохода между двумя кластерами.
// PosA/PosB — примерные мировые координаты по обе стороны портала (используются
// для визуализации/оценки маршрутов на верхнем уровне).
type Portal struct {
	From ClusterID
	To   ClusterID
	PosA Vec2 // world coords (approx)
	PosB Vec2
}

// ClusterGraph — простой meta-graph между кластерами. Для ускорения верхнего уровня
// поиска HPA* мы храним кластеры и набор примитивных порталов между соседями.
type ClusterGraph struct {
	mu       sync.RWMutex
	clusters map[ClusterID]struct{}
	portals  []Portal
}

// NewClusterGraph — конструктор пустого графа кластеров.
func NewClusterGraph() *ClusterGraph {
	return &ClusterGraph{clusters: make(map[ClusterID]struct{})}
}

// EnsureCluster — гарантирует, что кластер существует в графе, и если его нет —
// создаёт записи и простые порталы к 4 соседям (с востока/запада/севера/юга).
// Порталы здесь — упрощение: центр границы между кластерами.
func (g *ClusterGraph) EnsureCluster(cid ClusterID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.clusters[cid]; !ok {
		g.clusters[cid] = struct{}{}
		// generate portals to 4 neighbors
		neis := []ClusterID{{cid.X + 1, cid.Y}, {cid.X - 1, cid.Y}, {cid.X, cid.Y + 1}, {cid.X, cid.Y - 1}}
		for _, n := range neis {
			p := Portal{From: cid, To: n}
			cx := cid.X * ClusterChunkSize * ChunkSize
			cy := cid.Y * ClusterChunkSize * ChunkSize
			if n.X > cid.X { // east
				p.PosA = Vec2{cx + ClusterChunkSize*ChunkSize - 1, cy + ClusterChunkSize*ChunkSize/2}
				p.PosB = Vec2{p.PosA.X + 1, p.PosA.Y}
			} else if n.X < cid.X { // west
				p.PosA = Vec2{cx, cy + ClusterChunkSize*ChunkSize/2}
				p.PosB = Vec2{p.PosA.X - 1, p.PosA.Y}
			} else if n.Y > cid.Y { // south
				p.PosA = Vec2{cx + ClusterChunkSize*ChunkSize/2, cy + ClusterChunkSize*ChunkSize - 1}
				p.PosB = Vec2{p.PosA.X, p.PosA.Y + 1}
			} else { // north
				p.PosA = Vec2{cx + ClusterChunkSize*ChunkSize/2, cy}
				p.PosB = Vec2{p.PosA.X, p.PosA.Y - 1}
			}
			g.portals = append(g.portals, p)
		}
	}
}

// FindHighLevelPath — выполняет A* по сетке кластеров (4-ориентированные соседние клетки).
// Возвращает последовательность ClusterID (путь на высоком уровне). Это быстрый и грубый
// маршрут, который затем уточняется локально внутри кластеров/чанков.
func (g *ClusterGraph) FindHighLevelPath(s, t ClusterID) []ClusterID {
	g.EnsureCluster(s)
	g.EnsureCluster(t)
	type node struct {
		cid  ClusterID
		f, g int
		prev *node
	}
	open := make(clusterPQ, 0)
	heap.Init(&open)
	start := &clusterNode{cid: s, g: 0, f: heuristicCluster(s, t)}
	heap.Push(&open, start)
	closed := make(map[ClusterID]bool)
	nodes := map[ClusterID]*clusterNode{s: start}

	for open.Len() > 0 {
		n := heap.Pop(&open).(*clusterNode)
		if n.cid == t {
			path := []ClusterID{}
			for p := n; p != nil; p = p.prev {
				path = append(path, p.cid)
			}
			reverseCluster(path)
			return path
		}
		closed[n.cid] = true
		dirs := []ClusterID{{n.cid.X + 1, n.cid.Y}, {n.cid.X - 1, n.cid.Y}, {n.cid.X, n.cid.Y + 1}, {n.cid.X, n.cid.Y - 1}}
		for _, nei := range dirs {
			if closed[nei] {
				continue
			}
			ng := n.g + 1
			if ex, ok := nodes[nei]; !ok || ng < ex.g {
				hn := heuristicCluster(nei, t)
				node := &clusterNode{cid: nei, g: ng, f: ng + hn, prev: n}
				nodes[nei] = node
				heap.Push(&open, node)
			}
		}
	}
	return nil
}

// heuristicCluster — манхэттенская эвристика для кластерной сетки (быстрая и допустимая).
func heuristicCluster(a, b ClusterID) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}

func reverseCluster(s []ClusterID) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// ----------------------------- NAVMESH Types -------------------------------

// RectPoly — простая полигональная примитивная (оси-ориентированный прямоугольник),
// описывающая непрерывную область прохода в мировых координатах. Min/Max — включительно.
// Centroid — центр полигона (для эвристик). ID — индекс внутри NavMesh.
type RectPoly struct {
	Min       Vec2 // inclusive tile coords in world space
	Max       Vec2 // inclusive tile coords in world space
	Centroid  Vec2
	ID        int
	Neighbors []int // adjacent polygon IDs
}

// NavMesh — контейнер полигона (в данной реализации — список прямоугольников).
type NavMesh struct {
	Polys []*RectPoly
	// adjacency via Neighbors
}

// BuildNavMeshFromGrid — строит NavMesh из occupancy grid с помощью жадного
// объединения свободных тайлов в максимальные оси-ориентированные прямоугольники.
// Алгоритм:
// 1) сканируем строки, находим максимальную ширину свободного сегмента,
// 2) расширяем его вниз (по высоте), пока каждая следующая строка содержит тот же свободный сегмент,
// 3) помечаем эти тайлы как использованные и создаём RectPoly в координатах мира.
// В конце строим соседство (adjacency) между полученными RectPoly, проверяя смежность/пересечение.
// Ограничения: простая эвристика, даёт прямоугольные полигоны; для сложных контуров
// лучше использовать marching squares / полигонизацию.
func BuildNavMeshFromGrid(g *Grid, id ChunkID) *NavMesh {
	// We'll convert local tiles to world coords by offsetting by chunk origin
	ox := id.X * ChunkSize
	oy := id.Y * ChunkSize

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
			// find width
			w := 1
			for x+w < W && !used[(x+w)+y*W] && !g.Blocked(x+w, y) {
				w++
			}
			// find height by expanding rows with same width
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
			// mark used
			for yy := 0; yy < h; yy++ {
				for xx := 0; xx < w; xx++ {
					used[(x+xx)+(y+yy)*W] = true
				}
			}
			p := &RectPoly{Min: Vec2{ox + x, oy + y}, Max: Vec2{ox + x + w - 1, oy + y + h - 1}}
			p.Centroid = Vec2{(p.Min.X + p.Max.X) / 2, (p.Min.Y + p.Max.Y) / 2}
			p.ID = len(polys)
			polys = append(polys, p)
			x += w
		}
	}
	mesh := &NavMesh{Polys: polys}
	// build adjacency by touching edges
	for i, a := range mesh.Polys {
		for j, b := range mesh.Polys {
			if i == j {
				continue
			}
			if rectsTouch(a, b) {
				a.Neighbors = append(a.Neighbors, b.ID)
			}
		}
	}
	return mesh
}

// rectsTouch — проверяет, соприкасаются ли два прямоугольника по краю (или перекрываются).
// Возвращает true если у них есть смежная граница/перекрытие, пригодная для adjacency.
func rectsTouch(a, b *RectPoly) bool {
	// check if rectangles share edge with overlap >0
	if a.Max.X+1 < b.Min.X || b.Max.X+1 < a.Min.X || a.Max.Y+1 < b.Min.Y || b.Max.Y+1 < a.Min.Y {
		return false
	}
	// they touch or overlap
	if a.Max.X < b.Min.X || b.Max.X < a.Min.X { // vertical neighbors
		// require y-overlap
		return intervalsOverlap(a.Min.Y, a.Max.Y, b.Min.Y, b.Max.Y)
	}
	if a.Max.Y < b.Min.Y || b.Max.Y < a.Min.Y { // horizontal neighbors
		return intervalsOverlap(a.Min.X, a.Max.X, b.Min.X, b.Max.X)
	}
	// else overlap; consider them neighbors
	return true
}

// intervalsOverlap — вспомогательная: проверяет пересечение отрезков [a1,a2] и [b1,b2].
func intervalsOverlap(a1, a2, b1, b2 int) bool {
	if a2 < b1 || b2 < a1 {
		return false
	}
	// require overlap length > 0
	low := max(a1, b1)
	high := min(a2, b2)
	return high-low >= 0
}

// ----------------------------- NAVMESH PATHFINDING ------------------------

// FindPathOnNavMesh — основной метод поиска пути в пределах локальных NavMesh.
// Алгоритм:
//  1. Для start и goal находим соответствующие чанки и их NavMesh.
//  2. Находим полигоны, которые содержат начальную и конечную точку (или ближайшие полигоны).
//  3. Строим граф узлов (polyNode) из полигонов стартового и целевого чанков (в текущей
//     реализации — только из двух чанков; в будущем расширить на соседние чанки).
//  4. Выполняем A* по этому графу (эвристика — расстояние между центроидами).
//  5. Восстанавливаем последовательность полигонов — из неё строим список порталов
//     (пересекающихся сегментов между парами полигонов).
//  6. Пропускаем последовательность порталов через алгоритм funnel (string pulling)
//     вместе со start/goal, чтобы получить сглаженный путь в мировых координатах.
//
// Возвращает последовательность точек Vec2 (мировые координаты) или ошибку.
func FindPathOnNavMesh(cm *ChunkManager, start, goal Vec2) ([]Vec2, error) {
	// find chunk and poly containing start/goal
	cs, _ := WorldToChunk(start)
	ce, _ := WorldToChunk(goal)
	chunkS := cm.EnsureLoaded(cs)
	chunkE := cm.EnsureLoaded(ce)
	meshS := chunkS.Mesh
	meshE := chunkE.Mesh
	// find polygons containing local positions (or nearest)
	startPoly := findPolyContaining(meshS, start)
	goalPoly := findPolyContaining(meshE, goal)
	if startPoly == nil || goalPoly == nil {
		// fallback: if any missing, fail
		return nil, errors.New("start/goal not in navmesh")
	}
	// if same poly
	if startPoly == goalPoly {
		return []Vec2{start, goal}, nil
	}
	// A* on poly graph combining across chunks: we will allow neighbors only within same mesh or adjacent chunk meshes
	// For simplicity, build a list of nodes across the two meshes (could be extended to nearby chunks)
	nodes := map[int]*polyNode{}
	addPoly := func(p *RectPoly) {
		nodes[polyGlobalID(p, cs)] = &polyNode{poly: p, g: int(math.MaxInt32)}
	}
	for _, p := range meshS.Polys {
		addPoly(p)
	}
	if cs != ce {
		for _, p := range meshE.Polys {
			addPoly(p)
		}
	}
	// Build adjacency edges among nodes: include neighbors across meshes if rectangles touch in world coords
	// we can compare all pairs from these meshes
	for _, a := range nodes {
		for _, b := range nodes {
			if a == b {
				continue
			}
			if rectsTouch(a.poly, b.poly) {
				a.neis = append(a.neis, b)
			}
		}
	}
	// find start and goal nodes pointers
	startNode := nodes[polyGlobalID(startPoly, cs)]
	goalNode := nodes[polyGlobalID(goalPoly, ce)]
	if startNode == nil || goalNode == nil {
		return nil, errors.New("nodes missing")
	}
	// A*
	open := make(polyPQ, 0)
	heap.Init(&open)
	startNode.g = 0
	startNode.f = heuristicPoly(startNode, goalNode)
	heap.Push(&open, startNode)
	cameFrom := map[*polyNode]*polyNode{}
	for open.Len() > 0 {
		n := heap.Pop(&open).(*polyNode)
		if n == goalNode { // reconstruct path of polys
			pathPolys := []*RectPoly{}
			for cur := n; cur != nil; cur = cameFrom[cur] {
				pathPolys = append(pathPolys, cur.poly)
			}
			reversePolys(pathPolys)
			// Now build portal list between consecutive polys
			portals := buildPortalsFromPolySequence(pathPolys)
			// apply funnel to portals + start/goal
			path := stringPulling(portals, start, goal)
			return path, nil
		}
		for _, nei := range n.neis {
			ng := n.g + approxCost(n.poly, nei.poly)
			if ng < nei.g {
				nei.g = ng
				nei.f = ng + heuristicPoly(nei, goalNode)
				cameFrom[nei] = n
				heap.Push(&open, nei)
			}
		}
	}
	return nil, errors.New("no path on navmesh")
}

// polyGlobalID — формирует «глобальный» ключ для полигона принимая во внимание его
// локальный ID и координаты чанка. Это нужно, чтобы различать полигоны из разных чанков.
func polyGlobalID(p *RectPoly, chunk ChunkID) int {
	// unique-ish id combining chunk and poly id
	return (chunk.X+10000)*100000 + (chunk.Y+1000)*100 + p.ID
}

// polyNode — узел для A* на уровне полигонов.
type polyNode struct {
	poly *RectPoly
	g, f int
	neis []*polyNode
}

// heuristicPoly — эвристика для узла полигона: манхэттен отцентр to центр (дешёвая и допустимая).
func heuristicPoly(a, b *polyNode) int {
	dx := a.poly.Centroid.X - b.poly.Centroid.X
	dy := a.poly.Centroid.Y - b.poly.Centroid.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}

// approxCost — приближённая стоимость перехода между двумя RectPoly (евклидова длина между центроидами).
func approxCost(a, b *RectPoly) int {
	dx := a.Centroid.X - b.Centroid.X
	dy := a.Centroid.Y - b.Centroid.Y
	return int(math.Hypot(float64(dx), float64(dy)))
}

func reversePolys(s []*RectPoly) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// buildPortalsFromPolySequence: для каждой пары смежных полигонов вычисляет сегмент-портал
// (левая и правая границы) в мировых координатах. Этот набор порталов затем используется
// алгоритмом funnel для сглаживания пути (string pulling).
func buildPortalsFromPolySequence(polys []*RectPoly) [][2]Vec2 {
	if len(polys) == 0 {
		return nil
	}
	portals := make([][2]Vec2, 0, len(polys)-1)
	for i := 0; i < len(polys)-1; i++ {
		a := polys[i]
		b := polys[i+1]
		// compute overlap rectangle
		minx := max(a.Min.X, b.Min.X)
		maxx := min(a.Max.X, b.Max.X)
		miny := max(a.Min.Y, b.Min.Y)
		maxy := min(a.Max.Y, b.Max.Y)
		if minx <= maxx {
			// vertical overlap: portal along X between (minx..maxx) at y = touching edge
			if a.Max.Y < b.Min.Y { // a above b
				p1 := Vec2{minx, a.Max.Y}
				p2 := Vec2{maxx, a.Max.Y}
				portals = append(portals, [2]Vec2{p1, p2})
			} else if b.Max.Y < a.Min.Y { // b above a
				p1 := Vec2{minx, b.Max.Y}
				p2 := Vec2{maxx, b.Max.Y}
				portals = append(portals, [2]Vec2{p1, p2})
			} else {
				// overlapping area; choose segment between centers
				p1 := Vec2{minx, max(min(a.Min.Y, b.Min.Y), min(a.Max.Y, b.Max.Y))}
				p2 := Vec2{maxx, p1.Y}
				portals = append(portals, [2]Vec2{p1, p2})
			}
		} else if miny <= maxy {
			// horizontal overlap
			if a.Max.X < b.Min.X {
				p1 := Vec2{a.Max.X, miny}
				p2 := Vec2{a.Max.X, maxy}
				portals = append(portals, [2]Vec2{p1, p2})
			} else if b.Max.X < a.Min.X {
				p1 := Vec2{b.Max.X, miny}
				p2 := Vec2{b.Max.X, maxy}
				portals = append(portals, [2]Vec2{p1, p2})
			} else {
				p1 := Vec2{max(min(a.Min.X, b.Min.X), min(a.Max.X, b.Max.X)), miny}
				p2 := Vec2{p1.X, maxy}
				portals = append(portals, [2]Vec2{p1, p2})
			}
		} else {
			// fallback: connect centroids
			portals = append(portals, [2]Vec2{a.Centroid, b.Centroid})
		}
	}
	return portals
}

// ----------------------------- FUNNEL (STRING PULLING) --------------------
// convert to float64 points for funnel math
type fpt struct{ X, Y float64 }

// stringPulling — реализация funnel algorithm (string pulling) для набора порталов.
// Вход:
//   - portals: последовательность пар [left,right] точек, задающих «коридор» (в мировых координатах)
//   - start, goal: мировые координаты начала и конца маршрута
//
// Возвращает: упорядоченный список Vec2 (точки пути), которые можно использовать как waypoints для юнита.
// Замечания:
//   - Входные порталы должны задавать корректный проход между полигонами (в идеале от buildPortalsFromPolySequence).
//   - Алгоритм использует float64 для вычислений ориентаций и затем округляет координаты на выходе.
func stringPulling(portals [][2]Vec2, start, goal Vec2) []Vec2 {
	ptsLeft := make([]fpt, 0, len(portals)+2)
	ptsRight := make([]fpt, 0, len(portals)+2)
	// start as single point
	ptsLeft = append(ptsLeft, fpt{start.FloatX(), start.FloatY()})
	ptsRight = append(ptsRight, fpt{start.FloatX(), start.FloatY()})
	for _, p := range portals {
		l := fpt{p[0].FloatX(), p[0].FloatY()}
		r := fpt{p[1].FloatX(), p[1].FloatY()}
		ptsLeft = append(ptsLeft, l)
		ptsRight = append(ptsRight, r)
	}
	ptsLeft = append(ptsLeft, fpt{goal.FloatX(), goal.FloatY()})
	ptsRight = append(ptsRight, fpt{goal.FloatX(), goal.FloatY()})

	apex := fpt{start.FloatX(), start.FloatY()}
	left := fpt{start.FloatX(), start.FloatY()}
	right := fpt{start.FloatX(), start.FloatY()}
	apexIndex, leftIndex, rightIndex := 0, 0, 0
	path := []Vec2{{int(math.Round(apex.X)), int(math.Round(apex.Y))}}

	for i := 1; i < len(ptsLeft); i++ {
		l := ptsLeft[i]
		r := ptsRight[i]
		// update right
		if triArea2(apex, right, r) <= 0 {
			if triArea2(apex, left, r) < 0 {
				// tighten
				r = r
				right = r
				rightIndex = i
			} else {
				// new apex is left
				apex = left
				apexIndex = leftIndex
				path = append(path, Vec2{int(math.Round(apex.X)), int(math.Round(apex.Y))})
				// reset
				left = apex
				right = apex
				leftIndex = apexIndex
				rightIndex = apexIndex
				// restart scanning from apexIndex+1
				i = apexIndex
				continue
			}
		}
		// update left
		if triArea2(apex, left, l) >= 0 {
			if triArea2(apex, right, l) > 0 {
				l = l
				left = l
				leftIndex = i
			} else {
				// new apex is right
				apex = right
				apexIndex = rightIndex
				path = append(path, Vec2{int(math.Round(apex.X)), int(math.Round(apex.Y))})
				left = apex
				right = apex
				leftIndex = apexIndex
				rightIndex = apexIndex
				i = apexIndex
				continue
			}
		}
	}
	// finally append goal
	path = append(path, goal)
	return path
}

// triArea2 — возвращает удвоенную ориентированную площадь треугольника (апекс, a, b).
// Положительное значение — поворот влево, отрицательное — вправо. Используется для тестов ориентации
// в алгоритме funnel.
func triArea2(a, b, c fpt) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

// ----------------------------- PRIORITY QUEUES ----------------------------

// cluster-level node
type clusterNode struct {
	cid  ClusterID
	g, f int
	prev *clusterNode
}

// clusterPQ — простой priority queue для A* на кластерах. Меньше f => выше приоритет.
type clusterPQ []*clusterNode

func (pq clusterPQ) Len() int           { return len(pq) }
func (pq clusterPQ) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq clusterPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *clusterPQ) Push(x any)        { *pq = append(*pq, x.(*clusterNode)) }
func (pq *clusterPQ) Pop() any          { old := *pq; n := old[len(old)-1]; *pq = old[:len(old)-1]; return n }

// poly-level PQ — priority queue для A* по полигонам.
type polyPQ []*polyNode

func (pq polyPQ) Len() int           { return len(pq) }
func (pq polyPQ) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq polyPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *polyPQ) Push(x any)        { *pq = append(*pq, x.(*polyNode)) }
func (pq *polyPQ) Pop() any          { old := *pq; n := old[len(old)-1]; *pq = old[:len(old)-1]; return n }

// ----------------------------- GRID (occupancy) ---------------------------

// Grid represents a simple walkable tilemap; хранит булевую матрицу доступности тайлов.
type Grid struct {
	W, H  int
	cells []byte // 0 walkable, 1 blocked
}

// NewGrid — конструктор occupancy grid размера w x h.
func NewGrid(w, h int) *Grid { return &Grid{W: w, H: h, cells: make([]byte, w*h)} }

// idx — индекс в линейном массиве из координат x,y.
func (g *Grid) idx(x, y int) int { return y*g.W + x }

// InBounds — проверка, что координаты лежат в пределах сетки.
func (g *Grid) InBounds(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }

// SetBlocked — помечает тайл как блокированный/проходимый. Игнорирует выход за границы.
func (g *Grid) SetBlocked(x, y int, b bool) {
	if !g.InBounds(x, y) {
		return
	}
	if b {
		g.cells[g.idx(x, y)] = 1
	} else {
		g.cells[g.idx(x, y)] = 0
	}
}

// Blocked — возвращает true если тайл явно блокирован или находится за пределами сетки.
func (g *Grid) Blocked(x, y int) bool {
	if !g.InBounds(x, y) {
		return true
	}
	return g.cells[g.idx(x, y)] == 1
}

// ----------------------------- PATH CACHE (LRU) ---------------------------

// pathEntry — узел двусвязного списка для LRU-кеша путей.
type pathEntry struct {
	key        uint64
	path       []Vec2
	prev, next *pathEntry
}

// PathCache — простой потокобезопасный LRU-кеш для путей. Использует map + двусвязный список.
type PathCache struct {
	cap        int
	m          map[uint64]*pathEntry
	head, tail *pathEntry
	mu         sync.Mutex
}

// NewPathCache — возвращает новый кеш заданной ёмкости.
func NewPathCache(cap int) *PathCache { return &PathCache{cap: cap, m: make(map[uint64]*pathEntry)} }

// Get — получает путь по ключу; при попадании перемещает запись в head (most-recent).
func (c *PathCache) Get(k uint64) ([]Vec2, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok {
		c.moveToFront(e)
		return e.path, true
	}
	return nil, false
}

// Put — вставляет новый путь или обновляет существующий; при переполнении удаляется oldest.
func (c *PathCache) Put(k uint64, p []Vec2) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok {
		e.path = p
		c.moveToFront(e)
		return
	}
	e := &pathEntry{key: k, path: p}
	c.m[k] = e
	c.addFront(e)
	if len(c.m) > c.cap {
		c.removeOldest()
	}
}

func (c *PathCache) moveToFront(e *pathEntry) {
	if e == c.head {
		return
	}
	c.remove(e)
	c.addFront(e)
}
func (c *PathCache) addFront(e *pathEntry) {
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}
func (c *PathCache) removeOldest() {
	if c.tail != nil {
		c.remove(c.tail)
	}
}
func (c *PathCache) remove(e *pathEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
	delete(c.m, e.key)
}

// ----------------------------- WORKER POOL --------------------------------

// PathRequest — структура запроса: start/goal, канал ответа, context для отмены и приоритет.
// Приоритет пока используется концептуально (lower значит выше приоритет); текущая очередь
// — простая FIFO буферизированная.
type PathRequest struct {
	Start, Goal Vec2
	Resp        chan PathResult
	Ctx         context.Context
	Priority    int // lower = higher
}

// PathResult — результат поиска пути: либо path, либо Err.
type PathResult struct {
	Path []Vec2
	Err  error
}

// Scheduler — планировщик задач поиска пути; поддерживает пул воркеров, кеш и доступ к chunk/cluster.
type Scheduler struct {
	workers int
	tasks   chan *PathRequest
	cm      *ChunkManager
	cg      *ClusterGraph
	cache   *PathCache
	wg      sync.WaitGroup
	closed  int32
}

// NewScheduler — создает Scheduler и стартует воркеры. Если workers <= 0, то
// автоматически выбирается runtime.NumCPU()-1 (но не меньше 1).
func NewScheduler(cm *ChunkManager, cg *ClusterGraph, cache *PathCache, workers int) *Scheduler {
	if workers <= 0 {
		workers = runtime.NumCPU() - 1
		if workers < 1 {
			workers = 1
		}
	}
	s := &Scheduler{workers: workers, tasks: make(chan *PathRequest, 10000), cm: cm, cg: cg, cache: cache}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.workerLoop(i)
	}
	return s
}

// Shutdown — корректно завершает работу Scheduler: помечает closed и закрывает канал задач,
// затем ждёт завершения всех воркеров.
func (s *Scheduler) Shutdown() { atomic.StoreInt32(&s.closed, 1); close(s.tasks); s.wg.Wait() }

// Submit — отправляет запрос на планирование.
// Блокировка: если буфер полон, функция ждёт до 200ms, затем возвращает ошибку "submit timeout".
func (s *Scheduler) Submit(req *PathRequest) error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return errors.New("scheduler closed")
	}
	select {
	case s.tasks <- req:
		return nil
	default:
		select {
		case s.tasks <- req:
			return nil
		case <-time.After(200 * time.Millisecond):
			return errors.New("submit timeout")
		}
	}
}

// workerLoop — основной цикл воркера: берёт запросы из очереди, проверяет context, кеш,
// вычисляет путь (через HPA/кластерный маршрут + локальные уточнения через NavMesh) и
// отправляет результат в канал resp. Логика:
//  1. попытка получить путь из LRU-кеша;
//  2. получить высокоуровневый маршрут кластеров (HPA);
//  3. если HPA отсутствует — попробовать локальный NavMesh поиск напрямую;
//  4. иначе — построить waypoints (центры кластеров) и последовательно решать локальные
//     NavMesh задачи между waypoints (start..waypoints..goal), агрегировать сегменты;
//  5. сохранить результат в кеш и вернуть клиенту.
func (s *Scheduler) workerLoop(id int) {
	defer s.wg.Done()
	for req := range s.tasks {
		if req == nil {
			continue
		}
		if req.Ctx != nil {
			select {
			case <-req.Ctx.Done():
				req.Resp <- PathResult{nil, req.Ctx.Err()}
				continue
			default:
			}
		}
		k := pathKey(req.Start, req.Goal)
		if p, ok := s.cache.Get(k); ok {
			req.Resp <- PathResult{p, nil}
			continue
		}
		// high-level: clusters
		cs, _ := WorldToChunk(req.Start)
		ce, _ := WorldToChunk(req.Goal)
		sc := cs.ClusterID()
		ec := ce.ClusterID()
		clPath := s.cg.FindHighLevelPath(sc, ec)
		if clPath == nil {
			// fallback try navmesh in same chunk
			path, err := FindPathOnNavMesh(s.cm, req.Start, req.Goal)
			if err != nil {
				req.Resp <- PathResult{nil, err}
				continue
			}
			s.cache.Put(k, path)
			req.Resp <- PathResult{path, nil}
			continue
		}
		// build coarse waypoints (cluster centers), then refine locally with navmesh
		waypoints := []Vec2{}
		for _, cid := range clPath {
			cx := cid.X*ClusterChunkSize*ChunkSize + (ClusterChunkSize*ChunkSize)/2
			cy := cid.Y*ClusterChunkSize*ChunkSize + (ClusterChunkSize*ChunkSize)/2
			waypoints = append(waypoints, Vec2{cx, cy})
		}
		// chain local navmesh paths between successive waypoints (and start/goal)
		full := []Vec2{req.Start}
		prev := req.Start
		for _, wp := range append(waypoints, req.Goal) {
			seg, err := FindPathOnNavMesh(s.cm, prev, wp)
			if err != nil {
				// fall back to coarse waypoint
				full = append(full, wp)
				prev = wp
				continue
			}
			// append excluding duplicate start
			if len(seg) > 1 {
				full = append(full, seg[1:]...)
			}
			prev = wp
		}
		s.cache.Put(k, full)
		req.Resp <- PathResult{full, nil}
	}
}

// pathKey — генерирует 64-битный ключ для кеша на основе старт/целей; не криптографичный.
func pathKey(a, b Vec2) uint64 {
	h := fnv.New64a()
	writeVec(h, a)
	writeVec(h, b)
	return h.Sum64()
}
func writeVec(h hash.Hash64, v Vec2) {
	data := []byte(fmt.Sprintf("%d,%d;", v.X, v.Y))
	_, err := h.Write(data)
	if err != nil {
		slog.Error("pathKey: hash.Write", "err", err)
		return
	}
}

// ----------------------------- FLOW FIELD ---------------------------------

// BuildFlowField — строит простое векторное поле (flow field) для прямоугольной области
// [min..max] (в мировых координатах). Поле возвращается в виде map<позиция -> вектор шага>.
// Использует Dijkstra-like approach от цели: вычисляет для каждой клетки направление к соседу
// с наименьшей стоимостью. Подходит для массового движения большого числа юнитов к одной цели.
func BuildFlowField(cm *ChunkManager, min, max Vec2) (map[Vec2]Vec2, error) {
	w := max.X - min.X + 1
	h := max.Y - min.Y + 1
	if w <= 0 || h <= 0 {
		return nil, errors.New("bad region")
	}
	field := make(map[Vec2]Vec2, w*h)
	target := Vec2{max.X, max.Y}
	type cell struct {
		pos  Vec2
		cost int
	}
	pq := make([]cell, 0)
	push := func(c cell) {
		pq = append(pq, c)
		sort.Slice(pq, func(i, j int) bool { return pq[i].cost < pq[j].cost })
	}
	push(cell{target, 0})
	visited := make(map[Vec2]bool)
	for len(pq) > 0 {
		c := pq[0]
		pq = pq[1:]
		if visited[c.pos] {
			continue
		}
		visited[c.pos] = true
		for _, d := range []Vec2{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := Vec2{c.pos.X + d.X, c.pos.Y + d.Y}
			if n.X < min.X || n.Y < min.Y || n.X > max.X || n.Y > max.Y {
				continue
			}
			cid, local := WorldToChunk(n)
			chunk := cm.EnsureLoaded(cid)
			if chunk.Grid.Blocked(local.X, local.Y) {
				continue
			}
			if visited[n] {
				continue
			}
			field[n] = Vec2{-d.X, -d.Y}
			push(cell{n, c.cost + 1})
		}
	}
	return field, nil
}

// ----------------------------- DEMO / BENCH --------------------------------

// main — демонстрация: прогревает кластеры, создаёт Scheduler и отправляет N запросов на поиск пути
// параллельно, затем собирает статистику. Также показывает пример генерации flow field.
func main() {
	fmt.Println("HPA* + NavMesh + Funnel demo")
	cm := NewChunkManager()
	cg := NewClusterGraph()
	cache := NewPathCache(CacheCapacity)
	sched := NewScheduler(cm, cg, cache, MaxWorkers)
	defer sched.Shutdown()

	// Warm clusters
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			cg.EnsureCluster(ClusterID{x, y})
		}
	}

	// Run concurrent path requests to simulate many agents
	N := 1000
	requests := make([]*PathRequest, N)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	for i := 0; i < N; i++ {
		sx := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		sy := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		gx := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		gy := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		req := &PathRequest{Start: Vec2{sx, sy}, Goal: Vec2{gx, gy}, Resp: make(chan PathResult, 1), Ctx: ctx}
		requests[i] = req
		err := sched.Submit(req)
		if err != nil {
			fmt.Println("submit err", err)
			requests[i] = nil
		}
	}

	ok := 0
	totalNodes := 0
	for i := 0; i < N; i++ {
		r := requests[i]
		if r == nil {
			continue
		}
		select {
		case res := <-r.Resp:
			if res.Err != nil { /*fmt.Println("path err", res.Err)*/
			} else {
				ok++
				totalNodes += len(res.Path)
			}
		case <-ctx.Done():
			fmt.Println("timeout waiting for responses")
			break
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("Requests: %d ok=%d elapsed=%v avgPathLen=%.2f", N, ok, elapsed, float64(totalNodes)/float64(max(1, ok)))

	// flowfield demo for an area
	fmt.Println("Building flow field for area 128x128")
	min := Vec2{-64, -64}
	max := Vec2{63, 63}
	ff, err := BuildFlowField(cm, min, max)
	if err != nil {
		fmt.Println("flow err", err)
		return
	}
	fmt.Println("flow size", len(ff))
}

// ----------------------------- Helpers ------------------------------------

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return a
}

// findPolyContaining — находит полигон в mesh, содержащий точку w. Если нет точного попадания,
// возвращает ближайший полигон по расстоянию до центроида (используется как fallback).
func findPolyContaining(mesh *NavMesh, w Vec2) *RectPoly {
	if mesh == nil {
		return nil
	}
	for _, p := range mesh.Polys {
		if w.X >= p.Min.X && w.X <= p.Max.X && w.Y >= p.Min.Y && w.Y <= p.Max.Y {
			return p
		}
	}
	// if none contains, return nearest by centroid
	if len(mesh.Polys) == 0 {
		return nil
	}
	best := mesh.Polys[0]
	bestD := dist2(best.Centroid, w)
	for _, p := range mesh.Polys[1:] {
		d := dist2(p.Centroid, w)
		if d < bestD {
			best = p
			bestD = d
		}
	}
	return best
}

func dist2(a, b Vec2) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

// ----------------------------- NOTES -------------------------------------

// This file implements a practical NavMesh approach for tile-based chunks by merging free tiles into
// axis-aligned rectangles (RectPoly). It then builds adjacency between polygons and performs A*
// on polygon graph. For smoothing we compute portal segments between adjacent polys and apply the
// funnel (string pulling) algorithm to produce a smooth path.

// Production considerations (next steps):
//  - Replace rectangle merging with more advanced polygonization (marching squares -> polygon simplification)
//  - Support adjacency across multiple neighboring chunks when path crosses chunk boundaries
//  - Cache NavMesh per chunk and update incrementally on dynamic changes
//  - Optimize poly graph search by precomputing cluster-level meta nodes (portals between clusters)
//  - Add tests for funnel correctness and path validity
//  - Replace simple centroid heuristics with euclidean distances and admissible heuristics
