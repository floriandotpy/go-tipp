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
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf)

	// general routes
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.indexHandler))

	// user auth routes
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	// protected routes (=auth required)
	protected := dynamic.Append(app.requireAuthentication)

	// protected routes (=login required)
	mux.Handle("POST /tipp/update", protected.ThenFunc(app.tippUpdateMultipleHandler))
	mux.Handle("GET /tipp/view/{tippID}", protected.ThenFunc(app.tippViewHandler))
	mux.Handle("GET /spiele", protected.ThenFunc(app.matchesHandler))
	mux.Handle("GET /spiel/{matchID}", protected.ThenFunc(app.matchDetailsHandler))
	mux.Handle("GET /leaderboard", protected.ThenFunc(app.leaderboardHandler))
	mux.Handle("GET /scores.json", protected.ThenFunc(app.scoresJsonHandler))
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))

	// admin routes
	admin := dynamic.Append(app.requireAdminAuthentication)
	mux.Handle("GET /admin", admin.ThenFunc(app.adminIndex))
	mux.Handle("POST /admin/newinvite", admin.ThenFunc(app.adminCreateInvitePost))
	mux.Handle("POST /admin/updatepoints", admin.ThenFunc(app.adminUpdatePoints))

	// standard middleware chain
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
