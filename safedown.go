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

type PostShutdownStrategy uint8

const (
	DoNothing PostShutdownStrategy = iota
	PerformImmediately
	PerformCoordinately
)

// ShutdownActions contains actions that are run when the os receives an interrupt signal.
// This object must be created using the NewShutdownActions function.
type ShutdownActions struct {
	order        Order                // This determines the order the actions will be done.
	actions      []func()             // The actions done on shutdown.
	onSignalFunc func(os.Signal)      // The function to be called when a signal is received.
	strategy     PostShutdownStrategy // The strategy for actions after shutdown has been triggered

	isShutdownTriggered       bool // This is true if the shutdown actions have been triggered
	isProcessingStoredActions bool // This is true only while or immediately before stored actions are being processed.

	stopCh       chan struct{} // A channel to stop listening for signals.
	stopOnce     sync.Once     // Ensures listening to signals is stopped once.
	shutdownCh   chan struct{} // A channel that indicates if shutdown has been completed.
	shutdownOnce sync.Once     // Ensures shutdown actions are only done once.
	mutex        sync.Mutex    // A mutex to avoid clashes handling actions or onSignal.
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

// AddActions adds actions to be run on Shutdown or when a signal is received.
//
// Any action added after shutdown has been triggered will be handled according
// to the post shutdown strategy.
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.mutex.Lock()
	if !sa.isShutdownTriggered {
		sa.actions = append(sa.actions, actions...)
		sa.mutex.Unlock()
		return
	}

	// The decision to perform the actions in the background is a pragmatic one.
	// In the case of PerformCoordinately there would need to be an additional
	// mechanism to record if the actions had been performed which would require
	// significant changes.

	switch sa.strategy {
	case PerformImmediately:
		sa.mutex.Unlock()
		go sa.performActions(actions)
		return
	case PerformCoordinately:
		sa.actions = append(sa.actions, actions...)
		if sa.isProcessingStoredActions {
			sa.mutex.Unlock()
			return
		}

		sa.isProcessingStoredActions = true
		sa.mutex.Unlock()
		go sa.performStoredActions()
	default:
		sa.mutex.Unlock()
		return
	}

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

// UsePostShutdownStrategy sets the strategy for actions added after shutdown
// has been triggered. This strategy is typically only used if the application
// receives a signal during its initialisation.
func (sa *ShutdownActions) UsePostShutdownStrategy(strategy PostShutdownStrategy) {
	sa.mutex.Lock()
	sa.strategy = strategy
	sa.mutex.Unlock()
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

func (sa *ShutdownActions) performActions(actions []func()) {
	for i := range actions {
		if sa.order == FirstInFirstDone {
			actions[i]()
		} else {
			actions[len(actions)-i-1]()
		}
	}
}

func (sa *ShutdownActions) performStoredActions() {
	for {
		var action func()
		sa.mutex.Lock()
		switch {
		case len(sa.actions) == 0:
			sa.isProcessingStoredActions = false
			sa.mutex.Unlock()
			return
		case sa.order == FirstInLastDone:
			action = sa.actions[len(sa.actions)-1]
			sa.actions = sa.actions[:len(sa.actions)-1]
		default:
			action = sa.actions[0]
			sa.actions = sa.actions[1:]
		}
		sa.mutex.Unlock()

		action()
	}
}

// shutdown runs the shutdown actions
func (sa *ShutdownActions) shutdown() {
	sa.shutdownOnce.Do(
		func() {
			sa.mutex.Lock()
			sa.isShutdownTriggered = true
			sa.isProcessingStoredActions = true
			sa.mutex.Unlock()

			sa.performStoredActions()

			// Closes the shutdown channel indicating shutdown
			// is complete.
			close(sa.shutdownCh)
		},
	)
}
