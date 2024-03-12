package endless

var W Window

type Window struct {
	Width  float64
	Height float64
}

func (w *Window) ViewPortCenter(dir bool) (float64, float64) {
	if !dir {
		return (-w.Width) / 2, (-w.Height) / 2
	}
	return w.Width / 2, w.Height / 2

}

func (w *Window) GetWidth() float64 {
	return w.Width
}
func (w *Window) GetHeight() float64 {
	return w.Height
}

func (w *Window) Pos(perX, perY float64) (float64, float64) {
	return w.PosX(perX), w.PosY(perY)
}
func (w *Window) PosX(perX float64) float64 {
	return (w.Width) * perX / 100
}
func (w *Window) PosY(perY float64) float64 {
	return (w.Height) * perY / 100
}
