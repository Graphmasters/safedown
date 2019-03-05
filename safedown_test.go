package safedown

import (
	"fmt"
	"os"
	"syscall"
)

func ExampleNewShutdownActions() {
	sa := NewShutdownActions(FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
	defer sa.Shutdown()
	sa.SetOnSignal(func(signal os.Signal) {
		fmt.Printf("A signal was received: %s\n", signal.String())
	})

	sa.AddActions(func() {
		fmt.Println("... and this will be done last.")
	})

	sa.AddActions(func() {
		fmt.Println("This will be done first ...")
	})

	// Output:
	// This will be done first ...
	// ... and this will be done last.
}
