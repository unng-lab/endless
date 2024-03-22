package main

import (
	"context"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/robbert229/fxslog"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github/unng-lab/madfarmer/internal/cfg"
	"github/unng-lab/madfarmer/internal/ebitenfx"
	"github/unng-lab/madfarmer/internal/game"
	"github/unng-lab/madfarmer/internal/scr"
	"github/unng-lab/madfarmer/internal/slogfx"
	"github/unng-lab/madfarmer/internal/units/runner"
	"github/unng-lab/madfarmer/internal/window"
)

func main() {
	app := fx.New(
		fx.Provide(
			fx.Annotate(
				cfg.New,
				fx.As(new(ui_old.Config)),
				fx.As(new(ebitenfx.Config)),
				fx.As(new(game.Config)),
				fx.As(new(slogfx.Config)),
				fx.As(new(scr.Config)),
				fx.As(new(window.Config)),
				fx.As(new(runner.Config)),
			),
			fx.Annotate(
				ebitenfx.New,
				fx.As(new(ebiten.Game)),
			),
			fx.Annotate(
				ui_old.New,
				fx.As(new(ebitenfx.UI)),
			),
			fx.Annotate(
				scr.New,
				fx.As(new(ebitenfx.Screen)),
			),
			window.New,
			slogfx.New,
			game.New,
			runner.New,
		),
		fx.Invoke(
			ebiten.RunGame,
		),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxslog.SlogLogger{
				Logger: logger,
			}
		}),
	)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		slogfx.Fatal("app.Start", err)
	}
	<-app.Done()

	if err := app.Stop(ctx); err != nil {
		slogfx.Fatal("app.Stop", err)
	}
}
