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

	// general routes
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.indexHandler))
	mux.Handle("GET /tipp/view/{tippID}", dynamic.ThenFunc(app.tippViewHandler))
	mux.Handle("POST /tipp/update", dynamic.ThenFunc(app.tippUpdateMultipleHandler))

	// user auth routes
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))
	mux.Handle("POST /user/logout", dynamic.ThenFunc(app.userLogoutPost))

	// admin routes
	mux.Handle("GET /admin", dynamic.ThenFunc(app.adminIndex))
	mux.Handle("POST /admin/newinvite", dynamic.ThenFunc(app.adminCreateInvitePost))

	// standard middleware chain
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
