// Package safedown provides a graceful way to shutdown an application even when an interrupt signal is received.
package safedown

import (
	"os"
	"os/signal"
	"sync"
)

type Order bool

const (
	FirstInFirstDone Order = true  // Actions are executed in the order they are added.
	FirstInLastDone  Order = false // Actions are executed in the reversed order they are added.
)

// ShutdownActions is a set of actions that are run when the os receives an Interrupt signal.
type ShutdownActions struct {
	order         Order           // This determines the order the actions will be done.
	actions       []func()        // The actions done on shutdown.
	onSignalMutex sync.Mutex      // A mutex for handling the onSignalFunc.
	onSignalFunc  func(os.Signal) // The function to be called when a signal is received.
	stopCh        chan struct{}   // A channel to stop listening for signals.
	stopOnce      sync.Once       // Ensures listening to signals is stopped once.
	shutdownOnce  sync.Once       // Ensures shutdown actions are only done once.
}

// NewShutdownActions creates a new set of shutdown actions and starts listen for any of the signals provided.
// The order determines the order the actions will be executed.
func NewShutdownActions(order Order, signals ...os.Signal) *ShutdownActions {
	sa := &ShutdownActions{
		order:         order,
		actions:       make([]func(), 0),
		onSignalMutex: sync.Mutex{},
		onSignalFunc:  nil,
		stopCh:        make(chan struct{}),
		stopOnce:      sync.Once{},
		shutdownOnce:  sync.Once{},
	}

	go sa.start(signals)
	return sa
}

// AddActions adds actions to be run on shutdown or when a signal is received.
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.actions = append(sa.actions, actions...)
}

// SetOnSignal sets the method which will be called when a signal is received.
func (sa *ShutdownActions) SetOnSignal(onSignal func(os.Signal)) {
	sa.onSignalMutex.Lock()
	defer sa.onSignalMutex.Unlock()
	sa.onSignalFunc = onSignal
}

// Shutdown runs the shutdown actions and stops listening for signals.
func (sa *ShutdownActions) Shutdown() {
	sa.stopOnce.Do(func() {
		close(sa.stopCh)
	})
	sa.shutdown()
}

// onSignal passes the signal received to the on signal function
func (sa *ShutdownActions) onSignal(s os.Signal) {
	sa.onSignalMutex.Lock()
	defer sa.onSignalMutex.Unlock()

	if sa.onSignalFunc != nil {
		sa.onSignalFunc(s)
	}
}

// start listens for an interrupt signal and runs
func (sa *ShutdownActions) start(signals []os.Signal) {
	// Checks if there are signals to listen for
	if len(signals) == 0 {
		return
	}

	// Notification channel
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, signals...)

	// Listens for signals and close message
	select {
	case s := <-signalCh:
		sa.onSignal(s)
	case <-sa.stopCh:
	}

	// Stops listening for signals
	signal.Stop(signalCh)
	close(signalCh)

	// runs shutdown actions
	sa.shutdown()
}

// shutdown runs the shutdown actions
func (sa *ShutdownActions) shutdown() {
	sa.shutdownOnce.Do(
		func() {
			l := len(sa.actions)
			for i := 0; i < l; i++ {
				if sa.order == FirstInFirstDone {
					sa.actions[i]()
				} else {
					sa.actions[l-i-1]()
				}
			}
		},
	)
}
