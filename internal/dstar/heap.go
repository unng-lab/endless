package dstar

func (ds *Dstar) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !ds.Less(j, i) {
			break
		}
		ds.Swap(i, j)
		j = i
	}
}

func (ds *Dstar) down(i0, n int) bool {
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
