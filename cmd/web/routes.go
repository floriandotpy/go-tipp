package main

import "net/http"

func (app *application) routes() *http.ServeMux {

	mux := http.NewServeMux()

	// serve static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// routes
	mux.HandleFunc("GET /{$}", app.gamesHandler)
	mux.HandleFunc("GET /tipp/view/{tippID}", app.tippViewHandler)
	mux.HandleFunc("GET /tipp/create", app.tippCreateFormHandler)
	mux.HandleFunc("POST /tipp/create", app.tippCreatePostHandler)

	return mux
}
