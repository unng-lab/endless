package unit

import (
	"sync"
)

type WG struct {
	WG *sync.WaitGroup
	S  int
}

func (wg *WG) Done(sleeper int) {
	wg.S = sleeper
	wg.WG.Done()
}
