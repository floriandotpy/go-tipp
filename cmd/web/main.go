package main

import (
	"fmt"
	"net/http"
	// "time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", gamesHandler)
	mux.HandleFunc("GET /tipp/view/{tippID}", tippViewHandler)
	mux.HandleFunc("GET /tipp/create", tippCreateFormHandler)
	mux.HandleFunc("POST /tipp/create", tippCreatePostHandler)

	// http.HandleFunc("/", gamesHandler)
	fmt.Println("Http server listening on port 8090")
	http.ListenAndServe(":8090", mux)
}
