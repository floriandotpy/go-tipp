package models

import (
	"database/sql"
	"fmt"
	"time"
)

type EventsModel struct {
	DB *sql.DB
}

type EventPhase struct {
	Number int
	Title  string
	ApiUrl string
	Start  time.Time
	End    time.Time
}

var NoPhase = EventPhase{0, "Unknown", "", time.Time{}, time.Time{}}

var Phases = func() []EventPhase {
	tz := time.Now().Local().Location()
	phases := []EventPhase{
		{1, "Gruppenphase 1", "https://api.openligadb.de/getmatchdata/em/2024/1", time.Date(2024, time.June, 14, 0, 0, 0, 0, tz), time.Time{}},
		{2, "Gruppenphase 2", "https://api.openligadb.de/getmatchdata/em/2024/2", time.Date(2024, time.June, 19, 0, 0, 0, 0, tz), time.Time{}},
		{3, "Gruppenphase 3", "https://api.openligadb.de/getmatchdata/em/2024/3", time.Date(2024, time.June, 23, 0, 0, 0, 0, tz), time.Time{}},
		{4, "Achtelfinale", "https://api.openligadb.de/getmatchdata/em/2024/4", time.Date(2024, time.June, 29, 0, 0, 0, 0, tz), time.Time{}},
		{5, "Viertelfinale", "https://api.openligadb.de/getmatchdata/em/2024/5", time.Date(2024, time.July, 5, 0, 0, 0, 0, tz), time.Time{}},
		{6, "Halbfinale", "https://api.openligadb.de/getmatchdata/em/2024/6", time.Date(2024, time.July, 9, 0, 0, 0, 0, tz), time.Time{}},
		{7, "Finale", "https://api.openligadb.de/getmatchdata/em/2024/7", time.Date(2024, time.July, 14, 0, 0, 0, 0, tz), time.Time{}},
	}

	// Set End times
	for i := 0; i < len(phases)-1; i++ {
		phases[i].End = phases[i+1].Start.Add(-time.Nanosecond)
	}
	// Set End time for the last phase (Finale)
	phases[len(phases)-1].End = time.Date(2024, time.July, 14, 23, 59, 59, 999999999, tz)

	return phases
}()

func GetEventPhases() []EventPhase {
	// remove NoPhase
	return Phases
}

func GetEventPhaseById(id int) (EventPhase, error) {
	for _, p := range Phases {
		if p.Number == id {
			return p, nil
		}
	}
	return NoPhase, fmt.Errorf("phase with id %d not found", id)
}

func DetermineEventPhase(day time.Time) EventPhase {
	// out of event time frame?
	if day.Before(Phases[0].Start) {
		return NoPhase
	}

	for i, p := range Phases {
		if day.Before(p.Start) {
			return Phases[i-1]
		}
	}
	return Phases[len(Phases)-1]
}
