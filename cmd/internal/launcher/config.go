package launcher

import "flag"

// RunConfig groups every command-line toggle shared by the desktop launchers so each cmd can
// choose its own game scenario while still exposing the same profiling surface.
type RunConfig struct {
	Profiling ProfilingConfig
}

// ParseRunConfig binds and parses the shared profiling flags exactly once for the current
// process. Keeping the flag registration in one package prevents the regular and stress
// launchers from drifting apart over time.
func ParseRunConfig() RunConfig {
	config := RunConfig{}
	flag.StringVar(&config.Profiling.CPUProfilePath, "cpuprofile", "", "write CPU profile to file")
	flag.StringVar(&config.Profiling.HeapProfilePath, "memprofile", "", "write heap profile to file on shutdown")
	flag.StringVar(&config.Profiling.TracePath, "traceprofile", "", "write runtime trace to file")
	flag.StringVar(&config.Profiling.PprofAddress, "pprof", "", "serve net/http/pprof on address, for example 127.0.0.1:6060")
	flag.Parse()
	return config
}
