// scheduler/scheduler_test.go
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/unng-lab/madfarmer/internal/cache"
	"github.com/unng-lab/madfarmer/internal/chunk"
	"github.com/unng-lab/madfarmer/internal/geom"
	"github.com/unng-lab/madfarmer/internal/hpa"
)

func TestSchedulerBasics(t *testing.T) {
	// Создаем компоненты
	cm := chunk.NewChunkManager()
	cg := hpa.NewClusterGraph()
	c := cache.NewPathCache(100)

	// Создаем планировщик
	s := NewScheduler(cm, cg, c, 2)
	defer s.Shutdown()

	// Создаем запрос
	req := &PathRequest{
		Start: geom.Vec2{0, 0},
		Goal:  geom.Vec2{10, 10},
		Resp:  make(chan PathResult, 1),
		Ctx:   context.Background(),
	}

	// Отправляем запрос
	if err := s.Submit(req); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Ждем результата
	select {
	case res := <-req.Resp:
		if res.Err != nil {
			t.Fatalf("Pathfinding error: %v", res.Err)
		}
		if len(res.Path) < 2 {
			t.Error("Path should have at least start and goal points")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for path result")
	}
}

func TestSchedulerCancellation(t *testing.T) {
	cm := chunk.NewChunkManager()
	cg := hpa.NewClusterGraph()
	c := cache.NewPathCache(100)

	s := NewScheduler(cm, cg, c, 2)
	defer s.Shutdown()

	// Создаем запрос с отменой
	ctx, cancel := context.WithCancel(context.Background())
	req := &PathRequest{
		Start: geom.Vec2{0, 0},
		Goal:  geom.Vec2{100, 100},
		Resp:  make(chan PathResult, 1),
		Ctx:   ctx,
	}

	// Отправляем запрос
	if err := s.Submit(req); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Отменяем запрос
	cancel()

	// Проверяем, что запрос был отменен
	select {
	case res := <-req.Resp:
		if res.Err == nil {
			t.Error("Expected error for canceled request, got nil")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for canceled request result")
	}
}

func TestSchedulerShutdown(t *testing.T) {
	cm := chunk.NewChunkManager()
	cg := hpa.NewClusterGraph()
	c := cache.NewPathCache(100)

	s := NewScheduler(cm, cg, c, 1)

	// Отправляем запрос
	req := &PathRequest{
		Start: geom.Vec2{0, 0},
		Goal:  geom.Vec2{10, 10},
		Resp:  make(chan PathResult, 1),
		Ctx:   context.Background(),
	}

	if err := s.Submit(req); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Закрываем планировщик
	s.Shutdown()

	// Попытка отправить новый запрос после закрытия
	req2 := &PathRequest{
		Start: geom.Vec2{0, 0},
		Goal:  geom.Vec2{20, 20},
		Resp:  make(chan PathResult, 1),
		Ctx:   context.Background(),
	}

	if err := s.Submit(req2); err == nil {
		t.Error("Submit after shutdown should fail")
	}
}
