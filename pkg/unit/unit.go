package unit

type Kind string

const (
	KindRunner        Kind = "runner"
	KindRunnerFocused Kind = "runnerfocused"
	KindWall          Kind = "wall"
	KindBarricade     Kind = "barricade"
	KindProjectile    Kind = "projectile"
)

var runnerAnimation = Animation{
	FrameCount: 8,
	FrameTicks: 6,
}
