package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/unng-lab/endless/pkg/endless"
)

func main() {
	runConfig := parseRunConfig()
	profilerSession, err := startProfiler(runConfig.profiling)
	if err != nil {
		log.Fatalf("start profiler: %v", err)
	}
	defer func() {
		if stopErr := profilerSession.Stop(); stopErr != nil {
			log.Printf("stop profiler: %v", stopErr)
		}
	}()

	ebiten.SetWindowTitle("Endless")
	ebiten.SetWindowSize(endless.DefaultScreenWidth, endless.DefaultScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(false)
	ebiten.SetVsyncEnabled(false)

	if err := ebiten.RunGame(endless.NewGame()); err != nil {
		log.Fatalf("run endless: %v", err)
	}
}

type runConfig struct {
	profiling profilingConfig
}

// parseRunConfig centralizes all command-line flags for the desktop launcher so profiling can
// be enabled for stress runs without touching the game code or recompiling between sessions.
func parseRunConfig() runConfig {
	config := runConfig{}
	flag.StringVar(&config.profiling.cpuProfilePath, "cpuprofile", "", "write CPU profile to file")
	flag.StringVar(&config.profiling.heapProfilePath, "memprofile", "", "write heap profile to file on shutdown")
	flag.StringVar(&config.profiling.tracePath, "traceprofile", "", "write runtime trace to file")
	flag.StringVar(&config.profiling.pprofAddress, "pprof", "", "serve net/http/pprof on address, for example 127.0.0.1:6060")
	flag.Parse()
	return config
}
