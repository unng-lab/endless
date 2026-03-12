package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/unng-lab/endless/cmd/internal/launcher"
	"github.com/unng-lab/endless/pkg/endless"
)

func main() {
	startedAt := time.Now()
	log.Printf("[startup] launcher: stress process started")

	flagsStartedAt := time.Now()
	runConfig := launcher.ParseRunConfig()
	log.Printf("[startup] launcher: command-line flags parsed in %s", time.Since(flagsStartedAt))

	profilerStartedAt := time.Now()
	profilerSession, err := launcher.StartProfiler(runConfig.Profiling)
	if err != nil {
		log.Fatalf("start profiler: %v", err)
	}
	log.Printf("[startup] launcher: profiler configured in %s", time.Since(profilerStartedAt))
	defer func() {
		if stopErr := profilerSession.Stop(); stopErr != nil {
			log.Printf("stop profiler: %v", stopErr)
		}
	}()

	windowStartedAt := time.Now()
	ebiten.SetWindowTitle("Endless Stress")
	ebiten.SetWindowSize(endless.DefaultScreenWidth, endless.DefaultScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(false)
	ebiten.SetVsyncEnabled(false)
	log.Printf("[startup] launcher: window configured in %s", time.Since(windowStartedAt))

	gameStartedAt := time.Now()
	game, err := endless.NewStressGame()
	if err != nil {
		log.Fatalf("create stress game: %v", err)
	}
	log.Printf("[startup] launcher: NewStressGame completed in %s", time.Since(gameStartedAt))
	log.Printf("[startup] launcher: entering ebiten.RunGame after %s total startup prep", time.Since(startedAt))

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("run endless stress: %v", err)
	}
}
