package safedown_test

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/Graphmasters/safedown"
)

func TestMain(m *testing.M) {
	go func() {
		time.Sleep(2 * time.Second)
		process := os.Process{Pid: os.Getpid()}
		errSignal := process.Signal(syscall.SIGTERM)
		if errSignal != nil {
			fmt.Printf("error sending signal: %s", errSignal)
		}
	}()

	os.Exit(m.Run())
}

func ExampleNewShutdownActions_withContext() {
	defer fmt.Println("Finished")

	// The shutdown actions are initialised and will only run
	// if one of the provided signals is received.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

	// The context can be cancelled be either through the
	// shutdown actions or via the defer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sa.AddActions(cancel)

	fmt.Println("Processing is starting")
	t := time.After(10 * time.Second)
	select {
	case <-ctx.Done():
		fmt.Println("context done")
	case <-t:
		fmt.Println("10 seconds reached")
	}

	// Output:
	// Processing is starting
	// context done
	// Finished
}
