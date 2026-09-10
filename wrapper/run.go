package w

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/oklog/run"

	"github.com/sven-victor/ez-utils/signals"
)

// Group is a collection of actors that run concurrently and interrupt together.
type Group run.Group

// ErrStopping is returned by Group.Run when a termination signal is received.
var ErrStopping = errors.New("program stopping")

// NopInterrupt is an interrupt function that does nothing.
var NopInterrupt = func(err error) {}

// Add registers execute to run as part of the group. interrupt is called when the group begins shutting down.
func (g *Group) Add(execute func() error, interrupt func(error)) {
	stopCh := signals.SignalHandler()
	stopCh.AddFor(1, 1)
	(*run.Group)(g).Add(execute, func(err error) {
		if interrupt == nil {
			interrupt = NopInterrupt
		}
		interrupt(err)
		stopCh.DoneFor(1)
	})
}

// ExitFunc is the function used to terminate the process after a shutdown timeout. It defaults to os.Exit.
var ExitFunc = os.Exit

// Run starts all registered actors and blocks until they have all finished.
func (g *Group) Run() error {
	stopCh := signals.SignalHandler()
	(*run.Group)(g).Add(func() error {
		<-stopCh.Channel()
		return ErrStopping
	}, func(err error) {
		fmt.Println(">>>", err)
		stopCh.SafeStop(time.Second*30, ExitFunc)
	})
	return (*run.Group)(g).Run()
}
