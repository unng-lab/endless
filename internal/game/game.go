package game

import (
	"log/slog"

	"github/unng-lab/madfarmer/internal/scr"
)

type Config interface {
}

type MadFarmer struct {
	log    *slog.Logger
	cfg    Config
	units  map[string]Unit
	canvas *scr.Canvas
}

type Unit interface {
	ID() string
}

func New(log *slog.Logger, cfg Config, canvas *scr.Canvas, units ...Unit) *MadFarmer {
	var mf MadFarmer
	mf.log = log.With("game", "MadFarmer")
	mf.cfg = cfg
	mf.units = make(map[string]Unit, len(units))
	for i := range units {
		mf.units[units[i].ID()] = units[i]
	}
	mf.canvas = canvas
	return &mf
}

func (mf *MadFarmer) Run() {
	mf.canvas.Put()
}
