package unit

// Animation describes a looped sprite animation.
type Animation struct {
	FrameCount int
	FrameTicks int
}

// frameAt resolves the current sprite frame from a pure tick counter so animation timing stays
// in the same unit system as movement and sleep scheduling.
func (a Animation) frameAt(animationTicks int) int {
	if a.FrameCount <= 0 {
		return 0
	}

	frameTicks := a.FrameTicks
	if frameTicks <= 0 {
		frameTicks = 1
	}

	return (animationTicks / frameTicks) % a.FrameCount
}
