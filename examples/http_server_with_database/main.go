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
	defer sa.Shutdown()

	// The database is opened with the close method being added to the shutdown
	// actions. The choice of database, i.e. "badger", was arbitrary and
	// unimportant.
	db, err := badger.Open(badger.DefaultOptions("/path/to/badger/file"))
	if err != nil {
		log.Fatalf("unable to open badger: %v\n", err)
	}
	sa.AddActions(func() {
		// It is up to the user how the error (if applicable) from closing
		// should be handled. This is only for illustrative purposes.
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v\n", err)
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
		// The specifics of gracefully shutting down the service, include the timeout
		// and error handling, should be decided by the user.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown server: %v\n", err)
		}
	})

	// The server starts listening. The method `ListenAndServe` always returns
	// a non-nil errors which should never be anything other than the server
	// being closed if everything has worked correctly. If another error is
	// returned the user should decide how to handle the error.
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		// It is up to the user how the error should be handled. This is only
		// for illustrative purposes.
		log.Printf("server encountered error: %v", err)
	}
}
