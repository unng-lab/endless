// cache/cache.go
package cache

import (
	"hash"
	"hash/fnv"
	"sync"

	"github.com/unng-lab/madfarmer/internal/geom"
)

// PathCache — LRU-кеш для путей
type PathCache struct {
	cap        int
	mutex      sync.Mutex
	m          map[uint64]*pathEntry
	head, tail *pathEntry
}

type pathEntry struct {
	key        uint64
	path       []geom.Vec2
	prev, next *pathEntry
}

// NewPathCache создает новый кеш заданной емкости
func NewPathCache(cap int) *PathCache {
	return &PathCache{
		cap: cap,
		m:   make(map[uint64]*pathEntry),
	}
}

// Get получает путь по ключу
func (c *PathCache) Get(k uint64) ([]geom.Vec2, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if e, ok := c.m[k]; ok {
		c.moveToFront(e)
		return e.path, true
	}
	return nil, false
}

// Put вставляет новый путь в кеш
func (c *PathCache) Put(k uint64, p []geom.Vec2) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
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

// PathKey генерирует ключ для кеша на основе старт/целей
func PathKey(a, b geom.Vec2) uint64 {
	h := fnv.New64a()
	writeVec(h, a)
	writeVec(h, b)
	return h.Sum64()
}

func writeVec(h hash.Hash64, v geom.Vec2) {
	data := []byte{byte(v.X), byte(v.X >> 8), byte(v.X >> 16), byte(v.X >> 24),
		byte(v.Y), byte(v.Y >> 8), byte(v.Y >> 16), byte(v.Y >> 24)}
	h.Write(data)
}
