package main

import (
	"testing"

	"tipp.casualcoding.com/internal/models"
)

func TestLiveScore(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name    string
		goals   []models.Goal
		resultA *int
		resultB *int
		wantA   int
		wantB   int
	}{
		{
			name:    "no goals, no match result → 0:0",
			goals:   nil,
			resultA: nil,
			resultB: nil,
			wantA:   0,
			wantB:   0,
		},
		{
			name:    "no goals, match result exists → uses match result",
			goals:   nil,
			resultA: intPtr(2),
			resultB: intPtr(1),
			wantA:   2,
			wantB:   1,
		},
		{
			name: "goals exist, no match result → uses last goal score",
			goals: []models.Goal{
				{ScoreTeamA: 1, ScoreTeamB: 0, MatchMinute: 23},
				{ScoreTeamA: 2, ScoreTeamB: 0, MatchMinute: 45},
			},
			resultA: nil,
			resultB: nil,
			wantA:   2,
			wantB:   0,
		},
		{
			name: "goals exist and match result exists → prefers goals",
			goals: []models.Goal{
				{ScoreTeamA: 1, ScoreTeamB: 0, MatchMinute: 10},
				{ScoreTeamA: 1, ScoreTeamB: 1, MatchMinute: 55},
				{ScoreTeamA: 2, ScoreTeamB: 1, MatchMinute: 78},
			},
			resultA: intPtr(1),
			resultB: intPtr(0),
			wantA:   2,
			wantB:   1,
		},
		{
			name: "single goal with minute 0 (freshly scored) → still uses its score",
			goals: []models.Goal{
				{ScoreTeamA: 1, ScoreTeamB: 0, MatchMinute: 0},
			},
			resultA: nil,
			resultB: nil,
			wantA:   1,
			wantB:   0,
		},
		{
			name: "new goal with minute 0, match result still behind → prefers goals",
			goals: []models.Goal{
				{ScoreTeamA: 1, ScoreTeamB: 0, MatchMinute: 35},
				{ScoreTeamA: 1, ScoreTeamB: 1, MatchMinute: 62},
				{ScoreTeamA: 2, ScoreTeamB: 1, MatchMinute: 0},
			},
			resultA: intPtr(1),
			resultB: intPtr(1),
			wantA:   2,
			wantB:   1,
		},
		{
			name:    "partial match result (only one side nil) → falls back to 0:0",
			goals:   nil,
			resultA: intPtr(3),
			resultB: nil,
			wantA:   0,
			wantB:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotB := liveScore(tt.goals, tt.resultA, tt.resultB)
			if gotA != tt.wantA || gotB != tt.wantB {
				t.Errorf("liveScore() = %d:%d, want %d:%d", gotA, gotB, tt.wantA, tt.wantB)
			}
		})
	}
}
