package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

func gamesHandler(w http.ResponseWriter, req *http.Request) {

	games, err := loadGames()
	if err != nil {
		fmt.Println(err)
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
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// execute template
	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}

// view a single submitted tipp instance
func tippViewHandler(w http.ResponseWriter, r *http.Request) {
	tippId, err := strconv.Atoi(r.PathValue("tippID"))
	if err != nil || tippId < 0 {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "Display tipp with id %d ...", tippId)
}

// create a new tipp by user submission
func tippCreateFormHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Display a form for a new tipp...")
}

func tippCreatePostHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Save a new tipp...")
}