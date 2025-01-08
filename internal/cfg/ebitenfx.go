package cfg

import (
	"github.com/unng-lab/madfarmer/internal/ebitenfx"
)

var _ ebitenfx.Config = (*Default)(nil)

type ebitenfxConfig struct {
	screenWidth      uint64 `koanf:"width"`
	screenWidth1     uint64 `koanf:"screen.width"`
	screenWidth2     uint64 `koanf:"screen.width"`
	screenHeight     uint64 `koanf:"height"`
	windowResizeMode uint64 `koanf:"screen.window.resize.mode"`
}
