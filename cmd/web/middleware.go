package main

import (
	"net/http"
)

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// restrict where the resources can be loaded from
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com")

		// control what information is included in Referer header when a user leaves page
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")

		// prevents content-sniffing attacks
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// prevent clickjacking in older browsers that don’t support CSP headers
		w.Header().Set("X-Frame-Options", "deny")

		// disable the blocking of cross-site scripting attacks, recommended when using CSP headers
		w.Header().Set("X-XSS-Protection", "0")

		w.Header().Set("Server", "Go")

		next.ServeHTTP(w, r)
	})
}
