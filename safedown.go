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
	once    *sync.Once
	order   Order
	actions []func()
	closeCh chan struct{}
}

// NewShutdownActions creates a new set of shutdown actions and starts listen for any of the signals provided.
// The order determines the order the actions will be executed.
func NewShutdownActions(order Order, signals ...os.Signal) *ShutdownActions {
	sa := &ShutdownActions{
		once:    &sync.Once{},
		order:   order,
		actions: make([]func(), 0),
		closeCh: make(chan struct{}),
	}

	go sa.start(signals)
	return sa
}

// AddActions adds actions to be run on shutdown
func (sa *ShutdownActions) AddActions(actions ...func()) {
	sa.actions = append(sa.actions, actions...)
}

// Shutdown sends close message and shuts down
func (sa *ShutdownActions) Shutdown() {
	select {
	case <-sa.closeCh:
	default:
		close(sa.closeCh)
	}
	sa.shutdown()
}

// start listens for an interrupt signal and runs
func (sa *ShutdownActions) start(signals []os.Signal) {
	// Notification channel
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, signals...)

	// Listens for signals and close message
	select {
	case <-signalCh:
	case <-sa.closeCh:
	}

	// runs shutdown actions
	sa.shutdown()
}

// Runs shutdown actions
func (sa *ShutdownActions) shutdown() {
	sa.once.Do(
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
