package main

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strconv"
	"time"

	"tipp.casualcoding.com/internal/models"
)

type Leaderboard struct {
	Title string
	Users []models.User
	ID    int
}

type LiveResult struct {
	ResultA int
	ResultB int
}

type templateData struct {
	CurrentYear     int
	MatchTipps      []models.MatchTipp
	Tipps           []models.Tipp
	Matches         []models.Match
	Groups          []models.Group
	Leaderboards    []Leaderboard
	Goals           []models.Goal
	LiveResult      LiveResult
	Match           models.Match
	Status          string // move into Match object?
	Flash           string
	Form            any
	IsAuthenticated bool
	IsAdmin         bool
	CSRFToken       string
	AuthUserId      int
}

// prep translation mapping
var germanWeekdays = map[string]string{
	"Sunday":    "Sonntag",
	"Monday":    "Montag",
	"Tuesday":   "Dienstag",
	"Wednesday": "Mittwoch",
	"Thursday":  "Donnerstag",
	"Friday":    "Freitag",
	"Saturday":  "Samstag",
}

var germanMonths = map[string]string{
	"January":   "Januar",
	"February":  "Februar",
	"March":     "März",
	"April":     "April",
	"May":       "Mai",
	"June":      "Juni",
	"July":      "Juli",
	"August":    "August",
	"September": "September",
	"October":   "Oktober",
	"November":  "November",
	"December":  "Dezember",
}

func germanWeekday(t time.Time) string {
	englishhWeekday := t.Format("Monday")
	if germanDay, ok := germanWeekdays[englishhWeekday]; ok {
		return germanDay
	}
	return englishhWeekday
}

func germanDate(t time.Time) string {
	day := t.Day()
	monthStr := t.Format("January")
	year := t.Year()
	germanMonth, ok := germanMonths[monthStr]
	if !ok {
		germanMonth = monthStr // fall back to English if lookup fails
	}
	return fmt.Sprintf("%d. %s %d", day, germanMonth, year)
}

func germanYesNo(val bool) string {
	if val {
		return "Ja"
	} else {
		return "Nein"
	}
}

func matchResult(result_a *int, result_b *int) string {
	var str_a string
	var str_b string
	if result_a == nil {
		str_a = "-"
	} else {
		str_a = strconv.Itoa(*result_a)
	}
	if result_b == nil {
		str_b = "-"
	} else {
		str_b = strconv.Itoa(*result_b)
	}
	return fmt.Sprintf("%s:%s", str_a, str_b)

}

func isLast(idx int, goals []models.Goal) bool {
	return idx == len(goals)-1
}

func defaultStr(val *int, defaultStr string) string {
	if val == nil {
		return defaultStr
	} else {
		return strconv.Itoa(*val)
	}
}

func add(x, y int) int {
	return x + y
}

var functions = template.FuncMap{
	"germanWeekday": germanWeekday,
	"germanDate":    germanDate,
	"matchResult":   matchResult,
	"defaultStr":    defaultStr,
	"add":           add,
	"germanYesNo":   germanYesNo,
	"isLast":        isLast,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("./ui/html/pages/*html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.New(name).Funcs(functions).ParseFiles("./ui/html/base.html")
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob("./ui/html/partials/*.html")
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
