# Safedown

Safedown is for ensuring that an application shuts down 
gracefully even when termination or interruption signals 
are received. 
 
Adding shutdown actions along with a set of signals allows 
for methods (in this case `cancel`) to be run when a 
termination signal, or similar, is received.


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
    
    // The shutdown actions are initialised and will only run
    // if one of the provided signals is received.
    sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

    // The context can be cancelled be either through the 
    // shutdown actions or via the defer.
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

To ensure that the shutdown logic (represented by actions) 
always runs (particularly for running unending applications) 
one can also include the line `defer sa.Shutdown()`. 
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

    // The shutdown actions are initialised and Shutdown is
    // deferred. This ensures that the shutdown actions are
    // always run and that the main is blocked from 
    // finishing until the shutdown process is complete.
    sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
    defer sa.Shutdown()

    // The context can be cancelled be through the shutdown
    // action, triggered either by a signal or sa.Shutdown().
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

One can also manage shutdown actions across goroutines by 
creating the shutdown actions without any signals to be 
listened for.

Additional examples can be found in the examples module
1. [HTTP server with database](examples/http_server_with_database/main.go).

### F.A.Q.

- *How do I ensure that the shutdown actions complete before
the program terminates?* Use either `Shutdown()` or 
`Wait()`. We recommend against using `Wait()` as it is 
possible the shutdown actions will never be triggered and
the program will never stop.
