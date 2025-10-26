// hpa/hpa.go
package hpa

import (
	"container/heap"
	"math"
	"sync"

	"github.com/unng-lab/endless/internal/chunk"
	"github.com/unng-lab/endless/internal/geom"
)

const ClusterChunkSize = 4 // сколько чанков на сторону кластера

// ClusterGraph — граф кластеров для HPA*
type ClusterGraph struct {
	m        sync.Mutex
	clusters map[geom.ClusterID]struct{}
	portals  []Portal
}

// Portal — проход между двумя кластерами
type Portal struct {
	From, To geom.ClusterID
	PosA     geom.Vec2 // мировые координаты (приблизительно)
	PosB     geom.Vec2
}

// NewClusterGraph создает пустой граф кластеров
func NewClusterGraph() *ClusterGraph {
	return &ClusterGraph{clusters: make(map[geom.ClusterID]struct{})}
}

// EnsureCluster гарантирует существование кластера в графе
func (g *ClusterGraph) EnsureCluster(cid geom.ClusterID) {
	g.m.Lock()
	defer g.m.Unlock()
	if _, ok := g.clusters[cid]; ok {
		return
	}

	g.clusters[cid] = struct{}{}

	// Генерируем порталы к 4 соседям
	neis := []geom.ClusterID{
		{cid.X + 1, cid.Y},
		{cid.X - 1, cid.Y},
		{cid.X, cid.Y + 1},
		{cid.X, cid.Y - 1},
	}

	for _, n := range neis {
		p := Portal{From: cid, To: n}
		cx := cid.X * ClusterChunkSize * chunk.ChunkSize
		cy := cid.Y * ClusterChunkSize * chunk.ChunkSize

		if n.X > cid.X { // восток
			p.PosA = geom.Vec2{cx + ClusterChunkSize*chunk.ChunkSize - 1, cy + ClusterChunkSize*chunk.ChunkSize/2}
			p.PosB = geom.Vec2{p.PosA.X + 1, p.PosA.Y}
		} else if n.X < cid.X { // запад
			p.PosA = geom.Vec2{cx, cy + ClusterChunkSize*chunk.ChunkSize/2}
			p.PosB = geom.Vec2{p.PosA.X - 1, p.PosA.Y}
		} else if n.Y > cid.Y { // юг
			p.PosA = geom.Vec2{cx + ClusterChunkSize*chunk.ChunkSize/2, cy + ClusterChunkSize*chunk.ChunkSize - 1}
			p.PosB = geom.Vec2{p.PosA.X, p.PosA.Y + 1}
		} else { // север
			p.PosA = geom.Vec2{cx + ClusterChunkSize*chunk.ChunkSize/2, cy}
			p.PosB = geom.Vec2{p.PosA.X, p.PosA.Y - 1}
		}

		g.portals = append(g.portals, p)
	}
}

// FindHighLevelPath выполняет A* по сетке кластеров
func (g *ClusterGraph) FindHighLevelPath(s, t geom.ClusterID) []geom.ClusterID {
	g.EnsureCluster(s)
	g.EnsureCluster(t)

	open := make(clusterPQ, 0)
	heap.Init(&open)

	start := &node{cid: s, g: 0, f: heuristicCluster(s, t)}
	heap.Push(&open, start)

	closed := make(map[geom.ClusterID]bool)
	nodes := map[geom.ClusterID]*node{s: start}

	for open.Len() > 0 {
		n := heap.Pop(&open).(*node)
		if n.cid == t {
			path := []geom.ClusterID{}
			for p := n; p != nil; p = p.prev {
				path = append(path, p.cid)
			}
			reverseCluster(path)
			return path
		}

		closed[n.cid] = true
		dirs := []geom.ClusterID{
			{n.cid.X + 1, n.cid.Y},
			{n.cid.X - 1, n.cid.Y},
			{n.cid.X, n.cid.Y + 1},
			{n.cid.X, n.cid.Y - 1},
		}

		for _, nei := range dirs {
			if closed[nei] {
				continue
			}

			ng := n.g + 1
			if ex, ok := nodes[nei]; !ok || ng < ex.g {
				hn := heuristicCluster(nei, t)
				node := &node{cid: nei, g: ng, f: ng + hn, prev: n}
				nodes[nei] = node
				heap.Push(&open, node)
			}
		}
	}

	return nil
}

// heuristicCluster — манхэттенская эвристика для кластерной сетки
func heuristicCluster(a, b geom.ClusterID) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}

func reverseCluster(s []geom.ClusterID) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

type node struct {
	cid  geom.ClusterID
	f, g int
	prev *node
}

// clusterPQ — очередь с приоритетом для A* на кластерах
type clusterPQ []*node

func (pq clusterPQ) Len() int           { return len(pq) }
func (pq clusterPQ) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq clusterPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *clusterPQ) Push(x interface{}) {
	*pq = append(*pq, x.(*node))
}

func (pq *clusterPQ) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}
