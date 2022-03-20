package safedown_test

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/Graphmasters/safedown"
)

// Example_withSignalReceived demonstrates how setting up the safedown's
// shutdown actions works when a signal is received.
func Example_withSignalReceived() {
	// This code sends a termination signal being sent. This is here purely
	// to demonstrate functionality and should not be included in any production
	// code.
	go func(pid int) {
		time.Sleep(time.Second)
		process := os.Process{Pid: pid}
		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("error sending signal: %s", err)
		}
	}(os.Getpid())

	defer fmt.Println("Finished")

	// The shutdown actions are initialised and will only run
	// if one of the provided signals is received.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

	// Sets the function to be called if a signal is received
	sa.SetOnSignal(func(signal os.Signal) {
		fmt.Printf("Signal received: %s\n", signal.String())
	})

	// The context can be cancelled be either through the
	// shutdown actions or via the defer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sa.AddActions(cancel)

	fmt.Println("Processing starting")
	t := time.After(2 * time.Second)
	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled")
	case <-t:
		fmt.Println("Ticker ticked")
	}

	// Output:
	// Processing starting
	// Signal received: terminated
	// Context cancelled
	// Finished
}

// Example_withoutSignalReceived demonstrates how setting up the safedown's
// shutdown actions works when no signal is received (and the program can
// terminate of its own accord).
func Example_withoutSignalReceived() {
	defer fmt.Println("Finished")

	// The shutdown actions are initialised and will only run
	// if one of the provided signals is received.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

	// Sets the function to be called if a signal is received
	sa.SetOnSignal(func(signal os.Signal) {
		fmt.Printf("Signal received: %s\n", signal.String())
	})

	// The context can be cancelled be either through the
	// shutdown actions or via the defer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sa.AddActions(cancel)

	fmt.Println("Processing starting")
	t := time.After(2 * time.Second)
	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled")
	case <-t:
		fmt.Println("Ticker ticked")
	}

	// Output:
	// Processing starting
	// Ticker ticked
	// Finished
}

// Example_withShutDown demonstrates how to use the Shutdown method can be used.
func Example_withShutDown() {
	// Creates the shutdown actions and defers the Shutdown method.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone)
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
