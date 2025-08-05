package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/endless"
)

func main() {
	//go StartPProfHttp()
	ebiten.SetWindowSize(800, 800)
	//ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MadFarmer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetVsyncEnabled(false)
	go func() {
		for {
			time.Sleep(10 * time.Second)
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			slog.Info("current",
				"goroutines", runtime.NumGoroutine(),
				"memory in mb sys", m.Sys/1024/1024,
				"memory in mb", m.Alloc/1024/1024,
				"last gc was", time.Now().Sub(time.Unix(0, int64(m.LastGC))),
			)
		}
	}()

	if err := ebiten.RunGame(
		endless.NewGame(),
	); err != nil {
		panic(err)
	}
}

func StartPProfHttp() {
	err := http.ListenAndServe("localhost:38080", nil)
	if err != nil {
		return
	}
}
