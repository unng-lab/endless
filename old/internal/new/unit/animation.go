package unit

// Animation describes a sprite animation sequence.
type Animation struct {
	Frames        []int
	FrameDuration float64 // seconds per frame
}

// frameCount returns number of frames in animation.
func (a Animation) frameCount() int {
	return len(a.Frames)
}

// frameAt returns index within Frames for elapsed time.
func (a Animation) frameAt(elapsed float64) int {
	count := a.frameCount()
	if count == 0 {
		return 0
	}
	duration := a.FrameDuration
	if duration <= 0 {
		duration = 0.1
	}
	frame := int(elapsed/duration) % count
	return frame
}
