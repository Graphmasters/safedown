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
	sa := &ShutdownActions{}
	Initialise(sa, order, signals...)
	return sa
}

// Initialise initialises a set of shutdown actions and starts listen for any of the signals provided.
// The order determines the order the actions will be executed.
// A panic will occur if the shutdown actions have already been initialised.
//
// It is generally preferable to use the NewShutdownActions method, unless you are concerned about
// the ShutdownActions escaping to the heap.
func Initialise(sa *ShutdownActions, order Order, signals ...os.Signal) {
	// Checks if shutdown actions have already been initialised
	if sa.stopCh != nil {
		panic("shutdown actions cannot be initialised twice")
	}

	// Sets order and creates stop channel
	sa.order = order
	sa.stopCh = make(chan struct{})

	// Checks if there are signals to be listen to
	if len(signals) > 0 {
		go sa.start(signals)
	}
}

// AddActions adds actions to be run on shutdown or when a signal is received.
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.actions = append(sa.actions, actions...)
}

// SetOnSignal sets the method which will be called when a signal is received.
func (sa *ShutdownActions) SetOnSignal(onSignal func(os.Signal)) {
	sa.onSignalMutex.Lock()
	sa.onSignalFunc = onSignal
	sa.onSignalMutex.Unlock()
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
	if sa.onSignalFunc != nil {
		sa.onSignalFunc(s)
	}
	sa.onSignalMutex.Unlock()
}

// start listens for an interrupt signal and runs
func (sa *ShutdownActions) start(signals []os.Signal) {
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
