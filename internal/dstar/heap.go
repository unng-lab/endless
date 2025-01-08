package dstar

func (ds *DStar) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !ds.Less(j, i) {
			break
		}
		ds.Swap(i, j)
		j = i
	}
}

func (ds *DStar) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && ds.Less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !ds.Less(j, i) {
			break
		}
		ds.Swap(i, j)
		i = j
	}
	return i > i0
}

func (ds *DStar) Len() int { return len(ds.nodes) }

func (ds *DStar) Less(i, j int) bool {
	if ds.nodes[i].Key[0] == ds.nodes[j].Key[0] {
		return ds.nodes[i].Key[1] < ds.nodes[j].Key[1]
	}
	return ds.nodes[i].Key[0] < ds.nodes[j].Key[0]
}

func (ds *DStar) Swap(i, j int) {
	ds.nodes[i], ds.nodes[j] = ds.nodes[j], ds.nodes[i]
	ds.nodes[i].Index = i
	ds.nodes[j].Index = j
}

func (ds *DStar) Push(node *Node) {
	node.Index = len(ds.nodes)
	ds.nodes = append(ds.nodes, node)
	ds.up(node.Index)
}

func (ds *DStar) Pop() *Node {
	n := ds.Len() - 1
	ds.Swap(0, n)
	ds.down(0, n)
	node := ds.nodes[len(ds.nodes)-1]
	ds.nodes = ds.nodes[0 : len(ds.nodes)-1]
	node.Index = -1
	return node
}

// Fix re-establishes the heap ordering after the element at index i has changed its value.
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling [Remove](h, i) followed by a Push of the new value.
// The complexity is O(log n) where n = h.Len().
func (ds *DStar) Fix(i int) {
	if !ds.down(i, ds.Len()) {
		ds.up(i)
	}
}

// Remove removes and returns the element at index i from the heap.
// The complexity is O(log n) where n = h.Len().
func (ds *DStar) Remove(i int) *Node {
	n := ds.Len() - 1
	if n != i {
		ds.Swap(i, n)
		if !ds.down(i, n) {
			ds.up(i)
		}
	}
	return ds.Pop()
}
