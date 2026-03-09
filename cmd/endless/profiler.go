package main

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

type profilingConfig struct {
	cpuProfilePath  string
	heapProfilePath string
	tracePath       string
	pprofAddress    string
}

type profilerSession struct {
	cpuProfileFile  *os.File
	traceFile       *os.File
	heapProfilePath string

	stopOnce sync.Once
	stopErr  error
}

// startProfiler enables the optional profiling endpoints requested on the command line.
// CPU and trace profilers are started before the Ebiten loop begins, while heap, mutex and
// block profiles are configured so they can be captured after the stress run completes.
func startProfiler(config profilingConfig) (*profilerSession, error) {
	session := &profilerSession{
		heapProfilePath: config.heapProfilePath,
	}

	if config.pprofAddress != "" {
		if err := startPprofServer(config.pprofAddress); err != nil {
			return nil, err
		}
	}

	if config.cpuProfilePath != "" {
		cpuProfileFile, err := createProfileFile(config.cpuProfilePath)
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

	if config.tracePath != "" {
		traceFile, err := createProfileFile(config.tracePath)
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

	if config.heapProfilePath != "" {
		runtime.MemProfileRate = 1
		log.Printf("Heap profiling enabled: %s", config.heapProfilePath)
	}

	if config.pprofAddress != "" || config.cpuProfilePath != "" || config.heapProfilePath != "" || config.tracePath != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
		log.Printf("Blocking and mutex profiling enabled")
	}

	return session, nil
}

// Stop flushes every active profile in the right order so the generated files stay valid even
// when the game exits because the window was closed after a long stress session.
func (s *profilerSession) Stop() error {
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

// startPprofServer launches the standard library diagnostics handlers in the background so
// the running Ebiten process can be sampled with `go tool pprof` during a stress run.
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

// writeHeapProfile forces a GC cycle before the snapshot so the resulting heap profile shows
// retained memory instead of transient allocations that were already dead at shutdown.
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

// createProfileFile ensures profile outputs can be written into nested folders directly from
// the run command, which keeps repeatable stress captures convenient.
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
