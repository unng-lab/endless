package cfg

import (
	"github/unng-lab/madfarmer/internal/units/runner"
)

var _ runner.Config = (*Default)(nil)

type runnerConfig struct {
}
