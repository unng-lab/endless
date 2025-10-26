// cache/cache_test.go
package cache

import (
	"testing"

	"github.com/unng-lab/endless/internal/geom"
)

func TestPathCache(t *testing.T) {
	c := NewPathCache(2)

	// Добавляем пути
	path1 := []geom.Vec2{{0, 0}, {1, 1}, {2, 2}}
	path2 := []geom.Vec2{{0, 0}, {0, 1}, {0, 2}}
	path3 := []geom.Vec2{{1, 1}, {2, 2}, {3, 3}}

	key1 := PathKey(geom.Vec2{0, 0}, geom.Vec2{2, 2})
	key2 := PathKey(geom.Vec2{0, 0}, geom.Vec2{0, 2})
	key3 := PathKey(geom.Vec2{1, 1}, geom.Vec2{3, 3})

	c.Put(key1, path1)
	c.Put(key2, path2)

	// Проверяем получение
	if p, ok := c.Get(key1); !ok || len(p) != 3 {
		t.Error("Get failed for key1")
	}

	if p, ok := c.Get(key2); !ok || len(p) != 3 {
		t.Error("Get failed for key2")
	}

	// Добавляем третий путь, должен вытеснить первый (т.к. кеш размером 2)
	c.Put(key3, path3)

	if _, ok := c.Get(key1); ok {
		t.Error("Get failed: key1 should have been evicted")
	}

	if p, ok := c.Get(key2); !ok || len(p) != 3 {
		t.Error("Get failed for key2 after adding key3")
	}

	if p, ok := c.Get(key3); !ok || len(p) != 3 {
		t.Error("Get failed for key3")
	}
}

func TestPathKey(t *testing.T) {
	// Проверяем, что одинаковые пары дают одинаковые ключи
	key1 := PathKey(geom.Vec2{1, 2}, geom.Vec2{3, 4})
	key2 := PathKey(geom.Vec2{1, 2}, geom.Vec2{3, 4})

	if key1 != key2 {
		t.Error("PathKey failed: same inputs should produce same key")
	}

	// Проверяем, что разные пары дают разные ключи
	key3 := PathKey(geom.Vec2{2, 3}, geom.Vec2{4, 5})

	if key1 == key3 {
		t.Error("PathKey failed: different inputs should produce different keys")
	}
}
