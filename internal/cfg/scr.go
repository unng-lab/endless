package cfg

import (
	"github/unng-lab/madfarmer/internal/scr"
)

var _ scr.Config = (*Default)(nil)

type scrConfig struct {
}
