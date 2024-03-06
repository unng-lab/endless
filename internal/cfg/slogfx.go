package cfg

import (
	"github/unng-lab/madfarmer/internal/slogfx"
)

var _ slogfx.Config = (*Default)(nil)

type slogfxConfig struct {
}
