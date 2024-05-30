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

	dynamic := alice.New(app.sessionManager.LoadAndSave)

	// routes
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.indexHandler))
	mux.Handle("GET /tipp/view/{tippID}", dynamic.ThenFunc(app.tippViewHandler))
	mux.Handle("POST /tipp/update", dynamic.ThenFunc(app.tippUpdateMultipleHandler))

	// standard middleware chain
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
