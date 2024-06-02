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

	// unprotected app routes (no auth required)
	dynamic := alice.New(app.sessionManager.LoadAndSave)

	// general routes
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.indexHandler))

	// user auth routes
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	// protected routes (=auth required)
	protected := dynamic.Append(app.requireAuthentication)

	// admin routes
	mux.Handle("POST /tipp/update", protected.ThenFunc(app.tippUpdateMultipleHandler))
	mux.Handle("GET /tipp/view/{tippID}", protected.ThenFunc(app.tippViewHandler))
	mux.Handle("GET /spiele", protected.ThenFunc(app.matchesHandler))
	mux.Handle("GET /leaderboard", protected.ThenFunc(app.leaderboardHandler))
	mux.Handle("GET /admin", protected.ThenFunc(app.adminIndex))
	mux.Handle("POST /admin/newinvite", protected.ThenFunc(app.adminCreateInvitePost))
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))

	// standard middleware chain
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
