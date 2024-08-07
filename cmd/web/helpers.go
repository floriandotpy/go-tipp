package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/justinas/nosurf"
	"tipp.casualcoding.com/internal/models"
)

func (app *application) serverError(w http.ResponseWriter, req *http.Request, err error) {
	var (
		method = req.Method
		uri    = req.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, r, err)
		return
	}

	// first perform a trial render to catch any runtime errors
	buf := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// only when trial render has completed: write status code and contents
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) templateData {
	authUserId, _ := app.authUserId(r)
	eventFinished, _ := app.eventIsFinished()
	return templateData{
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
		IsAdmin:         app.isAdmin(r),
		CSRFToken:       nosurf.Token(r),
		AuthUserId:      authUserId,
		EventIsFinished: eventFinished,
	}
}

// Create a new decodePostForm() helper method. The second parameter here, dst,
// is the target destination that we want to decode the form data into.
func (app *application) decodePostForm(r *http.Request, dst any) error {

	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		// If we try to use an invalid target destination, raise a panic
		// rather than returning  the error.
		var invalidDecoderError *form.InvalidDecoderError
		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}
		// For all other errors, we return them as normal.
		return err
	}

	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	return app.sessionManager.Exists(r.Context(), "authenticatedUserID")
}

func (app *application) isAdmin(r *http.Request) bool {
	if !app.isAuthenticated(r) {
		return false
	}

	if !app.sessionManager.Exists(r.Context(), "authenticatedUserID") {
		return false
	}

	isAdmin := app.sessionManager.GetBool(r.Context(), "isAdmin")
	return isAdmin
}

func (app *application) authUserId(r *http.Request) (int, error) {
	if !app.isAuthenticated(r) {
		return 0, models.ErrNotLoggedIn
	}
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	return userId, nil
}

func (app *application) eventIsFinished() (bool, error) {
	finished, err := app.matches.AllMatchesFinished()
	if err != nil {
		return false, err
	}
	return finished, nil
}

func (app *application) getGroupID(invite string) (int, error) {

	// check for empty string to avoid accidental match if a group has no invite code set in the db
	if len(invite) < 1 {
		return 0, models.ErrInvalidInvite
	}

	group, err := app.groups.GetByInvite(invite)
	if err != nil {
		return 0, err
	}

	return group.ID, nil
}
