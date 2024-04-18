package main

import "fmt"

func countUniqueElements(arr []int) int {
	uniqueElements := make(map[int]bool)
	for _, num := range arr {
		uniqueElements[num] = true
	}
	return len(uniqueElements)
}

func main() {
	numbers := []int{1, 2, 3, 4, 2, 3, 1, 5}
	fmt.Printf("Количество уникальных элементов: %d\n", countUniqueElements(numbers))
}
