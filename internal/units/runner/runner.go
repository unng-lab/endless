package runner

import (
	"log/slog"

	"github/unng-lab/madfarmer/internal/game"
)

var _ game.Unit = (*Default)(nil)

const id = "runner"

type Default struct {
	log *slog.Logger
	cfg Config
}

type Config interface {
}

func (d *Default) ID() string {
	return id
}

func New(log *slog.Logger, cfg Config) *Default {
	var d Default
	d.log = log.With("runner", "Default")
	d.cfg = cfg

	return &d
}
