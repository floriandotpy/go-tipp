package stats

import (
	"testing"
	"time"

	"tipp.casualcoding.com/internal/scoring"
)

// baseTime is an arbitrary reference kickoff used to order matches in tests.
var baseTime = time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC)

func matchAt(id, offsetHours, a, b int, phase string) MatchInfo {
	return MatchInfo{
		ID:        id,
		Start:     baseTime.Add(time.Duration(offsetHours) * time.Hour),
		ResultA:   a,
		ResultB:   b,
		PhaseType: phase,
	}
}

func TestComputePersonal_Empty(t *testing.T) {
	matches := []MatchInfo{matchAt(1, 0, 2, 1, scoring.PhaseGroup)}
	p := ComputePersonal(matches, nil)

	if p.MatchesTotal != 1 {
		t.Errorf("MatchesTotal = %d, want 1", p.MatchesTotal)
	}
	if p.TotalTipps != 0 {
		t.Errorf("TotalTipps = %d, want 0", p.TotalTipps)
	}
	if p.PhaseVerdict != "Keine Tipps abgegeben" {
		t.Errorf("PhaseVerdict = %q, want %q", p.PhaseVerdict, "Keine Tipps abgegeben")
	}
	if len(p.GoalDist) != len(goalBinLabels) {
		t.Errorf("GoalDist len = %d, want %d", len(p.GoalDist), len(goalBinLabels))
	}
}

func TestComputePersonal_IgnoresTippsOutsideEvent(t *testing.T) {
	matches := []MatchInfo{matchAt(1, 0, 2, 1, scoring.PhaseGroup)}
	tipps := []TippInfo{
		{MatchID: 1, TippA: 2, TippB: 1, Points: 3, ResultCorrect: true},
		{MatchID: 999, TippA: 0, TippB: 0, Points: 3, ResultCorrect: true}, // not in event
	}
	p := ComputePersonal(matches, tipps)

	if p.TotalTipps != 1 {
		t.Errorf("TotalTipps = %d, want 1 (out-of-event tipp must be ignored)", p.TotalTipps)
	}
	if p.TotalPoints != 3 {
		t.Errorf("TotalPoints = %d, want 3", p.TotalPoints)
	}
	if p.ExactHits != 1 {
		t.Errorf("ExactHits = %d, want 1", p.ExactHits)
	}
}

func TestComputePersonal_Aggregates(t *testing.T) {
	matches := []MatchInfo{
		matchAt(1, 0, 2, 1, scoring.PhaseGroup),
		matchAt(2, 2, 0, 0, scoring.PhaseGroup),
		matchAt(3, 4, 3, 1, scoring.PhaseGroup),
		matchAt(4, 6, 1, 0, scoring.PhaseKO),
	}
	tipps := []TippInfo{
		{MatchID: 1, TippA: 2, TippB: 1, Points: 3, ResultCorrect: true},
		{MatchID: 2, TippA: 2, TippB: 1, Points: 0, ResultCorrect: false},
		{MatchID: 3, TippA: 3, TippB: 1, Points: 3, ResultCorrect: true},
		{MatchID: 4, TippA: 1, TippB: 0, Points: 6, ResultCorrect: true},
	}
	p := ComputePersonal(matches, tipps)

	if p.TotalPoints != 12 {
		t.Errorf("TotalPoints = %d, want 12", p.TotalPoints)
	}
	if p.ExactHits != 3 {
		t.Errorf("ExactHits = %d, want 3", p.ExactHits)
	}
	if want := 3.0 / 4.0; p.ExactHitRate != want {
		t.Errorf("ExactHitRate = %v, want %v", p.ExactHitRate, want)
	}
	if p.BestMatchPoints != 6 {
		t.Errorf("BestMatchPoints = %d, want 6", p.BestMatchPoints)
	}
	// Favorite scoreline: "2:1" appears twice, everything else once.
	if p.FavoriteScoreline != "2:1" {
		t.Errorf("FavoriteScoreline = %q, want %q", p.FavoriteScoreline, "2:1")
	}
	// Group avg: (3+0+3)/3 = 2.0 ; KO avg: 6/1 = 6.0
	if p.GroupAvg != 2.0 {
		t.Errorf("GroupAvg = %v, want 2.0", p.GroupAvg)
	}
	if p.KoAvg != 6.0 {
		t.Errorf("KoAvg = %v, want 6.0", p.KoAvg)
	}
	// Avg points per tipp: 12/4 = 3.0
	if p.AvgPointsPerTipp != 3.0 {
		t.Errorf("AvgPointsPerTipp = %v, want 3.0", p.AvgPointsPerTipp)
	}
	// Shares normalize by max per phase: group 2.0/3, KO 6.0/6 = 1.0
	if want := 2.0 / 3.0; p.GroupShare != want {
		t.Errorf("GroupShare = %v, want %v", p.GroupShare, want)
	}
	if p.KoShare != 1.0 {
		t.Errorf("KoShare = %v, want 1.0", p.KoShare)
	}
}

func TestLongestScoringStreak(t *testing.T) {
	matches := []MatchInfo{
		matchAt(1, 0, 0, 0, scoring.PhaseGroup),
		matchAt(2, 2, 0, 0, scoring.PhaseGroup),
		matchAt(3, 4, 0, 0, scoring.PhaseGroup),
		matchAt(4, 6, 0, 0, scoring.PhaseGroup),
		matchAt(5, 8, 0, 0, scoring.PhaseGroup),
	}
	// Points pattern by kickoff: 3, 0, 2, 1, 3 -> longest run of >0 is matches 3,4,5 = 3
	// Provide tipps out of order to verify ordering by kickoff.
	tipps := []TippInfo{
		{MatchID: 5, Points: 3},
		{MatchID: 1, Points: 3},
		{MatchID: 3, Points: 2},
		{MatchID: 2, Points: 0},
		{MatchID: 4, Points: 1},
	}
	p := ComputePersonal(matches, tipps)
	if p.LongestStreak != 3 {
		t.Errorf("LongestStreak = %d, want 3", p.LongestStreak)
	}
}

func TestGoalDistribution(t *testing.T) {
	matches := []MatchInfo{
		matchAt(1, 0, 1, 0, scoring.PhaseGroup), // actual 1 goal
		matchAt(2, 2, 3, 3, scoring.PhaseGroup), // actual 6 goals -> "6+"
		matchAt(3, 4, 2, 2, scoring.PhaseGroup), // actual 4 goals
	}
	tipps := []TippInfo{
		{MatchID: 1, TippA: 2, TippB: 1}, // predicted 3
		{MatchID: 2, TippA: 1, TippB: 0}, // predicted 1
		{MatchID: 3, TippA: 5, TippB: 2}, // predicted 7 -> "6+"
	}
	p := ComputePersonal(matches, tipps)

	byLabel := map[string]GoalDistBin{}
	for _, b := range p.GoalDist {
		byLabel[b.Label] = b
	}

	if byLabel["1"].Actual != 1 {
		t.Errorf("actual goals bucket 1 = %d, want 1", byLabel["1"].Actual)
	}
	if byLabel["4"].Actual != 1 {
		t.Errorf("actual goals bucket 4 = %d, want 1", byLabel["4"].Actual)
	}
	if byLabel["6+"].Actual != 1 {
		t.Errorf("actual goals bucket 6+ = %d, want 1", byLabel["6+"].Actual)
	}
	if byLabel["1"].Predicted != 1 {
		t.Errorf("predicted goals bucket 1 = %d, want 1", byLabel["1"].Predicted)
	}
	if byLabel["3"].Predicted != 1 {
		t.Errorf("predicted goals bucket 3 = %d, want 1", byLabel["3"].Predicted)
	}
	if byLabel["6+"].Predicted != 1 {
		t.Errorf("predicted goals bucket 6+ = %d, want 1", byLabel["6+"].Predicted)
	}
}

func TestPhaseVerdict(t *testing.T) {
	tests := []struct {
		name       string
		groupAvg   float64
		groupTipps int
		koAvg      float64
		koTipps    int
		want       string
	}{
		{"no tipps", 0, 0, 0, 0, "Keine Tipps abgegeben"},
		{"group only", 2, 5, 0, 0, "Gruppenphasen-Tipper"},
		{"ko only", 0, 0, 4, 5, "KO-Tipper"},
		// group norm 3/3=1.0 vs ko norm 2/6=0.33 -> group dominates
		{"group hero", 3, 5, 2, 5, "Gruppenphasen-Held"},
		// ko norm 6/6=1.0 vs group norm 1/3=0.33 -> ko dominates
		{"ko specialist", 1, 5, 6, 5, "KO-Spezialist"},
		// group norm 1.5/3=0.5 vs ko norm 3/6=0.5 -> balanced
		{"balanced", 1.5, 5, 3, 5, "Ausgeglichener Tipper"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := phaseVerdict(tc.groupAvg, tc.groupTipps, tc.koAvg, tc.koTipps)
			if got != tc.want {
				t.Errorf("phaseVerdict = %q, want %q", got, tc.want)
			}
		})
	}
}
