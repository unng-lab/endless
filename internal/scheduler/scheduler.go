// scheduler/scheduler.go
package scheduler

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unng-lab/madfarmer/internal/cache"
	"github.com/unng-lab/madfarmer/internal/chunk"
	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/hpa"
	"github.com/unng-lab/madfarmer/internal/pathfinding"
)

// PathRequest — запрос на поиск пути
type PathRequest struct {
	Start, Goal geom.Vec2
	Resp        chan PathResult
	Ctx         context.Context
	Priority    int // lower = higher
}

// PathResult — результат поиска пути
type PathResult struct {
	Path []geom.Vec2
	Err  error
}

// Scheduler — планировщик задач поиска пути
type Scheduler struct {
	workers int
	tasks   chan *PathRequest
	cm      *chunk.ChunkManager
	cg      *hpa.ClusterGraph
	cache   *cache.PathCache
	wg      sync.WaitGroup
	closed  int32
}

// NewScheduler создает Scheduler и запускает воркеры
func NewScheduler(cm *chunk.ChunkManager, cg *hpa.ClusterGraph, cache *cache.PathCache, workers int) *Scheduler {
	if workers <= 0 {
		workers = runtime.NumCPU() - 1
		if workers < 1 {
			workers = 1
		}
	}

	s := &Scheduler{
		workers: workers,
		tasks:   make(chan *PathRequest, 10000),
		cm:      cm,
		cg:      cg,
		cache:   cache,
	}

	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.workerLoop(i)
	}

	return s
}

// Shutdown завершает работу Scheduler
func (s *Scheduler) Shutdown() {
	atomic.StoreInt32(&s.closed, 1)
	close(s.tasks)
	s.wg.Wait()
}

// Submit отправляет запрос на планирование
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

// workerLoop — основной цикл воркера
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

		k := cache.PathKey(req.Start, req.Goal)
		if p, ok := s.cache.Get(k); ok {
			req.Resp <- PathResult{p, nil}
			continue
		}

		// Высокоуровневый поиск через кластеры
		cs, _ := geom.WorldToChunk(req.Start, chunk.ChunkSize)
		ce, _ := geom.WorldToChunk(req.Goal, chunk.ChunkSize)
		sc := cs.ClusterID()
		ec := ce.ClusterID()

		clPath := s.cg.FindHighLevelPath(sc, ec)
		if clPath == nil {
			// Попытка прямого поиска в одном чанке
			path, err := pathfinding.FindPathOnNavMesh(s.cm, req.Start, req.Goal)
			if err != nil {
				req.Resp <- PathResult{nil, err}
				continue
			}
			s.cache.Put(k, path)
			req.Resp <- PathResult{path, nil}
			continue
		}

		// Построение промежуточных точек (центры кластеров)
		waypoints := []geom.Vec2{}
		for _, cid := range clPath {
			cx := cid.X*hpa.ClusterChunkSize*chunk.ChunkSize + (hpa.ClusterChunkSize*chunk.ChunkSize)/2
			cy := cid.Y*hpa.ClusterChunkSize*chunk.ChunkSize + (hpa.ClusterChunkSize*chunk.ChunkSize)/2
			waypoints = append(waypoints, geom.Vec2{cx, cy})
		}

		// Последовательное решение локальных задач
		full := []geom.Vec2{req.Start}
		prev := req.Start
		for _, wp := range append(waypoints, req.Goal) {
			seg, err := pathfinding.FindPathOnNavMesh(s.cm, prev, wp)
			if err != nil {
				// Используем промежуточную точку как waypoint
				full = append(full, wp)
				prev = wp
				continue
			}
			// Добавляем сегмент, исключая дубликат начала
			if len(seg) > 1 {
				full = append(full, seg[1:]...)
			}
			prev = wp
		}

		s.cache.Put(k, full)
		req.Resp <- PathResult{full, nil}
	}
}
