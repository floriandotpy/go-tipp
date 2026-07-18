package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tipp.casualcoding.com/internal/models"
	statspkg "tipp.casualcoding.com/internal/stats"
)

// findRepoRoot walks up from the current directory until it finds go.mod, so the
// template cache can locate ./ui/html regardless of the test working directory.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root (go.mod)")
		}
		dir = parent
	}
}

// TestWrappedTemplateRenders builds the real template cache and executes the
// wrapped page with representative data, catching template execution errors such
// as missing fields or wrong function signatures.
func TestWrappedTemplateRenders(t *testing.T) {
	root := findRepoRoot(t)
	orig, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(orig)

	cache, err := newTemplateCache()
	if err != nil {
		t.Fatalf("newTemplateCache: %v", err)
	}
	ts, ok := cache["wrapped.html"]
	if !ok {
		t.Fatal("wrapped.html not in template cache")
	}

	data := templateData{
		IsAuthenticated:     true,
		EventIsFinished:     true,
		Event:               models.Event{ID: 1, Name: "FIFA WM 2026"},
		AuthUserId:          7,
		AuthUserName:        "Flo",
		WrappedPersonalName: "Flo",
		WrappedPersonal: statspkg.Personal{
			TotalPoints:       42,
			TotalTipps:        20,
			MatchesTotal:      64,
			MatchesTipped:     20,
			ExactHits:         5,
			ExactHitRate:      0.25,
			LongestStreak:     4,
			AvgPointsPerTipp:  2.1,
			GroupAvg:          2.1,
			KoAvg:             3.4,
			GroupShare:        0.3,
			KoShare:           0.18,
			PhaseVerdict:      "KO-Spezialist",
			BestMatchPoints:   6,
			FavoriteScoreline: "2:1",
			GoalDist: []statspkg.GoalDistBin{
				{Label: "0", Predicted: 1, Actual: 2},
				{Label: "1", Predicted: 3, Actual: 4},
				{Label: "2", Predicted: 6, Actual: 5},
				{Label: "3", Predicted: 5, Actual: 4},
				{Label: "4", Predicted: 3, Actual: 3},
				{Label: "5", Predicted: 1, Actual: 1},
				{Label: "6+", Predicted: 1, Actual: 1},
			},
		},
		WrappedStatsList: []WrappedStats{
			{
				Group: models.Group{ID: 1, Name: "Freunde"},
				Leaderboard: Leaderboard{Users: []models.User{
					{ID: 7, Name: "Flo", Points: 42, Tipps: 20, Place: 1},
					{ID: 8, Name: "Max", Points: 38, Tipps: 20, Place: 2},
					{ID: 9, Name: "Sara", Points: 30, Tipps: 18, Place: 3},
				}},
				BestInGroupPhase: []models.User{
					{ID: 8, Name: "Max", Points: 25, Place: 1},
				},
				BestInKoPhase: []models.User{
					{ID: 7, Name: "Flo", Points: 18, Place: 1},
				},
				Wrapped: statspkg.GroupWrapped{
					GroupID:               1,
					GroupName:             "Freunde",
					PlayerCount:           3,
					EligibleCount:         3,
					TotalTipps:            58,
					AvgPointsPerPlayer:    36.7,
					MostCommonTipp:        "2:1",
					MostCommonTippCount:   9,
					MostCommonActual:      "1:0",
					MostCommonActualCount: 7,
					GoalDist: []statspkg.GoalDistBin{
						{Label: "0", Predicted: 5, Actual: 8},
						{Label: "1", Predicted: 20, Actual: 22},
						{Label: "2", Predicted: 30, Actual: 28},
						{Label: "3", Predicted: 25, Actual: 22},
						{Label: "4", Predicted: 12, Actual: 12},
						{Label: "5", Predicted: 5, Actual: 5},
						{Label: "6+", Predicted: 3, Actual: 3},
					},
					Awards: []statspkg.Award{
						{Key: "aufholjagd", Title: "Beste Aufholjagd", Winner: "Sara", Detail: "von Platz 3 auf Platz 1", Explanation: "Größter Sprung nach vorne."},
						{Key: "stoiker", Title: "Stoischer Tipper", Winner: "Max", Detail: "nur 4 verschiedene Ergebnisse", Explanation: "Tippt am liebsten dasselbe."},
						{Key: "wildcard", Title: "Wildcard-Tipper", Winner: "Flo", Detail: "6 Tipps mit 5+ Toren", Explanation: "Mutige Tipps."},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := ts.ExecuteTemplate(&buf, "base", data); err != nil {
		t.Fatalf("execute wrapped.html: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"Dein Turnier", "Torverteilung", "KO-Spezialist", "Freunde", "2:1",
		"Titel &amp; Auszeichnungen", "Beste Aufholjagd", "Sara", "Gruppen-Statistiken",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q", want)
		}
	}
}
