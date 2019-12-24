// Package safedown provides a graceful way to shutdown an application even when an interrupt signal is received.
package safedown

import (
	"os"
	"os/signal"
	"sync"
)

// Order represents the order that the shutdown actions will be executed.
type Order bool

const (
	FirstInFirstDone Order = true  // Actions are executed in the order they are added.
	FirstInLastDone  Order = false // Actions are executed in the reversed order they are added.
)

// ShutdownActions contains actions that are run when the os receives an interrupt signal.
// It is highly recommended that this struct is created using the NewShutdownActions function.
type ShutdownActions struct {
	order         Order           // This determines the order the actions will be done.
	actions       []func()        // The actions done on shutdown.
	onSignalMutex sync.Mutex      // A mutex for handling the onSignalFunc.
	onSignalFunc  func(os.Signal) // The function to be called when a signal is received.
	stopCh        chan struct{}   // A channel to stop listening for signals (is nil if actions are not initialised).
	stopOnce      sync.Once       // Ensures listening to signals is stopped once.
	shutdownOnce  sync.Once       // Ensures shutdown actions are only done once.
}

// NewShutdownActions creates and initialises a new set of shutdown actions.
// The actions (added later) will be executed if any of the signals provided are received.
// The order determines the order the actions will be executed.
func NewShutdownActions(order Order, signals ...os.Signal) *ShutdownActions {
	sa := &ShutdownActions{}
	Initialise(sa, order, signals...)
	return sa
}

// Initialise initialises a set of shutdown actions and starts listen for any of the signals provided.
// The order determines the order the actions will be executed.
// A panic will occur if the shutdown actions have already been initialised.
//
// It is highly recommend that the NewShutdownActions function is used unless one is explicitly concerned about
// the ShutdownActions escaping to the heap.
func Initialise(sa *ShutdownActions, order Order, signals ...os.Signal) {
	// Checks if shutdown actions have already been initialised
	if sa.stopCh != nil {
		panic("shutdown actions cannot be initialised twice")
	}

	// Sets order and creates stop channel
	sa.order = order
	sa.stopCh = make(chan struct{})

	// If there are no signals to listen then the stop channel is not required
	if len(signals) == 0 {
		sa.closeStopCh()
		return
	}

	// Starts listen to signal
	sa.start(signals)
}

// AddActions adds actions to be run on shutdown or when a signal is received.
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.actions = append(sa.actions, actions...)
}

// SetOnSignal sets the method which will be called if a signal is received.
func (sa *ShutdownActions) SetOnSignal(onSignal func(os.Signal)) {
	sa.onSignalMutex.Lock()
	sa.onSignalFunc = onSignal
	sa.onSignalMutex.Unlock()
}

// Shutdown runs the shutdown actions and stops listening for signals.
func (sa *ShutdownActions) Shutdown() {
	// Checks if the the actions have been initialised
	if sa.stopCh == nil {
		panic("can not call shutdown when actions have not been initialised")
	}

	// Closes stop channel and runs shut down
	sa.closeStopCh()
	sa.shutdown()
}

// closeStopCh closes the stop channel
func (sa *ShutdownActions) closeStopCh() {
	sa.stopOnce.Do(func() {
		close(sa.stopCh)
	})
}

// onSignal passes the signal received to the on signal function
func (sa *ShutdownActions) onSignal(sig os.Signal) {
	sa.onSignalMutex.Lock()
	if sig != nil && sa.onSignalFunc != nil {
		sa.onSignalFunc(sig)
	}
	sa.onSignalMutex.Unlock()
}

// start listens for an interrupt signal and runs
func (sa *ShutdownActions) start(signals []os.Signal) {
	// Notification channel
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, signals...)

	// Starts a go routine for listening for signals and close messages
	go func() {
		// Listens for signals and close message
		var received os.Signal
		select {
		case received = <-signalCh:
		case <-sa.stopCh:
		}

		// Stops listening for signals and closes channels
		signal.Stop(signalCh)
		close(signalCh)
		sa.closeStopCh()

		// runs shutdown actions
		sa.onSignal(received)
		sa.shutdown()
	}()
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
