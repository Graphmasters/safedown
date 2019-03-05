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
	order         Order
	actions       []func()
	onSignalMutex sync.Mutex
	onSignalFunc  func(os.Signal)
	closeCh       chan struct{}
	closeOnce     sync.Once
	shutdownOnce  sync.Once
}

// NewShutdownActions creates a new set of shutdown actions and starts listen for any of the signals provided.
// The order determines the order the actions will be executed.
func NewShutdownActions(order Order, signals ...os.Signal) *ShutdownActions {
	sa := &ShutdownActions{
		order:         order,
		actions:       make([]func(), 0),
		onSignalMutex: sync.Mutex{},
		onSignalFunc:  nil,
		closeCh:       make(chan struct{}),
		closeOnce:     sync.Once{},
		shutdownOnce:  sync.Once{},
	}

	go sa.start(signals)
	return sa
}

// AddActions adds actions to be run on shutdown
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.actions = append(sa.actions, actions...)
}

// SetOnSignal sets the method which will be called when a signal is received
func (sa *ShutdownActions) SetOnSignal(onSignal func(os.Signal)) {
	sa.onSignalMutex.Lock()
	defer sa.onSignalMutex.Unlock()
	sa.onSignalFunc = onSignal
}

// Shutdown sends close message and shuts down
func (sa *ShutdownActions) Shutdown() {
	sa.closeOnce.Do(func() {
		close(sa.closeCh)
	})
	sa.shutdown()
}

// onSignal passes the signal calls the on signal function
func (sa *ShutdownActions) onSignal(s os.Signal) {
	sa.onSignalMutex.Lock()
	defer sa.onSignalMutex.Unlock()

	if sa.onSignalFunc != nil {
		sa.onSignalFunc(s)
	}
}

// start listens for an interrupt signal and runs
func (sa *ShutdownActions) start(signals []os.Signal) {
	// Notification channel
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, signals...)

	// Listens for signals and close message
	select {
	case s := <-signalCh:
		sa.onSignal(s)
	case <-sa.closeCh:
	}

	// runs shutdown actions
	sa.shutdown()
}

// Runs shutdown actions
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
