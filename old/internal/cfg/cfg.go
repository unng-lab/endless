package cfg

import (
	"log/slog"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

const (
	prefix = "GOAPP_"
	delim  = "."
)

type Default struct {
	log *slog.Logger
	runnerConfig
}

func New(log *slog.Logger) *Default {
	var d Default
	d.log = log.With("cfg", "Default")

	k := koanf.New(delim)
	if err := k.Load(confmap.Provider(map[string]interface{}{
		"ui.test":                     "developer",
		"ebitenfx.width":              800,
		"ebitenfx.height":             800,
		"ebitenfx.window.resize.mode": 2,
		"game.test":                   "developer",
		"scr.test":                    "developer",
		"slogfx.test":                 "developer",
	}, delim), nil); err != nil {
		d.log.Error("k.Load", "err", err)
		return nil
	}

	// Load environment variables and merge into the loaded config.
	// "MYVAR" is the prefix to filter the env vars by.
	// "." is the delimiter used to represent the key hierarchy in env vars.
	// The (optional, or can be nil) function can be used to transform
	// the env var names, for instance, to lowercase them.
	//
	// For example, env vars: MYVAR_TYPE and MYVAR_PARENT1_CHILD1_NAME
	// will be merged into the "type" and the nested "parent1.child1.name"
	// keys in the config file here as we lowercase the key,
	// replace `_` with `.` and strip the MYVAR_ prefix so that
	// only "parent1.child1.name" remains.

	if err := k.Load(env.Provider(prefix, delim, func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, prefix)), "_", delim, -1)
	}), nil); err != nil {
		d.log.Error("k.Load", "err", err)
		return nil
	}

	return &d
}
