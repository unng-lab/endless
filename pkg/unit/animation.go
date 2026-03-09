package unit

// Animation describes a looped sprite animation.
type Animation struct {
	FrameCount    int
	FrameDuration float64
}

func (a Animation) frameAt(elapsed float64) int {
	if a.FrameCount <= 0 {
		return 0
	}

	frameDuration := a.FrameDuration
	if frameDuration <= 0 {
		frameDuration = 0.1
	}

	return int(elapsed/frameDuration) % a.FrameCount
}
