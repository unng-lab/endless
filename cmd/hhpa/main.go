package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/unng-lab/madfarmer/internal/cache"
	"github.com/unng-lab/madfarmer/internal/chunk"
	"github.com/unng-lab/madfarmer/internal/flowfield"
	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/hpa"
	"github.com/unng-lab/madfarmer/internal/scheduler"
)

// ----------------------------- DEMO / BENCH --------------------------------
// ----------------------------- CONFIG -------------------------------------
const (
	ChunkSize        = 32 // tiles per chunk (square)
	ClusterChunkSize = 4  // how many chunks per cluster side
	MaxWorkers       = 0  // 0 -> auto (NumCPU - 1)
	CacheCapacity    = 4096
)

// main — демонстрация: прогревает кластеры, создаёт Scheduler и отправляет N запросов на поиск пути
// параллельно, затем собирает статистику. Также показывает пример генерации flow field.
func main() {
	fmt.Println("HPA* + NavMesh + Funnel demo")
	cm := chunk.NewChunkManager()
	cg := hpa.NewClusterGraph()
	cache := cache.NewPathCache(CacheCapacity)
	sched := scheduler.NewScheduler(cm, cg, cache, MaxWorkers)
	defer sched.Shutdown()

	// Warm clusters
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			cg.EnsureCluster(geom.ClusterID{x, y})
		}
	}

	// Run concurrent path requests to simulate many agents
	N := 1000
	requests := make([]*scheduler.PathRequest, N)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	for i := 0; i < N; i++ {
		sx := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		sy := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		gx := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		gy := rand.Intn(ClusterChunkSize*ChunkSize*8) - ClusterChunkSize*ChunkSize*4
		req := &scheduler.PathRequest{Start: geom.Vec2{sx, sy}, Goal: geom.Vec2{gx, gy}, Resp: make(chan scheduler.PathResult, 1), Ctx: ctx}
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
	min := geom.Vec2{-64, -64}
	max := geom.Vec2{63, 63}
	ff, err := flowfield.BuildFlowField(cm, min, max)
	if err != nil {
		fmt.Println("flow err", err)
		return
	}
	fmt.Println("flow size", len(ff))
}
