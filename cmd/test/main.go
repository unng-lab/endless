package main

import (
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

func main() {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	println("mem.Alloc-1 = ", mem.TotalAlloc)
	m := make(map[int]int)
	s := make([]int, 0, 10000)
	go func() {
		for i := range 10000 {
			m[i] = i
			s = append(s, i)
		}
		println("mem.Alloc0 = ", mem.TotalAlloc)
		println(len(m))
		println(GetMapSize(m))

		println(len(s))
		println(GetSliceSize(s))
		runtime.GC()
		println("mem.Alloc1 = ", mem.TotalAlloc)
		for ii := range 5000 {
			delete(m, ii)
		}
		println(len(m))
		println(GetMapSize(m))
		runtime.GC()
		println("mem.Alloc2 = ", mem.TotalAlloc)

		clear(m)
		clear(s)
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		println("mem.Alloc3 = ", mem.TotalAlloc)
		println(len(m))
		println(GetMapSize(m))

		println(len(s))
		println(GetSliceSize(s))

		m = nil
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		println("mem.Alloc4 = ", mem.TotalAlloc)
	}()

	time.Sleep(5 * time.Second)

}

func GetMapSize(m interface{}) uintptr {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		panic("Expected a map")
	}

	var size uintptr
	for _, key := range v.MapKeys() {
		size += unsafe.Sizeof(key.Interface())             // размер ключа
		size += unsafe.Sizeof(v.MapIndex(key).Interface()) // размер значения
	}
	// Добавляем размер самой мапы
	size += unsafe.Sizeof(v.Interface())

	return size
}

func GetSliceSize(slice interface{}) uintptr {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("Expected a slice")
	}

	var size uintptr

	// Добавляем размер структуры среза
	size += unsafe.Sizeof(v)

	// Размер элементов
	for i := 0; i < v.Len(); i++ {
		size += unsafe.Sizeof(v.Index(i).Interface())
	}

	return size
}
