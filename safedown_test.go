package safedown_test

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Graphmasters/safedown"
)

// TestShutdownActions_OnSignal tests if the correct signal is received.
// It also indirectly checks that the shutdown actions are intercepting
// signals as the test will simply fail otherwise.
// nolint: gomnd
func TestShutdownActions_OnSignal(t *testing.T) {
	// Makes error channel and checks that no errors were passed to it.
	errs := make(chan error, 1)
	defer close(errs)
	defer checkErrors(t, errs)

	// Wait Group for ensuring that every actions that is
	// supposed to happen does.
	wg := sync.WaitGroup{}
	defer addWaitGroupDeadline(t, &wg, time.Now().Add(3*time.Second))

	wg.Add(1)
	expected := os.Interrupt
	actionsA := safedown.NewShutdownActions(safedown.FirstInLastDone, expected)
	actionsA.SetOnSignal(func(received os.Signal) {
		if received != expected {
			errs <- fmt.Errorf("signal received was %s instead of %s", received.String(), expected.String())
		}
		wg.Done()
	})

	// Sends the expected signal
	sendSignalToSelf(t, expected)
}

// TestShutdownActions_OnSignal tests if the correct signal is received.
// It also indirectly checks that the shutdown actions are intercepting
// signals as the test will simply fail otherwise.
// nolint: gomnd
func TestShutdownActions_OnSignal_Multiple(t *testing.T) {
	// Makes error channel and checks that no errors were sent to it.
	errs := make(chan error, 3)
	defer close(errs)
	defer checkErrors(t, errs)

	// Wait Group for ensuring that every actions that is
	// supposed to happen does.
	wg := sync.WaitGroup{}
	defer addWaitGroupDeadline(t, &wg, time.Now().Add(3*time.Second))

	// Lists the expect & unexpected signals as well as
	// the number of
	expected := os.Interrupt
	unexpected := os.Kill
	wg.Add(2)

	// Shutdown actions only listen for the expected signal.
	// The OnSignal method should be triggered.
	actionsA := safedown.NewShutdownActions(safedown.FirstInLastDone, expected)
	actionsA.SetOnSignal(func(received os.Signal) {
		if received != expected {
			errs <- fmt.Errorf("signal received was %s instead of %s", received.String(), expected.String())
		}
		wg.Done()
	})

	// Shutdown actions listens for the expected and
	// unexpected signal. The OnSignal method should be
	// triggered.
	actionsB := safedown.NewShutdownActions(safedown.FirstInLastDone, expected, unexpected)
	actionsB.SetOnSignal(func(received os.Signal) {
		if received != expected {
			errs <- fmt.Errorf("signal received was %s instead of %s", received.String(), expected.String())
		}
		wg.Done()
	})

	// Shutdown actions only listen for the unexpected signal.
	// The OnSignal method should not be triggered.
	actionsC := safedown.NewShutdownActions(safedown.FirstInLastDone, unexpected)
	actionsC.SetOnSignal(func(received os.Signal) {
		errs <- fmt.Errorf("an unexpected signal (%s) was received", received.String())
	})

	// Sends the expected signal and sleeps to give it a
	// chance to be received even when it is unexpected.
	sendSignalToSelf(t, expected)
	time.Sleep(time.Second)
}

// TestNewShutdownActions_FirstInFirstDone checks that
// actions are down in the order they were added.
// nolint: gomnd
func TestNewShutdownActions_FirstInFirstDone(t *testing.T) {
	wg := sync.WaitGroup{}
	defer addWaitGroupDeadline(t, &wg, time.Now().Add(time.Second))

	count := 0
	sa := safedown.NewShutdownActions(safedown.FirstInFirstDone)
	sa.AddActions(counter(t, &wg, 1, &count))
	sa.AddActions(counter(t, &wg, 2, &count))
	sa.AddActions(counter(t, &wg, 3, &count))
	sa.Shutdown()
}

// TestNewShutdownActions_FirstInLastDone checks that
// actions are down in the opposite order to the order they
// were added.
// nolint: gomnd
func TestNewShutdownActions_FirstInLastDone(t *testing.T) {
	wg := sync.WaitGroup{}
	defer addWaitGroupDeadline(t, &wg, time.Now().Add(time.Second))

	count := 0
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone)
	sa.AddActions(counter(t, &wg, 3, &count))
	sa.AddActions(counter(t, &wg, 2, &count))
	sa.AddActions(counter(t, &wg, 1, &count))
	sa.Shutdown()
}

// addWaitGroupDeadline adds waits till either the wait group
// is done or until the deadline is reached. If the deadline
// is reached then test fails.
func addWaitGroupDeadline(t *testing.T, wg *sync.WaitGroup, deadline time.Time) {
	success := make(chan struct{})
	go func() {
		wg.Wait()
		close(success)
	}()

	select {
	case <-success:
	case <-time.After(time.Until(deadline)):
		t.Fatal("test exceeded timeout")
	}
}

// checkErrors checks that the error channel has received no
// errors.
func checkErrors(t *testing.T, errs chan error) {
	for {
		select {
		case err, ok := <-errs:
			if !ok {
				return
			}

			t.Fatal(err)
		default:
			return
		}
	}
}

// counter creates a function that should be added to the shutdown actions.
// The test will fail if the value given doesn't increment to the expected value.
// nolint: gomnd
func counter(t *testing.T, wg *sync.WaitGroup, expected int, value *int) func() {
	wg.Add(1)
	return func() {
		*value++
		if *value != expected {
			t.Fail()
		}
		wg.Done()
	}
}

// sendSignalToSelf sends the signal provided to the current process.
func sendSignalToSelf(t *testing.T, signal os.Signal) {
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Log("could not find process")
		t.Fail()
		return
	}

	err = process.Signal(signal)
	if err != nil {
		t.Log("could not send signal to process")
		t.Fail()
	}
}
