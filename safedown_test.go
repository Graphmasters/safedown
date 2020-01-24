package safedown

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

// region Examples

func ExampleNewShutdownActions() {
	// Creates the shutdown actions and defers the Shutdown method.
	sa := NewShutdownActions(FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
	defer sa.Shutdown()

	// Sets the function to be called if a signal is received
	sa.SetOnSignal(func(signal os.Signal) {
		fmt.Printf("A signal was received: %s\n", signal.String())
	})

	// The first action added will be the last done
	sa.AddActions(func() {
		fmt.Println("... and this will be done last.")
	})

	// The last action added will be the first done
	sa.AddActions(func() {
		fmt.Println("This will be done first ...")
	})

	// Output:
	// This will be done first ...
	// ... and this will be done last.
}

// endregion

// region Tests

func TestShutdownActions_SetOnSignal(t *testing.T) {
	finished := setTestTimeout(t, 3*time.Second)
	defer finished()

	wg := sync.WaitGroup{}
	sa := NewShutdownActions(FirstInLastDone, os.Interrupt)
	wg.Add(1)
	sa.SetOnSignal(func(signal os.Signal) {
		if signal != os.Interrupt {
			t.Logf("signal received was %s instead of %s", signal.String(), os.Interrupt.String())
			t.Fail()
		}
		wg.Done()
	})

	sendSignalToSelf(t, os.Interrupt)
	wg.Wait()
}

func TestNewShutdownActions_FirstInFirstDone(t *testing.T) {
	finished := setTestTimeout(t, time.Second)
	defer finished()

	count := 0
	wg := sync.WaitGroup{}
	sa := NewShutdownActions(FirstInFirstDone)
	sa.AddActions(counter(t, &wg, 1, &count))
	sa.AddActions(counter(t, &wg, 2, &count))
	sa.AddActions(counter(t, &wg, 3, &count))
	sa.Shutdown()
	wg.Wait()
}

func TestNewShutdownActions_FirstInLastDone(t *testing.T) {
	cancel := setTestTimeout(t, time.Second)
	defer cancel()

	count := 0
	wg := sync.WaitGroup{}
	sa := NewShutdownActions(FirstInLastDone)
	sa.AddActions(counter(t, &wg, 3, &count))
	sa.AddActions(counter(t, &wg, 2, &count))
	sa.AddActions(counter(t, &wg, 1, &count))
	sa.Shutdown()
	wg.Wait()
}

// endregion

// region Helpers

// counter creates a function that should be added to the shutdown actions.
// The test will fail if the value given doesn't increment to the expected value.
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

// setTestTimeout returns a function which should be called at the end of the test.
// If the function is not called before the timeout the test fails.
func setTestTimeout(t *testing.T, timeout time.Duration) func() {
	closeCh := make(chan struct{})

	go func() {
		select {
		case <-closeCh:
		case <-time.After(timeout):
			t.Log("test exceeded timeout")
			t.Fail()
		}
	}()

	return func() {
		close(closeCh)
	}
}

// endregion
