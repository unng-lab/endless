package cfg

import (
	"github.com/unng-lab/endless/internal/units/runner"
)

var _ runner.Config = (*Default)(nil)

type runnerConfig struct {
}
