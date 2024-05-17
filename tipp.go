package main

import (
	"fmt"
	"time"
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

type Game struct {
	TeamA     string
	TeamB     string
	StartTime time.Time

	// goal numbers
	ResultA *int
	ResultB *int
}

type Tipp struct {
	GuessA int
	GuessB int
}

func main() {
	// load the location you want to display the time in
	loc, _ := time.LoadLocation("Europe/Berlin")
	g1 := &Game{TeamA: TEAM_DE, TeamB: TEAM_DK,
		StartTime: time.Date(2024, 6, 14, 21, 0, 0, 0, loc),
		ResultA:   nil, ResultB: nil}
	fmt.Println(g1)
}
