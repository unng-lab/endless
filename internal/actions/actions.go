package actions

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func Update() {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		slog.Info("Left Mouse")
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		slog.Info("Right Mouse")
	}
}
