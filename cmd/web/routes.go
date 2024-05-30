package main

import (
	"net/http"

	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {

	mux := http.NewServeMux()

	// serve static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// routes
	mux.HandleFunc("GET /{$}", app.indexHandler)
	mux.HandleFunc("GET /tipp/view/{tippID}", app.tippViewHandler)
	mux.HandleFunc("POST /tipp/update", app.tippUpdateMultipleHandler)

	// standard middleware chain
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
