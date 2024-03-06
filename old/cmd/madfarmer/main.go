package main

import (
	"context"
	"log"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github/unng-lab/madfarmer/internal/scr"
)

func main() {
	app := fx.New(
		fx.Provide(
			zap.NewExample,
			scr.New,
		),
		fx.Invoke(
		//ebitenfx.RunGame,
		),
		fx.WithLogger(
			func(logger *zap.Logger) fxevent.Logger {
				return &fxevent.ZapLogger{Logger: logger}
			},
		),
	)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	<-app.Done()

	if err := app.Stop(ctx); err != nil {
		log.Fatal(err)
	}
}
