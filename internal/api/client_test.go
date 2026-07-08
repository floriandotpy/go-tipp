package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchMatchData_Success(t *testing.T) {
	matches := []ApiMatch{
		{
			MatchDateTime:   "2024-06-14T21:00:00",
			TeamA:           ApiTeam{TeamName: "Deutschland"},
			TeamB:           ApiTeam{TeamName: "Schottland"},
			MatchIsFinished: true,
			Goals:           []ApiGoal{{ScoreTeamA: 1, ScoreTeamB: 0, MatchMinute: 10, GoalGetterName: "Müller"}},
			MatchResults:    []ApiResult{{ResultName: "Endergebnis", PointsTeamA: 5, PointsTeamB: 1}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matches)
	}))
	defer server.Close()

	result, err := FetchMatchData(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].TeamA.TeamName != "Deutschland" {
		t.Errorf("TeamA: got %q, want %q", result[0].TeamA.TeamName, "Deutschland")
	}
	if result[0].TeamB.TeamName != "Schottland" {
		t.Errorf("TeamB: got %q, want %q", result[0].TeamB.TeamName, "Schottland")
	}
	if !result[0].MatchIsFinished {
		t.Error("expected MatchIsFinished to be true")
	}
	if len(result[0].Goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(result[0].Goals))
	}
	if result[0].Goals[0].GoalGetterName != "Müller" {
		t.Errorf("GoalGetterName: got %q, want %q", result[0].Goals[0].GoalGetterName, "Müller")
	}
}

func TestFetchMatchData_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := FetchMatchData(server.URL)
	if err == nil {
		t.Error("expected error for non-200 response, got nil")
	}
}

func TestFetchMatchData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	_, err := FetchMatchData(server.URL)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestConvertApiGoalToGoal(t *testing.T) {
	comment := "Freistoß"
	apiGoal := ApiGoal{
		ScoreTeamA:     2,
		ScoreTeamB:     1,
		MatchMinute:    45,
		GoalGetterID:   123,
		GoalGetterName: "  Müller  ", // has leading/trailing spaces
		IsPenalty:      true,
		IsOwnGoal:      false,
		IsOvertime:     true,
		Comment:        &comment,
	}

	goal := ConvertApiGoalToGoal(apiGoal)

	if goal.ScoreTeamA != 2 {
		t.Errorf("ScoreTeamA: got %d, want 2", goal.ScoreTeamA)
	}
	if goal.ScoreTeamB != 1 {
		t.Errorf("ScoreTeamB: got %d, want 1", goal.ScoreTeamB)
	}
	if goal.MatchMinute != 45 {
		t.Errorf("MatchMinute: got %d, want 45", goal.MatchMinute)
	}
	if goal.GoalGetterID != 123 {
		t.Errorf("GoalGetterID: got %d, want 123", goal.GoalGetterID)
	}
	if goal.GoalGetterName != "Müller" {
		t.Errorf("GoalGetterName: got %q, want %q (should be trimmed)", goal.GoalGetterName, "Müller")
	}
	if !goal.IsPenalty {
		t.Error("expected IsPenalty to be true")
	}
	if goal.IsOwnGoal {
		t.Error("expected IsOwnGoal to be false")
	}
	if !goal.IsOvertime {
		t.Error("expected IsOvertime to be true")
	}
	if goal.Comment == nil || *goal.Comment != "Freistoß" {
		t.Errorf("Comment: got %v, want %q", goal.Comment, "Freistoß")
	}
}

func TestConvertApiGoalToGoal_NilComment(t *testing.T) {
	apiGoal := ApiGoal{
		ScoreTeamA:     1,
		ScoreTeamB:     0,
		GoalGetterName: "Havertz",
		Comment:        nil,
	}

	goal := ConvertApiGoalToGoal(apiGoal)

	if goal.Comment != nil {
		t.Errorf("expected nil Comment, got %v", goal.Comment)
	}
}

func TestExtractMatchResults_PrefersNinetyMinuteResult(t *testing.T) {
	match := ApiMatch{
		MatchResults: []ApiResult{
			{ResultName: "Halbzeit", PointsTeamA: 0, PointsTeamB: 0, ResultTypeID: 1},
			{ResultName: "nach 90 Minuten", PointsTeamA: 0, PointsTeamB: 0, ResultTypeID: 3},
			{ResultName: "nach Verlängerung", PointsTeamA: 0, PointsTeamB: 0, ResultTypeID: 4},
			{ResultName: "nach Elfmeterschießen", PointsTeamA: 4, PointsTeamB: 3, ResultTypeID: 5},
			{ResultName: "Endergebnis", PointsTeamA: 4, PointsTeamB: 3, ResultTypeID: 2},
		},
	}

	results := ExtractMatchResults(match)

	if got := results[ResultRegularTime]; got != [2]int{0, 0} {
		t.Errorf("regular time: got %v, want [0 0]", got)
	}
	if got := results[ResultAfterExtraTime]; got != [2]int{0, 0} {
		t.Errorf("after extra time: got %v, want [0 0]", got)
	}
	if got := results[ResultAfterPenalties]; got != [2]int{4, 3} {
		t.Errorf("after penalties: got %v, want [4 3]", got)
	}
}

func TestExtractMatchResults_FallsBackToEndResultForOlderPayloads(t *testing.T) {
	match := ApiMatch{
		MatchResults: []ApiResult{
			{ResultName: "Halbzeitergebnis", PointsTeamA: 0, PointsTeamB: 1, ResultTypeID: 1},
			{ResultName: "Endergebnis", PointsTeamA: 1, PointsTeamB: 1, ResultTypeID: 2},
			{ResultName: "nach Verlängerung", PointsTeamA: 2, PointsTeamB: 1, ResultTypeID: 4},
		},
	}

	results := ExtractMatchResults(match)

	if got := results[ResultRegularTime]; got != [2]int{1, 1} {
		t.Errorf("regular time: got %v, want [1 1]", got)
	}
	if got := results[ResultAfterExtraTime]; got != [2]int{2, 1} {
		t.Errorf("after extra time: got %v, want [2 1]", got)
	}
}

func TestExtractMatchResults_UsesTypeIDsWhenNamesChange(t *testing.T) {
	match := ApiMatch{
		MatchResults: []ApiResult{
			{ResultName: "regular", PointsTeamA: 2, PointsTeamB: 2, ResultTypeID: 3},
			{ResultName: "extra", PointsTeamA: 3, PointsTeamB: 2, ResultTypeID: 4},
			{ResultName: "penalties", PointsTeamA: 5, PointsTeamB: 4, ResultTypeID: 5},
		},
	}

	results := ExtractMatchResults(match)

	if got := results[ResultRegularTime]; got != [2]int{2, 2} {
		t.Errorf("regular time: got %v, want [2 2]", got)
	}
	if got := results[ResultAfterExtraTime]; got != [2]int{3, 2} {
		t.Errorf("after extra time: got %v, want [3 2]", got)
	}
	if got := results[ResultAfterPenalties]; got != [2]int{5, 4} {
		t.Errorf("after penalties: got %v, want [5 4]", got)
	}
}
