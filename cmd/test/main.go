package main

import (
	"time"
)

func main() {
	offset := 3
	shift := 8
	for i := offset; i < 100; i += shift {
		println(i)
		time.Sleep(time.Second)
	}
}
