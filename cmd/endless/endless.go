package main

import (
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/unng-lab/endless/cmd/internal/launcher"
	"github.com/unng-lab/endless/pkg/endless"
	gamescenario "github.com/unng-lab/endless/pkg/endless/scenario"
	"github.com/unng-lab/endless/pkg/rl"
)

func main() {
	startedAt := time.Now()
	log.Printf("[startup] launcher: process started")

	sceneMode := string(gamescenario.ModeBasic)
	rlScenario := rl.DuelScenarioOpen
	rlPolicy := rl.PolicyLeadAndStrafe
	rlSeed := int64(1)
	rlModelPath := ""
	rlMaxTicks := int64(1200)
	flag.StringVar(&sceneMode, "scene", sceneMode, "scene bootstrap mode: basic or rl_duel")
	flag.StringVar(&rlScenario, "rl-scenario", rlScenario, "visual rl duel layout: duel_open or duel_with_cover")
	flag.StringVar(&rlPolicy, "rl-policy", rlPolicy, "visual rl duel shooter policy: lead_strafe or random")
	flag.Int64Var(&rlSeed, "rl-seed", rlSeed, "seed for visual rl duel layout and stochastic policies")
	flag.StringVar(&rlModelPath, "rl-model-path", "", "optional path to a saved runtime model artifact; accepts train-stub JSON, GoMLX manifest JSON, GoMLX checkpoint JSON/.bin, or a GoMLX checkpoint directory, and overrides -rl-policy")
	flag.Int64Var(&rlMaxTicks, "rl-max-ticks", rlMaxTicks, "tick budget for one visual rl duel episode before it times out")

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
	ebiten.SetWindowTitle("Endless")
	ebiten.SetWindowSize(endless.DefaultScreenWidth, endless.DefaultScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(false)
	ebiten.SetVsyncEnabled(false)
	log.Printf("[startup] launcher: window configured in %s", time.Since(windowStartedAt))

	gameStartedAt := time.Now()
	game, err := endless.NewGameWithConfig(endless.GameConfig{
		Mode: gamescenario.Mode(sceneMode),
		RLDuel: rl.VisualDuelScenarioConfig{
			Scenario:  rlScenario,
			Policy:    rlPolicy,
			Seed:      rlSeed,
			ModelPath: rlModelPath,
			MaxTicks:  rlMaxTicks,
		},
	})
	if err != nil {
		log.Fatalf("create game: %v", err)
	}
	log.Printf("[startup] launcher: NewGame completed in %s", time.Since(gameStartedAt))
	log.Printf("[startup] launcher: entering ebiten.RunGame after %s total startup prep", time.Since(startedAt))

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("run endless: %v", err)
	}
}
