package launcher

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
)

// ProfilingConfig describes every optional runtime profile the desktop launchers may enable
// before Ebiten takes control of the process.
type ProfilingConfig struct {
	CPUProfilePath  string
	HeapProfilePath string
	TracePath       string
	PprofAddress    string
}

// ProfilerSession owns the files and shutdown order for the active profiling tools so callers
// may always defer Stop immediately after startup succeeds.
type ProfilerSession struct {
	cpuProfileFile  *os.File
	traceFile       *os.File
	heapProfilePath string

	stopOnce sync.Once
	stopErr  error
}

// StartProfiler enables the optional profiling endpoints requested on the command line. CPU
// and trace profilers are started before the Ebiten loop begins, while heap, mutex and block
// profiles are configured so they can be captured after a long interactive run finishes.
func StartProfiler(config ProfilingConfig) (*ProfilerSession, error) {
	session := &ProfilerSession{
		heapProfilePath: config.HeapProfilePath,
	}

	if config.PprofAddress != "" {
		if err := startPprofServer(config.PprofAddress); err != nil {
			return nil, err
		}
	}

	if config.CPUProfilePath != "" {
		cpuProfileFile, err := createProfileFile(config.CPUProfilePath)
		if err != nil {
			return nil, fmt.Errorf("create CPU profile file: %w", err)
		}
		if err := pprof.StartCPUProfile(cpuProfileFile); err != nil {
			_ = cpuProfileFile.Close()
			return nil, fmt.Errorf("start CPU profile: %w", err)
		}
		session.cpuProfileFile = cpuProfileFile
		log.Printf("CPU profiling enabled: %s", cpuProfileFile.Name())
	}

	if config.TracePath != "" {
		traceFile, err := createProfileFile(config.TracePath)
		if err != nil {
			pprof.StopCPUProfile()
			if closeErr := closeFiles(session.cpuProfileFile); closeErr != nil {
				log.Printf("close CPU profile file after trace setup failure: %v", closeErr)
			}
			return nil, fmt.Errorf("create trace file: %w", err)
		}
		if err := trace.Start(traceFile); err != nil {
			_ = traceFile.Close()
			pprof.StopCPUProfile()
			if closeErr := closeFiles(session.cpuProfileFile); closeErr != nil {
				log.Printf("close CPU profile file after trace start failure: %v", closeErr)
			}
			return nil, fmt.Errorf("start runtime trace: %w", err)
		}
		session.traceFile = traceFile
		log.Printf("Runtime trace enabled: %s", traceFile.Name())
	}

	if config.HeapProfilePath != "" {
		runtime.MemProfileRate = 1
		log.Printf("Heap profiling enabled: %s", config.HeapProfilePath)
	}

	if config.PprofAddress != "" || config.CPUProfilePath != "" || config.HeapProfilePath != "" || config.TracePath != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
		log.Printf("Blocking and mutex profiling enabled")
	}

	return session, nil
}

// Stop flushes every active profile in the correct order so the produced files remain valid
// even when the game exits because the user closes the window during profiling.
func (s *ProfilerSession) Stop() error {
	if s == nil {
		return nil
	}

	s.stopOnce.Do(func() {
		var stopErrors []error

		if s.traceFile != nil {
			trace.Stop()
			if err := s.traceFile.Close(); err != nil {
				stopErrors = append(stopErrors, fmt.Errorf("close trace file: %w", err))
			}
		}

		if s.cpuProfileFile != nil {
			pprof.StopCPUProfile()
			if err := s.cpuProfileFile.Close(); err != nil {
				stopErrors = append(stopErrors, fmt.Errorf("close CPU profile file: %w", err))
			}
		}

		if s.heapProfilePath != "" {
			if err := writeHeapProfile(s.heapProfilePath); err != nil {
				stopErrors = append(stopErrors, err)
			}
		}

		s.stopErr = errors.Join(stopErrors...)
	})

	return s.stopErr
}

// startPprofServer launches the standard-library diagnostics handlers in the background so the
// running Ebiten process can be sampled with go tool pprof while the window stays interactive.
func startPprofServer(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen pprof server on %s: %w", address, err)
	}

	go func() {
		log.Printf("pprof HTTP server listening on http://%s/debug/pprof/", address)
		if err := http.Serve(listener, nil); err != nil {
			log.Printf("pprof HTTP server stopped: %v", err)
		}
	}()

	return nil
}

// writeHeapProfile forces a GC cycle before taking the snapshot so the saved heap profile
// reflects retained allocations instead of garbage that was already unreachable.
func writeHeapProfile(profilePath string) error {
	heapProfileFile, err := createProfileFile(profilePath)
	if err != nil {
		return fmt.Errorf("create heap profile file: %w", err)
	}
	defer func() {
		if closeErr := heapProfileFile.Close(); closeErr != nil {
			log.Printf("close heap profile file: %v", closeErr)
		}
	}()

	runtime.GC()
	if err := pprof.WriteHeapProfile(heapProfileFile); err != nil {
		return fmt.Errorf("write heap profile: %w", err)
	}

	log.Printf("Heap profile written: %s", heapProfileFile.Name())
	return nil
}

// createProfileFile ensures profiling outputs can be written into nested directories directly
// from the launch command without any manual filesystem preparation.
func createProfileFile(profilePath string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return nil, fmt.Errorf("create profile directory: %w", err)
	}

	file, err := os.Create(profilePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// closeFiles collapses repeated best-effort cleanup paths into one helper so the startup
// rollback logic stays readable when one of the profilers fails to initialize.
func closeFiles(files ...*os.File) error {
	var closeErrors []error
	for _, file := range files {
		if file == nil {
			continue
		}
		if err := file.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}

	return errors.Join(closeErrors...)
}
