package main

import (
	"fmt"
	"net/http"
	// "time"
)

func main() {
	mux := http.NewServeMux()

	// host static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /{$}", gamesHandler)
	mux.HandleFunc("GET /tipp/view/{tippID}", tippViewHandler)
	mux.HandleFunc("GET /tipp/create", tippCreateFormHandler)
	mux.HandleFunc("POST /tipp/create", tippCreatePostHandler)

	// http.HandleFunc("/", gamesHandler)
	fmt.Println("Http server listening on port 8090")
	http.ListenAndServe(":8090", mux)
}
