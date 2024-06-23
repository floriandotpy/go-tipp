package api

import "time"

type EventPhase struct {
	Number int
	Title  string
	ApiUrl string
}

func DetermineEventPhase(day time.Time) EventPhase {
	tz := time.Now().Local().Location()
	nonePhase := EventPhase{0, "Unknown", ""}
	phases := []struct {
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
		{time.Date(2024, time.July, 14, 23, 59, 0, 0, tz), nonePhase},
	}

	// out of event time frame?
	if day.After(phases[len(phases)-1].start) || day.Before(phases[0].start) {
		return nonePhase
	}

	for i, p := range phases {
		if day.Before(p.start) {
			return phases[i-1].phase
		}
	}
	return phases[len(phases)-1].phase
}
