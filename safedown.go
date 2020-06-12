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
// This object must be created using the NewShutdownActions function.
type ShutdownActions struct {
	order        Order           // This determines the order the actions will be done.
	actions      []func()        // The actions done on shutdown.
	onSignalFunc func(os.Signal) // The function to be called when a signal is received.
	stopCh       chan struct{}   // A channel to stop listening for signals.
	stopOnce     sync.Once       // Ensures listening to signals is stopped once.
	shutdownCh   chan struct{}   // A channel that indicates if shutdown has been completed.
	shutdownOnce sync.Once       // Ensures shutdown actions are only done once.
	mutex        sync.Mutex      // A mutex to avoid clashes handling actions or onSignal.
}

// NewShutdownActions creates and initialises a new set of shutdown actions.
// The actions (added later) will be executed if any of the signals provided are received.
// The order determines the order the actions will be executed.
func NewShutdownActions(order Order, signals ...os.Signal) *ShutdownActions {
	// Creates struct with order and stop channel
	sa := &ShutdownActions{
		order:      order,
		stopCh:     make(chan struct{}),
		shutdownCh: make(chan struct{}),
	}

	// If there are no signals to listen to then the stop channel is not required and initialisation is complete
	if len(signals) == 0 {
		sa.closeStopCh()
		return sa
	}

	// Creates channel to receive notification signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, signals...)

	// Starts a go routine for listening for signals and close messages
	go func() {
		// Listens for signal or close message
		var received os.Signal
		select {
		case <-sa.stopCh:
		case received = <-signalCh:
		}

		// Stops listening for signals and closes channels
		signal.Stop(signalCh)
		close(signalCh)
		sa.closeStopCh()

		// Runs on signal and shutdown actions
		sa.onSignal(received)
		sa.shutdown()
	}()

	return sa
}

// AddActions adds actions to be run on shutdown or when a
// signal is received. Any action added after a signal has
// been received or the Shutdown method has been called will
// not be executed.
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.mutex.Lock()
	sa.actions = append(sa.actions, actions...)
	sa.mutex.Unlock()
}

// SetOnSignal sets the method which will be called if a signal is received.
func (sa *ShutdownActions) SetOnSignal(onSignal func(os.Signal)) {
	sa.mutex.Lock()
	sa.onSignalFunc = onSignal
	sa.mutex.Unlock()
}

// Shutdown runs the shutdown actions and stops listening
// for signals (if doing so). This method blocks until all
// shutdown actions have been run, regardless of if they
// have been triggered by receiving a signal or calling this
// method.
func (sa *ShutdownActions) Shutdown() {
	sa.closeStopCh()
	sa.shutdown()
}

// Wait waits until all the shutdown actions have been
// called.
func (sa *ShutdownActions) Wait() {
	<-sa.shutdownCh
}

// closeStopCh closes the stop channel
func (sa *ShutdownActions) closeStopCh() {
	sa.stopOnce.Do(func() {
		close(sa.stopCh)
	})
}

// onSignal passes the signal received to the on signal function
func (sa *ShutdownActions) onSignal(s os.Signal) {
	// Ensures signal is not nil
	if s == nil {
		return
	}

	// Gets the on signal function and checks that it is not nil
	var onSignal func(os.Signal)
	sa.mutex.Lock()
	onSignal = sa.onSignalFunc
	sa.mutex.Unlock()
	if onSignal == nil {
		return
	}

	// Calls the on signal function
	onSignal(s)
}

// shutdown runs the shutdown actions
func (sa *ShutdownActions) shutdown() {
	sa.shutdownOnce.Do(
		func() {
			// Gets current length of actions
			sa.mutex.Lock()
			l := len(sa.actions)
			sa.mutex.Unlock()

			// Executes actions in order
			for i := 0; i < l; i++ {
				if sa.order == FirstInFirstDone {
					sa.actions[i]()
				} else {
					sa.actions[l-i-1]()
				}
			}

			// Closes the shutdown channel indicating shutdown
			// is complete.
			close(sa.shutdownCh)
		},
	)
}
