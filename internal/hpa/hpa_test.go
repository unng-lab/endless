// hpa/hpa_test.go
package hpa

import (
	"testing"

	"github.com/unng-lab/madfarmer/internal/geom"
)

func TestClusterGraph(t *testing.T) {
	g := NewClusterGraph()

	// Проверяем создание кластера
	cid := geom.ClusterID{0, 0}
	g.EnsureCluster(cid)

	if _, ok := g.clusters[cid]; !ok {
		t.Error("EnsureCluster failed: cluster not added to map")
	}

	// Проверяем наличие порталов к соседям
	if len(g.portals) != 4 {
		t.Errorf("EnsureCluster failed: expected 4 portals, got %d", len(g.portals))
	}

	// Проверяем наличие порталов к каждому соседу
	hasPortal := func(from, to geom.ClusterID) bool {
		for _, p := range g.portals {
			if p.From == from && p.To == to {
				return true
			}
		}
		return false
	}

	neighbors := []geom.ClusterID{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	}

	for _, n := range neighbors {
		if !hasPortal(cid, n) {
			t.Errorf("EnsureCluster failed: missing portal to %v", n)
		}
	}
}

func TestFindHighLevelPath(t *testing.T) {
	g := NewClusterGraph()

	// Создаем путь от (0,0) до (2,2)
	path := g.FindHighLevelPath(geom.ClusterID{0, 0}, geom.ClusterID{2, 2})

	if path == nil {
		t.Fatal("FindHighLevelPath failed: returned nil path")
	}

	if len(path) != 5 { // (0,0) -> (1,0) -> (1,1) -> (2,1) -> (2,2)
		t.Errorf("FindHighLevelPath failed: expected path length 5, got %d", len(path))
	}

	// Проверяем начало и конец пути
	if !path[0].Eq(geom.ClusterID{0, 0}) {
		t.Errorf("FindHighLevelPath failed: path should start at (0,0), got %v", path[0])
	}
	if !path[len(path)-1].Eq(geom.ClusterID{2, 2}) {
		t.Errorf("FindHighLevelPath failed: path should end at (2,2), got %v", path[len(path)-1])
	}

	// Проверяем, что путь корректный (каждый следующий кластер соседствует с предыдущим)
	for i := 1; i < len(path); i++ {
		dx := path[i].X - path[i-1].X
		dy := path[i].Y - path[i-1].Y
		if !(dx == 0 && dy == 1 || dx == 0 && dy == -1 || dx == 1 && dy == 0 || dx == -1 && dy == 0) {
			t.Errorf("FindHighLevelPath failed: invalid step from %v to %v", path[i-1], path[i])
		}
	}
}
