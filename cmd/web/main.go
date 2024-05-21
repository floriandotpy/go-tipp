package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
)

// application-wide dependencies
type application struct {
	logger *slog.Logger
}

func main() {
	// CLI flags
	addr := flag.String("addr", ":8090", "HTTP network address")
	flag.Parse()

	// logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// setup for dependency injection across our app
	app := &application{
		logger: logger,
	}

	// start server
	logger.Info("Starting server", "addr", *addr)
	err := http.ListenAndServe(":8090", app.routes())

	logger.Error(err.Error())
	os.Exit(1)
}
