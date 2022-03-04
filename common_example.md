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
	// that the database which will be added first will be closed last.
	sa := safedown.NewShutdownActions(safedown.FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)

	// The database is opened with the close method being added to the shutdown
	// actions.
	db, err := badger.Open(badger.DefaultOptions("/path/to/badger/file"))
	if err != nil {
		log.Fatalf("unable to open badger: %v", err)
	}
	sa.AddActions(func() {
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