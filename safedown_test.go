package safedown

import (
	"fmt"
	"syscall"
)

func ExampleNewShutdownActions() {
	sa := NewShutdownActions(FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
	defer sa.Shutdown()

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
