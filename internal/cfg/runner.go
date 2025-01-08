package cfg

import (
	"github.com/unng-lab/madfarmer/internal/units/runner"
)

var _ runner.Config = (*Default)(nil)

type runnerConfig struct {
}
