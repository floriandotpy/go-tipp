package models

import (
	"database/sql"
	"fmt"
	"time"
)

type EventPhase struct {
	Number int
	Title  string
	ApiUrl string
}

type EventsModel struct {
	DB *sql.DB
}

var NoPhase = EventPhase{0, "Unknown", ""}

var Phases = func() []struct {
	start time.Time
	phase EventPhase
} {
	tz := time.Now().Local().Location()
	return []struct {
		start time.Time
		phase EventPhase
	}{
		{time.Date(2024, time.June, 14, 0, 0, 0, 0, tz), EventPhase{1, "Gruppenphase 1", "https://api.openligadb.de/getmatchdata/em/2024/1"}},
		{time.Date(2024, time.June, 19, 0, 0, 0, 0, tz), EventPhase{2, "Gruppenphase 2", "https://api.openligadb.de/getmatchdata/em/2024/2"}},
		{time.Date(2024, time.June, 23, 0, 0, 0, 0, tz), EventPhase{3, "Gruppenphase 3", "https://api.openligadb.de/getmatchdata/em/2024/3"}},
		{time.Date(2024, time.June, 29, 0, 0, 0, 0, tz), EventPhase{4, "Achtelfinale", "https://api.openligadb.de/getmatchdata/em/2024/4"}},
		{time.Date(2024, time.July, 5, 0, 0, 0, 0, tz), EventPhase{5, "Viertelfinale", "https://api.openligadb.de/getmatchdata/em/2024/5"}},
		{time.Date(2024, time.July, 9, 0, 0, 0, 0, tz), EventPhase{6, "Halbfinale", "https://api.openligadb.de/getmatchdata/em/2024/6"}},
		{time.Date(2024, time.July, 14, 0, 0, 0, 0, tz), EventPhase{7, "Finale", "https://api.openligadb.de/getmatchdata/em/2024/7"}},
		{time.Date(2024, time.July, 14, 23, 59, 0, 0, tz), NoPhase},
	}
}()

func GetEventPhases() []EventPhase {
	phases := make([]EventPhase, len(Phases))
	for i, p := range Phases {
		phases[i] = p.phase
	}

	// remove NoPhase
	phases = phases[:len(phases)-1]

	return phases
}

func GetEventPhaseById(id int) (EventPhase, error) {
	for _, p := range Phases {
		if p.phase.Number == id {
			return p.phase, nil
		}
	}
	return NoPhase, fmt.Errorf("phase with id %d not found", id)
}

func DetermineEventPhase(day time.Time) EventPhase {
	// out of event time frame?
	if day.After(Phases[len(Phases)-1].start) || day.Before(Phases[0].start) {
		return NoPhase
	}

	for i, p := range Phases {
		if day.Before(p.start) {
			return Phases[i-1].phase
		}
	}
	return Phases[len(Phases)-1].phase
}
