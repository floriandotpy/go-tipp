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

	//
	mux := http.NewServeMux()

	// serve static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// routes
	mux.HandleFunc("GET /{$}", app.gamesHandler)
	mux.HandleFunc("GET /tipp/view/{tippID}", app.tippViewHandler)
	mux.HandleFunc("GET /tipp/create", app.tippCreateFormHandler)
	mux.HandleFunc("POST /tipp/create", app.tippCreatePostHandler)

	//
	logger.Info("Starting server", "addr", *addr)
	err := http.ListenAndServe(":8090", mux)

	logger.Error(err.Error())
	os.Exit(1)
}
