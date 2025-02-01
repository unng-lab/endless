package main

import (
	"fmt"
	"iter"
)

func main() {
	for p := range genFib() {
		fmt.Println(p)

		if p > 1000 {
			break
		}
	}
}

func genFib() iter.Seq[int] {
	return func(yield func(int) bool) {
		a, b := 1, 1

		for {
			if !yield(a) {
				return
			}
			a, b = b, a+b
		}
	}
}
