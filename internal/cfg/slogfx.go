package cfg

import (
	"github.com/unng-lab/madfarmer/internal/slogfx"
)

var _ slogfx.Config = (*Default)(nil)

type slogfxConfig struct {
}
