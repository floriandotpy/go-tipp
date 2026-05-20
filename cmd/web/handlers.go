package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tipp.casualcoding.com/internal/api"
	"tipp.casualcoding.com/internal/models"
	appsync "tipp.casualcoding.com/internal/sync"
	"tipp.casualcoding.com/internal/validator"
)

var slugRX = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

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
	app.render(w, req, http.StatusOK, "rules.html", data)
}

func (app *application) boredomHandler(w http.ResponseWriter, req *http.Request) {
	data := app.newTemplateData(req)
	app.render(w, req, http.StatusOK, "boredom.html", data)
}

func (app *application) leaderboardHandler(w http.ResponseWriter, req *http.Request) {

	event := eventFromContext(req)

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
		users, err := app.users.GroupLeaderboard(group.ID, event.ID)
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

	globalLeaderboardUsers, err := app.users.GlobalLeaderboard(event.ID)
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

	event := eventFromContext(req)

	// read phase from URL
	groupsStr := req.URL.Query().Get("groups")

	// split string "1,2,3" into array of int {1,2,3}
	groupsArr := strings.Split(groupsStr, ",")
	var groups []int
	for _, g := range groupsArr {
		gInt, err := strconv.Atoi(g)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		groups = append(groups, gInt)
	}

	response, err := app.tipps.GetScoreboardData(groups, event.ID)
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

	event := eventFromContext(req)

	// read phase from URL
	selectedPhaseStr := req.URL.Query().Get("phase")

	eventPhases, err := app.eventPhases.AllForEvent(event.ID)
	if err != nil {
		app.serverError(w, req, err)
		return
	}

	var phaseId int

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
		todaysPhase, phaseErr := app.eventPhases.DetermineCurrentPhase(event.ID, time.Now())
		if phaseErr != nil {
			// No phase covers "now" — check if event hasn't started yet or is over
			if len(eventPhases) > 0 {
				now := time.Now()
				if now.Before(eventPhases[0].Start) {
					// Event hasn't started yet → show first phase
					phaseId = eventPhases[0].Number
				} else {
					// Event is over → show last phase
					phaseId = eventPhases[len(eventPhases)-1].Number
				}
			} else {
				phaseId = 1
			}
		} else {
			phaseId = todaysPhase.Number
		}
	}

	selectedPhase, err := app.eventPhases.GetByEventAndNumber(event.ID, phaseId)
	if err != nil {
		http.NotFound(w, req)
		return
	}

	// Build prev/next links based on actual phase ordering (handles non-contiguous numbers)
	var nextLink, prevLink string
	for i, ep := range eventPhases {
		if ep.Number == phaseId {
			if i > 0 {
				prevLink = fmt.Sprintf("/spiele?phase=%d", eventPhases[i-1].Number)
			}
			if i < len(eventPhases)-1 {
				nextLink = fmt.Sprintf("/spiele?phase=%d", eventPhases[i+1].Number)
			}
			break
		}
	}

	userId, err := app.authUserId(req)
	if err != nil {
		// TODO: or a proper not authenticated error?
		app.serverError(w, req, err)
	}

	// fetch joined data (matches & tipps)
	matchTipps, err := app.matchTipps.AllByDaterange(userId, event.ID, selectedPhase.Start, selectedPhase.End)
	if err != nil {
		app.serverError(w, req, err)
	}

	data := app.newTemplateData(req)
	data.MatchTipps = matchTipps
	data.EventPhases = eventPhases
	data.SelectedPhase = selectedPhase
	data.NextLink = nextLink
	data.PrevLink = prevLink

	app.render(w, req, http.StatusOK, "matches.html", data)
}

func (app *application) matchDetailsHandler(w http.ResponseWriter, r *http.Request) {

	event := eventFromContext(r)

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

	// Verify match belongs to the resolved event
	if match.EventID != event.ID {
		http.NotFound(w, r)
		return
	}

	data := app.newTemplateData(r)
	data.Match = match

	now := time.Now()
	status := app.matchTipps.MatchStatus(match, now)
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

	// Look up phase type from DB
	eventPhaseType, err := models.InferEventPhaseType(app.matches.DB, &match)
	if err != nil {
		app.serverError(w, r, err)
	}

	if matchHasBegun { // TODO: not pretty: runs second query
		tipps, err := app.tipps.AllForMatch(matchId)
		if err != nil {
			app.serverError(w, r, err)
		}
		data.Tipps = tipps

		var scoreA, scoreB = 0, 0 // no live data yet? default to 0:0
		if match.ResultA != nil && match.ResultB != nil {
			scoreA, scoreB = *match.ResultA, *match.ResultB
		}

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

	event := eventFromContext(r)

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

	// get all matches for the event
	matches, err := app.matches.All(event.ID)
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

func (app *application) wrappedHandler(w http.ResponseWriter, r *http.Request) {

	event := eventFromContext(r)

	userId, err := app.authUserId(r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	groups, err := app.groups.AllForUser(userId)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Determine group-phase and ko-phase numbers dynamically from the event's phases
	eventPhases, err := app.eventPhases.AllForEvent(event.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	var groupPhaseNumbers, koPhaseNumbers []int
	for _, ep := range eventPhases {
		if ep.PhaseType == "phase_group" {
			groupPhaseNumbers = append(groupPhaseNumbers, ep.Number)
		} else if ep.PhaseType == "phase_ko" {
			koPhaseNumbers = append(koPhaseNumbers, ep.Number)
		}
	}

	data := app.newTemplateData(r)
	var stats = []WrappedStats{}
	for _, group := range groups {
		users, err := app.users.GroupLeaderboard(group.ID, event.ID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		var leaderboard = Leaderboard{
			Title: fmt.Sprintf("%s Leaderboard", group.Name),
			ID:    group.ID,
			Users: users,
		}

		var bestInGrouphase []models.User
		if len(groupPhaseNumbers) > 0 {
			bestInGrouphase, err = app.users.GetBestInSelectedPhases(group.ID, event.ID, groupPhaseNumbers)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
		}

		var bestInKoPhase []models.User
		if len(koPhaseNumbers) > 0 {
			bestInKoPhase, err = app.users.GetBestInSelectedPhases(group.ID, event.ID, koPhaseNumbers)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
		}
		var closestGoalCount []models.User

		var wrappedStats = WrappedStats{
			Group:            group,
			Leaderboard:      leaderboard,
			BestInGroupPhase: bestInGrouphase,
			BestInKoPhase:    bestInKoPhase,
			ClosestGoalCount: closestGoalCount,
		}
		stats = append(stats, wrappedStats)
	}
	data.WrappedStatsList = stats
	app.render(w, r, http.StatusOK, "wrapped.html", data)
}

func (app *application) tippUpdateMultipleHandler(w http.ResponseWriter, r *http.Request) {

	event := eventFromContext(r)

	// Only allow tipp submissions for the active event
	if !isActiveEvent(r) {
		app.sessionManager.Put(r.Context(), "flash", "Tipps können nur für das aktive Event abgegeben werden.")
		http.Redirect(w, r, "/spiele", http.StatusSeeOther)
		return
	}

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

			// Verify match belongs to the active event
			match, err := app.matches.Get(matchId)
			if err != nil {
				// skip matches that can't be found
				continue
			}
			if match.EventID != event.ID {
				// skip matches that don't belong to the active event
				continue
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

	events, err := app.events.All()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Events = events

	// Load phases grouped by event
	phasesMap := make(map[int][]models.EventPhase)
	phaseMatchCounts := make(map[int]int)
	for _, event := range events {
		phases, err := app.eventPhases.AllForEvent(event.ID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		phasesMap[event.ID] = phases
		for _, phase := range phases {
			count, err := app.matches.CountByEventAndPhase(event.ID, phase.Number)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
			phaseMatchCounts[phase.ID] = count
		}
	}
	data.EventPhasesMap = phasesMap
	data.PhaseMatchCounts = phaseMatchCounts

	app.render(w, r, http.StatusOK, "admin.html", data)
}

func (app *application) adminCreateInvitePost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Create new invites in the database...")
}

func (app *application) adminUpdatePoints(w http.ResponseWriter, r *http.Request) {

	event := eventFromContext(r)

	rowsAffected, err := app.tipps.UpdatePoints(event.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	msg := fmt.Sprintf("Punkte erfolgreich aktualisiert für %d Einträge", rowsAffected)
	app.sessionManager.Put(r.Context(), "flash", msg)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)

}

// --- Admin Event CRUD ---

type eventCreateForm struct {
	Name               string `form:"name"`
	Slug               string `form:"slug"`
	ApiBaseURL         string `form:"api_base_url"`
	validator.Validator `form:"-"`
}

func (app *application) adminCreateEvent(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = eventCreateForm{}
	app.render(w, r, http.StatusOK, "admin_event_new.html", data)
}

func (app *application) adminCreateEventPost(w http.ResponseWriter, r *http.Request) {
	var form eventCreateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate name: 1-100 chars
	form.CheckField(validator.NotBlank(form.Name), "name", "Darf nicht leer sein")
	form.CheckField(validator.MaxChars(form.Name, 100), "name", "Maximal 100 Zeichen")

	// Validate slug: 1-100 chars, lowercase alphanumeric and hyphens only
	form.CheckField(validator.NotBlank(form.Slug), "slug", "Darf nicht leer sein")
	form.CheckField(validator.MaxChars(form.Slug, 100), "slug", "Maximal 100 Zeichen")
	form.CheckField(validator.Matches(form.Slug, slugRX), "slug", "Nur Kleinbuchstaben, Zahlen und Bindestriche erlaubt")

	// Validate api_base_url: 1-255 chars
	form.CheckField(validator.NotBlank(form.ApiBaseURL), "api_base_url", "Darf nicht leer sein")
	form.CheckField(validator.MaxChars(form.ApiBaseURL, 255), "api_base_url", "Maximal 255 Zeichen")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "admin_event_new.html", data)
		return
	}

	_, err = app.events.Insert(form.Name, form.Slug, form.ApiBaseURL)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateSlug) {
			form.AddFieldError("slug", "Dieser Slug ist bereits vergeben")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "admin_event_new.html", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Event erfolgreich erstellt!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

type eventEditForm struct {
	Name                string `form:"name"`
	Slug                string `form:"slug"`
	ApiBaseURL          string `form:"api_base_url"`
	validator.Validator `form:"-"`
}

func (app *application) adminEditEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	event, err := app.events.Get(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Form = eventEditForm{
		Name:       event.Name,
		Slug:       event.Slug,
		ApiBaseURL: event.ApiBaseURL,
	}
	data.Event = event
	app.render(w, r, http.StatusOK, "admin_event_edit.html", data)
}

func (app *application) adminEditEventPost(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	var form eventEditForm
	err = app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "Darf nicht leer sein")
	form.CheckField(validator.NotBlank(form.Slug), "slug", "Darf nicht leer sein")
	form.CheckField(validator.Matches(form.Slug, slugRX), "slug", "Nur Kleinbuchstaben, Zahlen und Bindestriche erlaubt")
	form.CheckField(validator.NotBlank(form.ApiBaseURL), "api_base_url", "Darf nicht leer sein")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.Event = models.Event{ID: eventID}
		app.render(w, r, http.StatusUnprocessableEntity, "admin_event_edit.html", data)
		return
	}

	err = app.events.Update(eventID, form.Name, form.Slug, form.ApiBaseURL)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateSlug) {
			form.AddFieldError("slug", "Dieser Slug ist bereits vergeben")
			data := app.newTemplateData(r)
			data.Form = form
			data.Event = models.Event{ID: eventID}
			app.render(w, r, http.StatusUnprocessableEntity, "admin_event_edit.html", data)
		} else if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Event erfolgreich aktualisiert!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *application) adminSetActiveEventPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	eventIDStr := r.PostForm.Get("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil || eventID < 1 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.events.SetActive(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.clientError(w, http.StatusNotFound)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Aktives Event erfolgreich geändert!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *application) adminDeleteEventPost(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	err = app.events.Delete(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Event erfolgreich gelöscht!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// --- Admin Phase Edit ---

type phaseEditForm struct {
	Number              int    `form:"number"`
	Title               string `form:"title"`
	PhaseType           string `form:"phase_type"`
	Start               string `form:"start"`
	End                 string `form:"end"`
	validator.Validator `form:"-"`
}

func (app *application) adminEditPhase(w http.ResponseWriter, r *http.Request) {
	phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
	if err != nil || phaseID < 1 {
		http.NotFound(w, r)
		return
	}

	phase, err := app.eventPhases.Get(phaseID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	const timeLayout = "2006-01-02T15:04"
	data := app.newTemplateData(r)
	data.Form = phaseEditForm{
		Number:    phase.Number,
		Title:     phase.Title,
		PhaseType: phase.PhaseType,
		Start:     phase.Start.Format(timeLayout),
		End:       phase.End.Format(timeLayout),
	}
	data.Event = models.Event{ID: phase.EventID}
	app.render(w, r, http.StatusOK, "admin_phase_edit.html", data)
}

func (app *application) adminEditPhasePost(w http.ResponseWriter, r *http.Request) {
	phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
	if err != nil || phaseID < 1 {
		http.NotFound(w, r)
		return
	}

	var form phaseEditForm
	err = app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(form.Number >= 1, "number", "Muss mindestens 1 sein")
	form.CheckField(validator.NotBlank(form.Title), "title", "Darf nicht leer sein")
	form.CheckField(validator.PermittedValue(form.PhaseType, "phase_group", "phase_ko"), "phase_type", "Muss 'phase_group' oder 'phase_ko' sein")

	const timeLayout = "2006-01-02T15:04"
	var startTime, endTime time.Time

	form.CheckField(validator.NotBlank(form.Start), "start", "Darf nicht leer sein")
	form.CheckField(validator.NotBlank(form.End), "end", "Darf nicht leer sein")

	if validator.NotBlank(form.Start) {
		startTime, err = time.Parse(timeLayout, form.Start)
		if err != nil {
			form.AddFieldError("start", "Ungültiges Datumsformat")
		}
	}
	if validator.NotBlank(form.End) {
		endTime, err = time.Parse(timeLayout, form.End)
		if err != nil {
			form.AddFieldError("end", "Ungültiges Datumsformat")
		}
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		form.CheckField(startTime.Before(endTime), "end", "Ende muss nach dem Start liegen")
	}

	if !form.Valid() {
		// Look up the phase to get the event ID for the template
		phase, _ := app.eventPhases.Get(phaseID)
		data := app.newTemplateData(r)
		data.Form = form
		data.Event = models.Event{ID: phase.EventID}
		app.render(w, r, http.StatusUnprocessableEntity, "admin_phase_edit.html", data)
		return
	}

	// Look up existing phase to get event_id
	existingPhase, err := app.eventPhases.Get(phaseID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	ep := models.EventPhase{
		ID:        phaseID,
		EventID:   existingPhase.EventID,
		Number:    form.Number,
		Title:     form.Title,
		PhaseType: form.PhaseType,
		Start:     startTime,
		End:       endTime,
	}

	err = app.eventPhases.Update(ep)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Phase erfolgreich aktualisiert!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *application) adminDeletePhasePost(w http.ResponseWriter, r *http.Request) {
	phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
	if err != nil || phaseID < 1 {
		http.NotFound(w, r)
		return
	}

	err = app.eventPhases.Delete(phaseID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Phase erfolgreich gelöscht!")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// --- Admin Event Sync ---

func (app *application) adminSyncEventGet(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	event, err := app.events.Get(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Fetch all match data from the event's API
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		data := app.newTemplateData(r)
		data.Event = event
		data.ImportError = fmt.Sprintf("Fehler beim Abrufen der API-Daten: %v", err)
		app.render(w, r, http.StatusOK, "admin_sync_preview.html", data)
		return
	}

	// Group matches by groupOrderID
	grouped := appsync.GroupMatches(apiMatches)

	// Sort group keys for consistent display order
	groupKeys := make([]int, 0, len(grouped))
	for k := range grouped {
		groupKeys = append(groupKeys, k)
	}
	sort.Ints(groupKeys)

	// Build preview phases with duplicate detection
	var previewPhases []appsync.SyncPreviewPhase

	for _, groupOrderID := range groupKeys {
		groupMatches := grouped[groupOrderID]
		groupName := groupMatches[0].Group.GroupName

		// Build phase from group
		phase, err := appsync.PhaseFromGroup(eventID, groupOrderID, groupName, groupMatches)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		// Check if phase is new or existing
		isNew := true
		_, err = app.eventPhases.GetByEventAndNumber(eventID, groupOrderID)
		if err == nil {
			isNew = false
		} else if !errors.Is(err, models.ErrNoRecord) {
			app.serverError(w, r, err)
			return
		}

		// Build preview matches with duplicate detection
		var previewMatches []appsync.SyncPreviewMatch
		for _, am := range groupMatches {
			parsedTime, parseErr := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
			if parseErr != nil {
				app.serverError(w, r, parseErr)
				return
			}

			day := parsedTime.Format("2006-01-02")
			isDuplicate := false
			isUpdate := false

			// Look up by API match ID
			existing, err := app.matches.GetByApiMatchID(am.MatchID)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
			if existing.ID != 0 {
				// Match exists — check if anything changed
				if existing.TeamA != am.TeamA.TeamName || existing.TeamB != am.TeamB.TeamName || !existing.Start.Equal(parsedTime) {
					isUpdate = true
				} else {
					isDuplicate = true
				}
			}

			previewMatches = append(previewMatches, appsync.SyncPreviewMatch{
				Date:        day,
				Time:        parsedTime.Format("15:04"),
				TeamA:       am.TeamA.TeamName,
				TeamB:       am.TeamB.TeamName,
				IsDuplicate: isDuplicate,
				IsUpdate:    isUpdate,
				ApiMatchID:  am.MatchID,
			})
		}

		previewPhases = append(previewPhases, appsync.SyncPreviewPhase{
			Phase:   phase,
			IsNew:   isNew,
			Matches: previewMatches,
		})
	}

	data := app.newTemplateData(r)
	data.Event = event
	data.SyncPreviewPhases = previewPhases
	app.render(w, r, http.StatusOK, "admin_sync_preview.html", data)
}

func (app *application) adminSyncEventPost(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	event, err := app.events.Get(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Re-fetch API data
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Group and process
	grouped := appsync.GroupMatches(apiMatches)
	var phasesCreated, phasesUpdated, matchesInserted, matchesUpdated int

	for groupOrderID, groupMatches := range grouped {
		groupName := groupMatches[0].Group.GroupName
		phase, err := appsync.PhaseFromGroup(eventID, groupOrderID, groupName, groupMatches)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		_, isNew, err := app.eventPhases.Upsert(phase)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		if isNew {
			phasesCreated++
		} else {
			phasesUpdated++
		}

		// Upsert matches by API match ID
		for _, am := range groupMatches {
			parsedTime, parseErr := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
			if parseErr != nil {
				app.serverError(w, r, parseErr)
				return
			}

			existing, err := app.matches.GetByApiMatchID(am.MatchID)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
			if existing.ID != 0 {
				// Match exists — update if anything changed
				if existing.TeamA != am.TeamA.TeamName || existing.TeamB != am.TeamB.TeamName || !existing.Start.Equal(parsedTime) {
					err = app.matches.UpdateMatch(existing.ID, am.TeamA.TeamName, am.TeamB.TeamName, parsedTime, phase.PhaseType, groupOrderID)
					if err != nil {
						app.serverError(w, r, err)
						return
					}
					matchesUpdated++
				}
				continue
			}

			_, err = app.matches.Insert(
				am.TeamA.TeamName, am.TeamB.TeamName,
				parsedTime, phase.PhaseType, groupOrderID, eventID, am.MatchID,
			)
			if err != nil {
				app.serverError(w, r, err)
				return
			}
			matchesInserted++
		}
	}

	msg := fmt.Sprintf("Sync abgeschlossen: %d Phasen erstellt, %d aktualisiert, %d Spiele importiert, %d Spiele aktualisiert.",
		phasesCreated, phasesUpdated, matchesInserted, matchesUpdated)
	app.sessionManager.Put(r.Context(), "flash", msg)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
