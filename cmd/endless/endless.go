package main

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/endless"
)

func main() {
	ebiten.SetWindowSize(800, 800)
	//ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MadFarmer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	go func() {
		for {
			time.Sleep(10 * time.Second)
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			slog.Info("current",
				"goroutines", runtime.NumGoroutine(),
				"memory in mb", m.Alloc/1024/1024,
				"last gc was", time.Now().Sub(time.Unix(0, int64(m.LastGC))),
			)
		}
	}()
	err := ebiten.RunGame(endless.NewGame())
	if err != nil {
		panic(err)
	}
}
