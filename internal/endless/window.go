package endless

import "sync/atomic"

var W Window

type Window struct {
	Width  atomic.Int64
	Height atomic.Int64
}

func (w *Window) ViewPortCenter(dir bool) (float64, float64) {
	if !dir {
		return float64(-w.Width.Load()) / 2, float64(-w.Height.Load()) / 2
	}
	return float64(w.Width.Load() / 2), float64(w.Height.Load() / 2)
}

func (w *Window) GetWidth() int64 {
	return w.Width.Load()
}

func (w *Window) Pos(perX, perY float32) (int, int) {
	return w.PosX(perX), w.PosY(perY)
}
func (w *Window) PosX(perX float32) int {
	return int(float32(w.Width.Load()) * perX / 100)
}
func (w *Window) PosY(perY float32) int {
	return int(float32(w.Height.Load()) * perY / 100)
}

func (w *Window) GetHeight() int64 {
	return w.Height.Load()
}
