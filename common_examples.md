### HTTP server with Database

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/Graphmasters/safedown"
	"github.com/dgraph-io/badger/v3"
)

func main() {
	// Safedown is initialised with the order "FirstInLastDone" to ensure
	// that the database which will be added first and be closed last. This
	// will allow the HTTP server to keep using the database while gracefully
	// shutting down.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

	// The database is opened with the close method being added to the shutdown
	// actions. The choice of database, i.e. "badger", was arbitrary and 
	// unimportant.
	db, err := badger.Open(badger.DefaultOptions("/path/to/badger/file"))
	if err != nil {
		log.Fatalf("unable to open badger: %v", err)
	}
	sa.AddActions(func() {
		// It is upto the user how the error (if applicable) from closing 
		// should be handled.
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	})

	// A service is created with its shutdown method being added to the 
	// shutdown actions.
	handler := func(writer http.ResponseWriter, request *http.Request) {
		// TODO: Write your handle method which uses the database
	}
	server := &http.Server{
		Addr:    "address",
		Handler: http.HandlerFunc(handler),
	}
	sa.AddActions(func() {
		// The specifics of gracefully shutdown the service, include the timeout
		// and error handling, should be decided by the user.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown server: %v", err)
		}
	})

	// The server starts listening. It should never encounter an error other
	// than the server being closed.
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server encountered error: %v", err)
	}
}

```