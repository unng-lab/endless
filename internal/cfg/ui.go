package cfg

import (
	"github/unng-lab/madfarmer/internal/ui"
)

var _ ui.Config = (*Default)(nil)

type uiConfig struct {
}

func (d Default) Test() {
	//TODO implement me
	panic("implement me")
}
