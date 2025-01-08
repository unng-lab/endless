package cfg

import (
	"github.com/unng-lab/madfarmer/internal/scr"
)

var _ scr.Config = (*Default)(nil)

type scrConfig struct {
}
