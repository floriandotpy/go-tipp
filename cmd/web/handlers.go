package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tipp.casualcoding.com/internal/models"
	"tipp.casualcoding.com/internal/validator"
)

const TEAM_DE = "Deutschland"
const TEAM_DK = "Dänemark"
const TEAM_ES = "Spanien"
const TEAM_SCO = "Schottland"
const TEAM_FR = "Frankreich"
const TEAM_NL = "Niederlande"
const TEAM_EN = "England"
const TEAM_IT = "Italien"
const TEAM_TR = "Türkei"
const TEAM_HR = "Kroatien"
const TEAM_AL = "Albanien"
const TEAM_CZ = "Tschechien"
const TEAM_BE = "Belgien"
const TEAM_AT = "Österreich"
const TEAM_HU = "Ungarn"
const TEAM_RS = "Serbien"
const TEAM_SI = "Slowenien"
const TEAM_RO = "Rumänien"
const TEAM_CH = "Schweiz"
const TEAM_PT = "Portugal"
const TEAM_SK = "Slowakei"
const TEAM_PL = "Polen"
const TEAM_UA = "Ukraine"
const TEAM_GR = "Griechenland"

func (app *application) indexHandler(w http.ResponseWriter, req *http.Request) {

	matches, err := app.matches.All()
	if err != nil {
		app.serverError(w, req, err)
	}

	// TODO: load user id from current auth session eventually
	userId := 1

	// fetch joined data (matches & tipps)
	matchTipps, err := app.matchTipps.All(userId)
	if err != nil {
		app.serverError(w, req, err)
	}

	data := app.newTemplateData(req)
	data.MatchTipps = matchTipps
	data.Matches = matches

	app.render(w, req, http.StatusOK, "home.html", data)
}

// view a single submitted tipp instance
func (app *application) tippViewHandler(w http.ResponseWriter, r *http.Request) {
	tippId, err := strconv.Atoi(r.PathValue("tippID"))
	if err != nil || tippId < 0 {
		http.NotFound(w, r)
		return
	}

	// Fetch Tipp instance
	tipp, err := app.tipps.Get(tippId)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Fetch corresponding Match instance
	match, err := app.matches.Get(tipp.MatchId)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	fmt.Fprintf(w, "Tipp:\n%+v\nMatch:\n%+v", tipp, match)
}

func (app *application) tippUpdateMultipleHandler(w http.ResponseWriter, r *http.Request) {
	// parse form data
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// TODO: get user id from auth session
	userId := 1

	// Iterate through form data
	for key, values := range r.PostForm {
		if strings.HasPrefix(key, "match_id_") {
			matchIdStr := values[0]
			matchId, err := strconv.Atoi(matchIdStr)
			if err != nil {
				app.clientError(w, http.StatusBadRequest)
				return
			}

			tippAKey := "tipp_a_" + matchIdStr
			tippBKey := "tipp_b_" + matchIdStr

			// check if match still accepts tipps (i.e. it hasn't started yet)
			acceptsTipps, err := app.matches.AcceptsTipps(matchId)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
			if !acceptsTipps {
				// silently skip this one, because we wouldn't want to cancel the whole bulk update
				continue
			}

			tippAStr := r.PostForm.Get(tippAKey)
			tippBStr := r.PostForm.Get(tippBKey)
			if tippAStr == "" || tippBStr == "" {
				// delete previous tipp (user may want to reset it), then continue to next item
				tippExists, err := app.tipps.Exists(matchId, userId)
				if err != nil {
					app.serverError(w, r, err)
					return
				}
				if tippExists {
					err = app.tipps.Delete(matchId, userId)
					if err != nil {
						app.serverError(w, r, err)
						return
					}
				}
				continue
			}

			tippA, err := strconv.Atoi(tippAStr)
			if err != nil {
				app.clientError(w, http.StatusBadRequest)
				return
			}

			tippB, err := strconv.Atoi(tippBStr)
			if err != nil {
				app.clientError(w, http.StatusBadRequest)
				return
			}

			err = app.tipps.InsertOrUpdate(matchId, userId, tippA, tippB)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Tipps gespeichert!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Create a new userSignupForm struct.
type userSignupForm struct {
	Invite              string `form:"invite"`
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, r, http.StatusOK, "signup.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	// Declare an zero-valued instance of our userSignupForm struct.
	var form userSignupForm

	// Parse the form data into the userSignupForm struct.
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate the form contents using our helper functions.
	form.CheckField(validator.NotBlank(form.Invite), "invite", "Darf nicht leer sein")
	form.CheckField(validator.NotBlank(form.Name), "name", "Darf nicht leer sein")
	form.CheckField(validator.NotBlank(form.Email), "email", "Darf nicht leer sein")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "Keine valide E-Mail-Adresse")
	form.CheckField(validator.NotBlank(form.Password), "password", "Darf nicht leer sein")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "Mindestens 8 Zeichen lang")

	// TODO: setup proper invite code management through database eventually
	if form.Invite != "runde-ins-eckige-24" {
		form.AddFieldError("invite", "Dieser Invitecode funktioniert nicht")
	}

	// If there are any errors, redisplay the signup form along with a 422
	// status code.
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "signup.html", data)
		return
	}

	// Otherwise send the placeholder response (for now!).
	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "E-Mail wird bereits verwendet")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, r, err)
		}

		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Anmeldung erfolgreich! Du kannst dich nun einloggen.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Display a form for logging in a user...")
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Authenticate and login the user...")
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Logout the user...")
}

func (app *application) adminIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "List of available admin actions...")
}

func (app *application) adminCreateInvitePost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Create new invites in the database...")
}
