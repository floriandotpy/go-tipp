package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tipp.casualcoding.com/internal/models"
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
		fmt.Printf("errrrrrror!!!!!")
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

// create a new tipp by user submission
func (app *application) tippCreateFormHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: probably not needed because this all happens on the index
	fmt.Fprintf(w, "Display a form for a new tipp...")
}

func (app *application) tippCreatePostHandler(w http.ResponseWriter, r *http.Request) {
	// parse form data
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// TODO: get user id from auth session
	userId := 1

	// read form data
	matchId, err := strconv.Atoi(r.PostForm.Get("match_id"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}
	tippA, err := strconv.Atoi(r.PostForm.Get("tipp_a"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}
	tippB, err := strconv.Atoi(r.PostForm.Get("tipp_b"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}

	_, err = app.tipps.Insert(matchId, userId, tippA, tippB)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
