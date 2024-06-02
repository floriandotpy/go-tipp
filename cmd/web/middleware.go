package main

import (
	"fmt"
	"net/http"

	"github.com/justinas/nosurf"
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

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)
		app.logger.Info("received request", "ip", ip, "proto", proto, "method", method, "uri", uri)
		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Not logged in? redirect
		if !app.isAuthenticated(r) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}

		// Logged in? Set the "Cache-Control: no-store" header so that pages
		// require authentication are not cached
		w.Header().Add("Cache-Control", "no-store")

		// Call next handler in chain
		next.ServeHTTP(w, r)
	})
}

// csrf protection
func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true, Path: "/", Secure: true,
	})
	return csrfHandler
}
