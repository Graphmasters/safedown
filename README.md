# Safedown

Safedown is for ensuring that an application shuts down gracefully even when termination or interruption signals are sent. 
 
Adding shutdown actions along with a set of signals allows for methods (in this case `cancel`) to be run when a termination or similar signal is received.
```go
package main

import (
	"context"
	"syscall"
	"time"
	
	"github.com/Graphmasters/safedown"
)

func main() {
    defer println("Finished")
    
    sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    sa.AddActions(cancel)

    println("Processing is starting")
    t := time.After(10 * time.Second)   
    select {
    case <-ctx.Done():
    case <-t:
    }
}
```

For the sake of brevity one can also include the line `defer sa.Shutdown()`. 
This ensures that the shutdown logic (represented by actions) is the same regardless of whether a shutdown signal is received or not.
```go
package main

import (
	"context"
	"syscall"
	"time"
	
	"github.com/Graphmasters/safedown"
)

func main() {
    defer println("Finished")

    sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
    defer sa.Shutdown() // Adding this means `defer cancel()` is no longer needed.

    ctx, cancel := context.WithCancel(context.Background())
    sa.AddActions(cancel)

    println("Processing is starting")
    t := time.After(10 * time.Second)   
    select {
    case <-ctx.Done():
    case <-t:
    }
}
```

One can also manage shutdown actions across goroutines by creating the shutdown actions without any signals to be listened for.