// Package signals provides a process-wide handler for graceful shutdown signals.
package signals

import (
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type stopFunc func()

type stopFuncs []stopFunc

// Handler coordinates graceful shutdown: it waits for in-flight work, runs pre-stop
// hooks, and closes a stop channel when a termination signal is received.
type Handler struct {
	stopCh       chan struct{}
	wg           sync.Map
	preStopFuncs []stopFuncs
	mux          sync.Mutex
	logger       log.Logger
}

var once = sync.Once{}

const (
	// LevelRoot is the lowest wait-group level, used for process-wide work.
	LevelRoot uint8 = 0
	// LevelTrace is a wait-group level for trace-scoped work.
	LevelTrace uint8 = 5
	// LevelDB is a wait-group level for database-scoped work.
	LevelDB uint8 = 10
	// LevelRequest is a wait-group level for in-flight requests.
	LevelRequest uint8 = 20
	// LevelMax is the highest valid wait-group level.
	LevelMax uint8 = 31
)

// WaitRequest waits until the request-level wait group is idle.
func (s *Handler) WaitRequest() {
	s.getWaitGroup(LevelRequest).Wait()
}

// DoneRequest marks one request-level unit of work as finished.
func (s *Handler) DoneRequest() {
	s.getWaitGroup(LevelRequest).Done()
}

// AddRequest adds delta to the request-level wait group.
func (s *Handler) AddRequest(delta int) {
	s.getWaitGroup(LevelRequest).Add(delta)
}

// Wait waits until the root-level wait group is idle.
func (s *Handler) Wait() {
	s.getWaitGroup(LevelRoot).Wait()
}

// Done marks one root-level unit of work as finished.
func (s *Handler) Done() {
	s.getWaitGroup(LevelRoot).Done()
}

// Add adds delta to the root-level wait group.
func (s *Handler) Add(delta int) {
	s.getWaitGroup(LevelRoot).Add(delta)
}

func (s *Handler) getWaitGroup(level uint8) *sync.WaitGroup {
	wg, _ := s.wg.LoadOrStore(level, &sync.WaitGroup{})
	return wg.(*sync.WaitGroup)
}

// WaitFor waits until the wait group at level is idle.
func (s *Handler) WaitFor(level uint8) {
	s.getWaitGroup(level).Wait()
}

// DoneFor marks one unit of work at level as finished.
func (s *Handler) DoneFor(level uint8) {
	s.getWaitGroup(level).Done()
}

// AddFor adds delta to the wait group at level.
func (s *Handler) AddFor(level uint8, delta int) {
	s.getWaitGroup(level).Add(delta)
}

// Channel returns a channel that is closed when shutdown begins.
func (s *Handler) Channel() <-chan struct{} {
	return s.stopCh
}

// PreStop registers f to run at level during shutdown, before waiting for that level.
func (s *Handler) PreStop(level uint8, f stopFunc) {
	if level > LevelMax || level < LevelRoot {
		panic(ErrorLevelOutOfBounds)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.preStopFuncs[level] = append(s.preStopFuncs[level], f)
}

func (s *Handler) safeStop(logger log.Logger, timeout time.Duration, exitFunc func(int)) {
	go func() {
		timer := time.NewTimer(timeout)
		<-timer.C
		exitFunc(0)
	}()
	stopHandler.mux.Lock()
	defer stopHandler.mux.Unlock()
	level.Info(logger).Log("msg", "Received stop signal, the process is about to stop.")
	close(stopHandler.stopCh)
	for lvl := len(stopHandler.preStopFuncs) - 1; lvl >= 0; lvl-- {
		funcs := stopHandler.preStopFuncs[lvl]
		var wg sync.WaitGroup
		wg.Add(len(funcs))
		for _, f := range funcs {
			go func(sf stopFunc) {
				defer wg.Done()
				sf()
			}(f)
		}
		wg.Wait()
		stopHandler.WaitFor(uint8(lvl))
	}
	exitFunc(0)
}

// SafeStop runs pre-stop hooks and wait groups, then calls exitFunc. If shutdown
// takes longer than timeout, exitFunc is invoked immediately.
func (s *Handler) SafeStop(timeout time.Duration, exitFunc func(int)) {
	s.safeStop(s.logger, timeout, exitFunc)
}

var stopHandler *Handler

// SetupSignalHandler installs a process-wide Handler that shuts down on termination signals.
// It must be called once; later calls return the same Handler.
func SetupSignalHandler(logger log.Logger) (stopCh *Handler) {
	once.Do(func() {
		onlyOneSignalHandler := make(chan struct{})
		close(onlyOneSignalHandler) // panics when called twice
		stopHandler = &Handler{
			stopCh:       make(chan struct{}),
			preStopFuncs: make([]stopFuncs, LevelMax+1),
			logger:       logger,
		}
		c := make(chan os.Signal, 2)
		signal.Notify(c, shutdownSignals...)

		go func() {
			sig := <-c
			stopHandler.safeStop(log.With(logger, "signal", sig), time.Second*30, os.Exit)
			// second signal. Exit directly.
		}()
	})
	return stopHandler
}

var (
	// ErrorNoInit is the panic value from SignalHandler when SetupSignalHandler has not been called.
	ErrorNoInit = errors.New("stopChan is not init")
	// ErrorLevelOutOfBounds is the panic value from PreStop when level is outside [LevelRoot, LevelMax].
	ErrorLevelOutOfBounds = errors.New("level out of bounds")
)

// SignalHandler returns the process-wide Handler installed by SetupSignalHandler.
func SignalHandler() (stopCh *Handler) {
	if stopHandler == nil {
		panic(ErrorNoInit)
	}
	return stopHandler
}
