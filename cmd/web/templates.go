package main

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strconv"
	"time"

	"tipp.casualcoding.com/internal/models"
	"tipp.casualcoding.com/internal/sync"
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

type UserDetailsRow struct {
	MatchNo         int
	TippUser        models.Tipp
	TippCompareUser models.Tipp
	Match           models.Match
}

// type EventStats struct {
// 	MatchCount          int
// 	GoalCount           int
// 	TeamWithMostGoals   string
// 	PlayerWithMostGoals string
// 	EarliestGoal        models.Goal
// 	LatestGoal          models.Goal
// }

type WrappedStats struct {
	Group            models.Group
	Leaderboard      Leaderboard
	BestInGroupPhase []models.User
	BestInKoPhase    []models.User
	ClosestGoalCount []models.User
}

type templateData struct {
	CurrentYear     int
	EventIsFinished bool
	IsActiveEvent   bool
	Event           models.Event
	Events          []models.Event
	MatchTipps      []models.MatchTipp
	Tipps           []models.Tipp
	Groups          []models.Group
	Leaderboards    []Leaderboard
	Goals           []models.Goal
	EventPhases    []models.EventPhase
	EventPhasesMap map[int][]models.EventPhase
	SelectedPhase  models.EventPhase
	LiveResult      LiveResult
	Match           models.Match
	Status          string // move into Match object?
	Flash           string
	Form            any
	IsAuthenticated bool
	IsAdmin         bool
	CSRFToken       string
	AuthUserId      int
	AuthUserName    string
	// for user_details view
	User            models.User
	UserCompare     models.User
	UserDetailsRows []UserDetailsRow
	// pagination of matches
	NextLink string
	PrevLink string
	// wrapped stats
	WrappedStatsList []WrappedStats
	// admin phase match counts: key is phase ID
	PhaseMatchCounts map[int]int
	// import error
	ImportError          string
	// sync preview
	SyncPreviewPhases []sync.SyncPreviewPhase
	// countdown target for index page
	CountdownTarget string
	// admin user list
	Users []models.User
	// API token management
	TokenExists bool
	NewToken    string
	// Job run history for admin
	JobRunsFetchResults []models.JobRun
	JobRunsSyncPhases   []models.JobRun
	// Group members for admin group management
	GroupMembers map[int][]models.User
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

func defaultStr(val *string, defaultStr string) string {
	if val == nil || *val == "" {
		return defaultStr
	} else {
		return *val
	}
}

func defaultIntStr(val *int, defaultStr string) string {
	if val == nil {
		return defaultStr
	} else {
		return strconv.Itoa(*val)
	}
}

func add(x, y int) int {
	return x + y
}

func even(x int) bool {
	return x%2 == 0
}

func odd(x int) bool {
	return (x+1)%2 == 0
}

func isKOPhase(phase models.EventPhase) bool {
	return phase.Number >= 4
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func duration(start, end time.Time) string {
	d := end.Sub(start)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func detailsJSON(data []byte) string {
	if data == nil {
		return ""
	}
	return string(data)
}

// timeUntilKickoff returns a human-readable German string showing how much time
// is left until the match starts. Returns empty string if the match is in the past.
func timeUntilKickoff(start time.Time) string {
	now := time.Now()
	if !start.After(now) {
		return ""
	}

	d := start.Sub(now)

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	var parts []string
	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 Tag")
		} else {
			parts = append(parts, fmt.Sprintf("%d Tage", days))
		}
	}
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 Stunde")
		} else {
			parts = append(parts, fmt.Sprintf("%d Stunden", hours))
		}
	}
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 Minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d Minuten", minutes))
		}
	}

	if len(parts) == 0 {
		return "Noch weniger als eine Minute bis zum Anstoß"
	}

	// Join with " und " for last element, ", " for the rest
	var result string
	if len(parts) == 1 {
		result = parts[0]
	} else if len(parts) == 2 {
		result = parts[0] + " und " + parts[1]
	} else {
		result = parts[0] + ", " + parts[1] + " und " + parts[2]
	}

	return "Noch " + result + " bis zum Anstoß"
}

// staticVersion is set once at startup and used as a cache-busting query parameter.
var staticVersion = fmt.Sprintf("%d", time.Now().Unix())

var functions = template.FuncMap{
	"germanWeekday":    germanWeekday,
	"germanDate":       germanDate,
	"matchResult":      matchResult,
	"defaultIntStr":    defaultIntStr,
	"defaultStr":       defaultStr,
	"add":              add,
	"germanYesNo":      germanYesNo,
	"isLast":           isLast,
	"even":             even,
	"odd":              odd,
	"isKOPhase":        isKOPhase,
	"pluralize":        pluralize,
	"duration":         duration,
	"detailsJSON":      detailsJSON,
	"timeUntilKickoff": timeUntilKickoff,
	"assetVersion":     func() string { return staticVersion },
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
