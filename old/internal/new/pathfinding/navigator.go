package pathfinding

import (
	"container/heap"
	"math"

	"github.com/unng-lab/endless/internal/new/tilemap"
)

// Point represents discrete tile coordinates on the map.
type Point struct {
	X int
	Y int
}

// Navigator implements hierarchical path-finding tailored for very large maps.
type Navigator struct {
	tiles       *tilemap.TileMap
	clusterSize int
	walkable    map[clusterCoord]bool
}

// clusterCoord represents coordinates in the coarse cluster grid.
type clusterCoord struct {
	X int
	Y int
}

// NewNavigator builds a Navigator using the provided tile map. clusterSize controls
// how many tiles belong to a single high-level cluster. Larger values reduce the
// size of the search graph while keeping local refinements efficient.
func NewNavigator(m *tilemap.TileMap, clusterSize int) *Navigator {
	if clusterSize <= 0 {
		clusterSize = 32
	}
	return &Navigator{
		tiles:       m,
		clusterSize: clusterSize,
		walkable:    make(map[clusterCoord]bool),
	}
}

// FindPath calculates a route between two tile coordinates. It returns the
// inclusive path as a slice of points. The algorithm first performs a coarse
// search on cluster cells and then refines each step with an A* search inside
// the union of the current and next clusters. This dramatically reduces the
// amount of nodes explored for very large maps while still producing detailed
// paths.
func (n *Navigator) FindPath(start, goal Point) ([]Point, bool) {
	if start == goal {
		return []Point{start}, true
	}

	if !n.tiles.IsWalkable(start.X, start.Y) || !n.tiles.IsWalkable(goal.X, goal.Y) {
		return nil, false
	}

	clusters := n.highLevelPath(start, goal)
	if len(clusters) == 0 {
		return nil, false
	}

	result := make([]Point, 0, len(clusters)*n.clusterSize)
	current := start

	for i := 0; i < len(clusters); i++ {
		var target Point
		if i == len(clusters)-1 {
			target = goal
		} else {
			var ok bool
			target, ok = n.portalPoint(clusters[i], clusters[i+1])
			if !ok {
				return nil, false
			}
		}

		next, hasNext := nextCluster(clusters, i)
		segment := n.localPath(current, target, clusters[i], next, hasNext)
		if len(segment) == 0 {
			return nil, false
		}

		if len(result) > 0 {
			segment = segment[1:]
		}
		result = append(result, segment...)
		current = target
	}

	return result, true
}

func nextCluster(path []clusterCoord, index int) (clusterCoord, bool) {
	if index+1 >= len(path) {
		return clusterCoord{}, false
	}
	return path[index+1], true
}

func (n *Navigator) localPath(start, goal Point, current, next clusterCoord, hasNext bool) []Point {
	minX := current.X * n.clusterSize
	maxX := (current.X + 1) * n.clusterSize
	minY := current.Y * n.clusterSize
	maxY := (current.Y + 1) * n.clusterSize

	if hasNext {
		minX = minInt(minX, next.X*n.clusterSize)
		minY = minInt(minY, next.Y*n.clusterSize)
		maxX = maxInt(maxX, (next.X+1)*n.clusterSize)
		maxY = maxInt(maxY, (next.Y+1)*n.clusterSize)
	}

	minX = clampInt(minX, 0, n.tiles.Columns())
	minY = clampInt(minY, 0, n.tiles.Rows())
	maxX = clampInt(maxX, 0, n.tiles.Columns())
	maxY = clampInt(maxY, 0, n.tiles.Rows())

	bounds := rect{minX: minX, minY: minY, maxX: maxX, maxY: maxY}
	return n.aStar(start, goal, bounds)
}

func (n *Navigator) clusterOf(p Point) clusterCoord {
	return clusterCoord{X: p.X / n.clusterSize, Y: p.Y / n.clusterSize}
}

func (n *Navigator) highLevelPath(start, goal Point) []clusterCoord {
	startCluster := n.clusterOf(start)
	goalCluster := n.clusterOf(goal)
	if startCluster == goalCluster {
		return []clusterCoord{startCluster}
	}

	open := make(clusterQueue, 0)
	heap.Init(&open)

	startNode := &clusterNode{coord: startCluster, g: 0, f: heuristicCluster(startCluster, goalCluster)}
	heap.Push(&open, startNode)

	cameFrom := map[clusterCoord]*clusterNode{startCluster: startNode}
	closed := make(map[clusterCoord]bool)

	for open.Len() > 0 {
		current := heap.Pop(&open).(*clusterNode)
		if current.coord == goalCluster {
			return reconstructClusterPath(current)
		}

		closed[current.coord] = true

		for _, nei := range neighbors(current.coord) {
			if closed[nei] {
				continue
			}
			if !n.clusterWalkable(nei) {
				continue
			}

			if _, ok := n.portalPoint(current.coord, nei); !ok {
				continue
			}

			tentativeG := current.g + 1
			if existing, ok := cameFrom[nei]; !ok || tentativeG < existing.g {
				node := &clusterNode{
					coord: nei,
					g:     tentativeG,
					f:     tentativeG + heuristicCluster(nei, goalCluster),
					prev:  current,
				}
				cameFrom[nei] = node
				heap.Push(&open, node)
			}
		}
	}

	return nil
}

func (n *Navigator) clusterWalkable(c clusterCoord) bool {
	if v, ok := n.walkable[c]; ok {
		return v
	}

	minX := c.X * n.clusterSize
	minY := c.Y * n.clusterSize
	maxX := minX + n.clusterSize
	maxY := minY + n.clusterSize

	for y := minY; y < maxY; y++ {
		if y < 0 || y >= n.tiles.Rows() {
			continue
		}
		for x := minX; x < maxX; x++ {
			if x < 0 || x >= n.tiles.Columns() {
				continue
			}
			if n.tiles.IsWalkable(x, y) {
				n.walkable[c] = true
				return true
			}
		}
	}

	n.walkable[c] = false
	return false
}

func (n *Navigator) portalPoint(from, to clusterCoord) (Point, bool) {
	// clusters are expected to be neighbors. Determine shared border.
	dx := to.X - from.X
	dy := to.Y - from.Y

	minX := maxInt(from.X, to.X) * n.clusterSize
	minY := maxInt(from.Y, to.Y) * n.clusterSize

	if dx == 1 {
		x := to.X * n.clusterSize
		for y := minY; y < minY+n.clusterSize; y++ {
			if n.tiles.IsWalkable(x, y) {
				return Point{X: clampInt(x, 0, n.tiles.Columns()-1), Y: clampInt(y, 0, n.tiles.Rows()-1)}, true
			}
		}
	} else if dx == -1 {
		x := clampInt(from.X*n.clusterSize-1, 0, n.tiles.Columns()-1)
		for y := minY; y < minY+n.clusterSize; y++ {
			if n.tiles.IsWalkable(x, y) {
				return Point{X: x, Y: clampInt(y, 0, n.tiles.Rows()-1)}, true
			}
		}
	} else if dy == 1 {
		y := to.Y * n.clusterSize
		for x := minX; x < minX+n.clusterSize; x++ {
			if n.tiles.IsWalkable(x, y) {
				return Point{X: clampInt(x, 0, n.tiles.Columns()-1), Y: y}, true
			}
		}
	} else if dy == -1 {
		y := clampInt(from.Y*n.clusterSize-1, 0, n.tiles.Rows()-1)
		for x := minX; x < minX+n.clusterSize; x++ {
			if n.tiles.IsWalkable(x, y) {
				return Point{X: clampInt(x, 0, n.tiles.Columns()-1), Y: y}, true
			}
		}
	}

	return Point{}, false
}

func neighbors(c clusterCoord) []clusterCoord {
	return []clusterCoord{
		{X: c.X + 1, Y: c.Y},
		{X: c.X - 1, Y: c.Y},
		{X: c.X, Y: c.Y + 1},
		{X: c.X, Y: c.Y - 1},
	}
}

func reconstructClusterPath(n *clusterNode) []clusterCoord {
	path := []clusterCoord{}
	for current := n; current != nil; current = current.prev {
		path = append(path, current.coord)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

type clusterNode struct {
	coord clusterCoord
	g     int
	f     int
	prev  *clusterNode
}

type clusterQueue []*clusterNode

func (pq clusterQueue) Len() int { return len(pq) }

func (pq clusterQueue) Less(i, j int) bool { return pq[i].f < pq[j].f }

func (pq clusterQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *clusterQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*clusterNode))
}

func (pq *clusterQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[:n-1]
	return x
}

func heuristicCluster(a, b clusterCoord) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}

type rect struct {
	minX int
	minY int
	maxX int
	maxY int
}

func (n *Navigator) aStar(start, goal Point, bounds rect) []Point {
	open := make(tileQueue, 0)
	heap.Init(&open)

	startNode := &tileNode{point: start, g: 0, f: heuristicTile(start, goal)}
	heap.Push(&open, startNode)

	cameFrom := map[Point]*tileNode{start: startNode}
	closed := make(map[Point]bool)

	for open.Len() > 0 {
		current := heap.Pop(&open).(*tileNode)
		if current.point == goal {
			return reconstructTilePath(current)
		}

		closed[current.point] = true

		for _, nei := range tileNeighbors(current.point) {
			if nei.X < bounds.minX || nei.X >= bounds.maxX || nei.Y < bounds.minY || nei.Y >= bounds.maxY {
				continue
			}
			if closed[nei] || !n.tiles.IsWalkable(nei.X, nei.Y) {
				continue
			}

			tentativeG := current.g + 1
			if existing, ok := cameFrom[nei]; !ok || tentativeG < existing.g {
				node := &tileNode{
					point: nei,
					g:     tentativeG,
					f:     tentativeG + heuristicTile(nei, goal),
					prev:  current,
				}
				cameFrom[nei] = node
				heap.Push(&open, node)
			}
		}
	}

	return nil
}

type tileNode struct {
	point Point
	g     int
	f     int
	prev  *tileNode
}

type tileQueue []*tileNode

func (pq tileQueue) Len() int { return len(pq) }

func (pq tileQueue) Less(i, j int) bool { return pq[i].f < pq[j].f }

func (pq tileQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *tileQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*tileNode))
}

func (pq *tileQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[:n-1]
	return x
}

func reconstructTilePath(n *tileNode) []Point {
	path := []Point{}
	for current := n; current != nil; current = current.prev {
		path = append(path, current.point)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func tileNeighbors(p Point) []Point {
	return []Point{
		{X: p.X + 1, Y: p.Y},
		{X: p.X - 1, Y: p.Y},
		{X: p.X, Y: p.Y + 1},
		{X: p.X, Y: p.Y - 1},
	}
}

func heuristicTile(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
