package slogfx

import (
	"log/slog"
	"os"
)

type Config interface {
}

func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

}

func Fatal(msg string, err error) {
	slog.Error(msg, err.Error())
	os.Exit(1)
}
