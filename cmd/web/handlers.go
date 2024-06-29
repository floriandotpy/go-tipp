package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	data := app.newTemplateData(req)

	// if user is logged in, go to leaderboard
	if app.isAuthenticated(req) {
		http.Redirect(w, req, "/leaderboard", http.StatusTemporaryRedirect)
	}

	app.render(w, req, http.StatusOK, "index.html", data)
}

func (app *application) rulesHandler(w http.ResponseWriter, req *http.Request) {
	data := app.newTemplateData(req)
	fmt.Println("rulesHandler")
	app.render(w, req, http.StatusOK, "rules.html", data)
}

func (app *application) leaderboardHandler(w http.ResponseWriter, req *http.Request) {

	// fetch all user groups from database
	userId, err := app.authUserId(req)
	if err != nil {
		app.serverError(w, req, err)
		return
	}
	groups, err := app.groups.AllForUser(userId)
	if err != nil {
		app.serverError(w, req, err)
		return
	}

	var leaderboards []Leaderboard
	for _, group := range groups {
		users, err := app.users.GroupLeaderboard(group.ID)
		if err != nil {
			app.serverError(w, req, err)
			return
		}

		var leaderboard = Leaderboard{
			Title: fmt.Sprintf("%s Leaderboard", group.Name),
			ID:    group.ID,
			Users: users,
		}

		leaderboards = append(leaderboards, leaderboard)
	}

	globalLeaderboardUsers, err := app.users.GlobalLeaderboard()
	if err != nil {
		app.serverError(w, req, err)
		return
	}
	var globalLeaderboard = Leaderboard{
		Title: "Global Leaderboard",
		Users: globalLeaderboardUsers,
	}
	leaderboards = append(leaderboards, globalLeaderboard)

	data := app.newTemplateData(req)
	data.Leaderboards = leaderboards

	app.render(w, req, http.StatusOK, "leaderboard.html", data)
}

func (app *application) scoresJsonHandler(w http.ResponseWriter, req *http.Request) {

	response, err := app.tipps.GetScoreboardData()
	if err != nil {
		app.serverError(w, req, err)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (app *application) matchesHandler(w http.ResponseWriter, req *http.Request) {

	// read phase from URL
	selectedPhaseStr := req.URL.Query().Get("phase")

	eventPhases := models.GetEventPhases()

	var phaseId int
	var err error

	// phase given? convert to numeric phase id
	if selectedPhaseStr != "" {
		phaseId, err = strconv.Atoi(selectedPhaseStr)
		if err != nil || phaseId < 0 {
			http.NotFound(w, req)
			return
		}
	}

	// no phase id set? determine current phase
	if phaseId == 0 {
		// default to today's phase
		todaysPhase := models.DetermineEventPhase(time.Now())
		phaseId = todaysPhase.Number

		// if today is not a match day, assume the event is over and show the final phase
		if todaysPhase.Number == 0 {
			phaseId = eventPhases[len(eventPhases)-1].Number
		}
	}

	selectedPhase, err := models.GetEventPhaseById(phaseId)
	if err != nil {
		http.NotFound(w, req)
		return
	}

	if err != nil {
		app.serverError(w, req, err)
	}

	userId, err := app.authUserId(req)
	if err != nil {
		// TODO: or a proper not authenticated error?
		app.serverError(w, req, err)
	}

	// fetch joined data (matches & tipps)
	matchTipps, err := app.matchTipps.AllByDaterange(userId, selectedPhase.Start, selectedPhase.End)
	if err != nil {
		app.serverError(w, req, err)
	}

	data := app.newTemplateData(req)
	data.MatchTipps = matchTipps
	data.EventPhases = models.GetEventPhases()
	data.SelectedPhase = selectedPhase

	app.render(w, req, http.StatusOK, "matches.html", data)
}

func (app *application) matchDetailsHandler(w http.ResponseWriter, r *http.Request) {

	matchId, err := strconv.Atoi(r.PathValue("matchID"))
	if err != nil || matchId < 0 {
		http.NotFound(w, r)
		return
	}
	match, err := app.matches.Get(matchId)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Match = match

	now := time.Now()
	status := app.matchTipps.MatchStatus(match.Start, now, match.ResultA, match.ResultB)
	data.Status = status

	// fetch goals (will work on live matches and finished matches both)
	goals, err := app.goals.AllForMatch(matchId)
	if err != nil {
		app.serverError(w, r, err)
	}
	data.Goals = goals

	// below: if match has begun, we can display other users' tipps
	matchHasBegun, err := app.matches.MatchHasBegun(matchId)
	if err != nil {
		app.serverError(w, r, err)
	}

	eventPhaseType, err := models.InferEventPhaseType(&match)
	if err != nil {
		app.serverError(w, r, err)
	}

	if matchHasBegun { // TODO: not pretty: runs second query
		tipps, err := app.tipps.AllForMatch(matchId)
		if err != nil {
			app.serverError(w, r, err)
		}
		data.Tipps = tipps

		scoreA, scoreB := *match.ResultA, *match.ResultB
		liveTipps, err := app.tipps.ComputeLiveTipps(tipps, scoreA, scoreB, eventPhaseType)
		if err != nil {
			app.serverError(w, r, err)
		}
		data.Tipps = liveTipps

		liveResult := LiveResult{
			ResultA: scoreA,
			ResultB: scoreB,
		}
		data.LiveResult = liveResult
	}

	app.render(w, r, http.StatusOK, "match_details.html", data)
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

func (app *application) userDetailsHandler(w http.ResponseWriter, r *http.Request) {
	userName := r.PathValue("name")
	if len(userName) < 1 {
		http.NotFound(w, r)
		return
	}

	user, err := app.users.GetByUsername(userName)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	tipps, err := app.tipps.AllForUser(user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Tipps = tipps
	data.User = user

	authUserId, err := app.authUserId(r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// looking at someone else's profile?
	// Also include current user's data for comparison
	var tippsCompare []models.Tipp
	if user.ID != authUserId {
		userCompare, err := app.users.Get(authUserId)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		tippsCompare, err = app.tipps.AllForUser(authUserId)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		data.UserCompare = userCompare
	}

	// prepare tipp sets for match id lookup
	var tippsSet = make(map[int]models.Tipp)
	for _, tippA := range tipps {
		tippsSet[tippA.MatchId] = tippA
	}
	var tippsCompareSet = make(map[int]models.Tipp)
	for _, tippB := range tippsCompare {
		tippsCompareSet[tippB.MatchId] = tippB
	}

	// get all matches
	matches, err := app.matches.All()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var matchesSet = make(map[int]models.Match)
	var rows []UserDetailsRow
	for i, match := range matches {
		matchesSet[i] = match
		row := UserDetailsRow{
			MatchNo:         i + 1,
			TippUser:        tippsSet[match.ID],
			TippCompareUser: tippsCompareSet[match.ID],
			Match:           match,
		}
		rows = append(rows, row)
	}
	data.UserDetailsRows = rows

	app.render(w, r, http.StatusOK, "user_details.html", data)

}

func (app *application) tippUpdateMultipleHandler(w http.ResponseWriter, r *http.Request) {
	// parse form data
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// get phase from request
	phase, err := strconv.Atoi(r.URL.Query().Get("phase"))
	if err != nil {
		phase = 0
	}

	// get user id from session
	userId, err := app.authUserId(r)
	if err != nil {
		// TODO: or a proper not authenticated error?
		app.serverError(w, r, err)
	}

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

			// check if match accepts tipps (i.e. it hasn't started yet and both teams are known)
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

	var redirectUrl = "/spiele"
	if phase != 0 {
		redirectUrl = fmt.Sprintf("/spiele?phase=%d", phase)
	}

	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
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
	form := userSignupForm{
		Invite: r.URL.Query().Get("invite"),
	}
	data.Form = form
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
	form.CheckField(validator.Matches(form.Name, validator.UsernameRX), "name", "Kein valider Username")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "Keine valide E-Mail-Adresse")
	form.CheckField(validator.NotBlank(form.Password), "password", "Darf nicht leer sein")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "Mindestens 8 Zeichen lang")

	// TODO: setup proper invite code management through database eventually
	groupId, err := app.getGroupID(form.Invite)
	if err != nil {
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
	userId, err := app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) || errors.Is(err, models.ErrDuplicateName) {

			if errors.Is(err, models.ErrDuplicateEmail) {
				form.AddFieldError("email", "E-Mail wird bereits verwendet")
			} else if errors.Is(err, models.ErrDuplicateName) {
				form.AddFieldError("name", "Username bereits vergeben")
			}
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, r, err)
		}

		return
	}

	app.groups.AddUserToGroup(userId, groupId)

	app.sessionManager.Put(r.Context(), "flash", "Registrierung erfolgreich! Du kannst dich nun einloggen.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {

	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "Darf nicht leer sein")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "Keine valide E-Mail-Adresse")
	form.CheckField(validator.NotBlank(form.Password), "password", "Darf nicht leer sein")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.html", data)
		return
	}

	id, isAdmin, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("E-Mail oder Passwort sind nicht korrekt")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Use the RenewToken() method on the current session to change the session
	// ID. It's good practice to generate a new session ID when the
	// authentication state or privilege levels changes for the user (e.g. login
	// and logout operations).
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	app.sessionManager.Put(r.Context(), "isAdmin", isAdmin)

	http.Redirect(w, r, "/spiele", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Remove(r.Context(), "isAdmin")

	app.sessionManager.Put(r.Context(), "flash", "Erfolgreich ausgeloggt")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) adminIndex(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	groups, err := app.groups.All()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Groups = groups
	app.render(w, r, http.StatusOK, "admin.html", data)
}

func (app *application) adminCreateInvitePost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Create new invites in the database...")
}

func (app *application) adminUpdatePoints(w http.ResponseWriter, r *http.Request) {

	rowsAffected, err := app.tipps.UpdatePoints()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	msg := fmt.Sprintf("Punkte erfolgreich aktualisiert für %d Einträge", rowsAffected)
	app.sessionManager.Put(r.Context(), "flash", msg)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)

}
