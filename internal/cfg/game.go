package cfg

import (
	"github/unng-lab/madfarmer/internal/game"
)

var _ game.Config = (*Default)(nil)

type gameConfig struct {
}
