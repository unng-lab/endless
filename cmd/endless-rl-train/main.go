package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/unng-lab/endless/pkg/rl"
)

func main() {
	config := rl.DuelRunConfig{}
	flag.IntVar(&config.Episodes, "episodes", 100, "number of duel episodes to generate")
	flag.Int64Var(&config.MaxTicksPerEpisode, "max-ticks", 600, "maximum simulation ticks per episode")
	flag.Int64Var(&config.Seed, "seed", time.Now().UnixNano(), "seed for deterministic duel generation")
	flag.IntVar(&config.WorldColumns, "world-columns", 64, "world column count for duel episodes")
	flag.IntVar(&config.WorldRows, "world-rows", 64, "world row count for duel episodes")
	flag.Float64Var(&config.TileSize, "tile-size", 16, "world tile size for duel episodes")
	flag.Parse()

	clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
	ctx := context.Background()
	recorder, err := rl.NewClickHouseRecorder(ctx, clickhouseConfig)
	if err != nil {
		log.Fatalf("create clickhouse recorder: %v", err)
	}

	if err := rl.RunDuelCollection(ctx, config, recorder); err != nil {
		log.Fatalf("run duel collection: %v", err)
	}
}
