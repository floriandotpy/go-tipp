package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

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

type Games struct {
	Games []Game `json:"games"`
}

type Game struct {
	TeamA     string
	TeamB     string
	StartTime string //time.Time
	Title     string

	// goal numbers
	// ResultA *int
	// ResultB *int
}

type Tipp struct {
	GuessA int
	GuessB int
}

func loadGames() (*Games, error) {
	// load games
	jsonFile, err := os.Open("games.json")
	if err != nil {
		return nil, err
	}
	fmt.Println("Opened games.json")
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var games Games
	json.Unmarshal(byteValue, &games)

	return &games, nil
}

func (app *application) gamesHandler(w http.ResponseWriter, req *http.Request) {

	games, err := loadGames()
	if err != nil {
		app.serverError(w, req, err)
	}
	// for now just log to console
	fmt.Println(games)

	// for i := 0; i < len(games.Games); i++ {
	// 	fmt.Fprintln(w, "StartTime: "+games.Games[i].StartTime)
	// 	fmt.Fprintln(w, "TeamA: "+games.Games[i].TeamA)
	// 	fmt.Fprintln(w, "TeamB: "+games.Games[i].TeamB)
	// 	fmt.Fprintln(w, "")
	// }

	// templates
	files := []string{
		"./ui/html/base.html",
		"./ui/html/partials/nav.html",
		"./ui/html/pages/home.html",
	}

	// load template
	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, req, err)
		return
	}

	// execute template
	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		app.serverError(w, req, err)
	}

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
	fmt.Fprintf(w, "Display a form for a new tipp...")
}

func (app *application) tippCreatePostHandler(w http.ResponseWriter, r *http.Request) {
	// dummy data for now
	userId := 1
	matchId := 1
	tippA := 2
	tippB := 1

	id, err := app.tipps.Insert(matchId, userId, tippA, tippB)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/tipp/view/%d", id), http.StatusSeeOther)
}
