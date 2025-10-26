package endless

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/assets/img"
	"github.com/unng-lab/endless/internal/board"
	"github.com/unng-lab/endless/internal/camera"
	"github.com/unng-lab/endless/internal/unit"
)

const (
	testTileSize  = 16
	testTileCount = 8
)

func newTestBoard(t *testing.T) (*board.Board, *camera.Camera) {
	t.Helper()
	cam := camera.New(testTileSize, testTileCount)
	cam.W.Width = 640
	cam.W.Height = 480

	b, err := board.NewBoard(cam, testTileSize, testTileSize, testTileCount)
	if err != nil {
		t.Fatalf("failed to create board: %v", err)
	}
	return b, cam
}

func TestNextSerialMonotonic(t *testing.T) {
	unitSerial.Store(0)
	first := nextSerial()
	second := nextSerial()
	if first != 0 || second != 1 {
		t.Fatalf("unexpected serial sequence: got (%d, %d)", first, second)
	}
}

func TestSliceSpriteFrames(t *testing.T) {
	sprite, err := img.Img("runner.png", 256, 96)
	if err != nil {
		t.Fatalf("failed to load sprite: %v", err)
	}

	frames := sliceSpriteFrames(sprite, frameCount)
	if len(frames) != frameCount {
		t.Fatalf("expected %d frames, got %d", frameCount, len(frames))
	}

	seen := make(map[*ebiten.Image]struct{}, frameCount)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame %d is nil", i)
		}
		bounds := frame.Bounds()
		if width, height := bounds.Dx(), bounds.Dy(); width != frameWidth || height != frameHeight {
			t.Fatalf("frame %d has unexpected size %dx%d", i, width, height)
		}
		if _, ok := seen[frame]; ok {
			t.Fatalf("frame %d reused image pointer", i)
		}
		seen[frame] = struct{}{}
	}
}

func TestNewInventoryCreatesPieces(t *testing.T) {
	unitSerial.Store(0)
	board, cam := newTestBoard(t)
	inv := NewInventory(board, cam)

	runner, ok := inv.Units["runner"]
	if !ok || runner == nil {
		t.Fatal("runner piece not registered in inventory")
	}
	rock, ok := inv.Units["rock"]
	if !ok || rock == nil {
		t.Fatal("rock piece not registered in inventory")
	}
	if len(inv.Units) != 2 {
		t.Fatalf("inventory should contain exactly two entries, got %d", len(inv.Units))
	}
}

func TestRunnerUnitCreation(t *testing.T) {
	unitSerial.Store(0)
	board, cam := newTestBoard(t)
	inv := NewInventory(board, cam)
	runnerPiece := inv.Units["runner"]

	tickCh := make(chan int64, 1)
	cameraTickCh := make(chan struct{}, 1)
	name := "Test Runner"
	u := runnerPiece.Unit(5, name, cameraTickCh, tickCh)

	if u.Index != 5 {
		t.Fatalf("unexpected index: %d", u.Index)
	}
	if u.Name != name {
		t.Fatalf("unexpected name: %s", u.Name)
	}
	if u.Type != "runner" {
		t.Fatalf("unexpected type: %s", u.Type)
	}
	expectedSpeed := 1 / float64(ebiten.DefaultTPS) / slowness
	if math.Abs(u.Speed-expectedSpeed) > 1e-9 {
		t.Fatalf("unexpected speed: got %f want %f", u.Speed, expectedSpeed)
	}
	if len(u.Graphics.Animation) != frameCount || len(u.Graphics.FocusedAnimation) != frameCount {
		t.Fatalf("runner animations should contain %d frames", frameCount)
	}
	if u.ID != 0 {
		t.Fatalf("first unit ID should start at 0, got %d", u.ID)
	}

	other := runnerPiece.Unit(6, "Next", cameraTickCh, tickCh)
	if other.ID == u.ID {
		t.Fatal("unit IDs must be unique")
	}
	if other.Index != 6 {
		t.Fatalf("unexpected index for second unit: %d", other.Index)
	}
}

func TestRockUnitCreation(t *testing.T) {
	unitSerial.Store(0)
	board, cam := newTestBoard(t)
	inv := NewInventory(board, cam)
	rockPiece := inv.Units["rock"]

	tickCh := make(chan int64, 1)
	cameraTickCh := make(chan struct{}, 1)
	unit := rockPiece.Unit(2, "Rock", cameraTickCh, tickCh)

	if unit.Type != "rock" {
		t.Fatalf("unexpected type: %s", unit.Type)
	}
	if unit.Graphics == nil || len(unit.Graphics.Animation) != 1 {
		t.Fatalf("rock should have single-frame animation")
	}
	if unit.Positioning.SizeX != 1 || unit.Positioning.SizeY != 1 {
		t.Fatalf("rock should occupy one tile, got %fx%f", unit.Positioning.SizeX, unit.Positioning.SizeY)
	}
}

func TestPieceBaseCreatesUnits(t *testing.T) {
	unitSerial.Store(0)
	board, cam := newTestBoard(t)
	graphics := &unit.Graphics{Animation: []*ebiten.Image{}, FocusedAnimation: []*ebiten.Image{}}
	base := pieceBase{
		name:        "custom",
		board:       board,
		camera:      cam,
		graphics:    graphics,
		positioning: unit.Positioning{SizeX: 1, SizeY: 1},
		speed:       0.5,
	}

	tickCh := make(chan int64, 1)
	cameraTickCh := make(chan struct{}, 1)
	u := base.createUnit(3, "Name", cameraTickCh, tickCh)

	if u.Board != board || u.Camera != cam {
		t.Fatal("unit should reference provided board and camera")
	}
	if u.Speed != 0.5 {
		t.Fatalf("unexpected speed: %f", u.Speed)
	}
	if u.Tasks.Tasks == nil {
		t.Fatal("unit tasks slice should be initialized")
	}
}
