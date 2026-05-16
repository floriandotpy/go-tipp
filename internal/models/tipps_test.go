package models

import (
	"testing"

	"tipp.casualcoding.com/internal/scoring"
)

func TestComputeLiveTipps_ExactResult_GroupPhase(t *testing.T) {
	m := &TippModel{}
	tipps := []Tipp{{TippA: 2, TippB: 1}}

	result, err := m.ComputeLiveTipps(tipps, 2, 1, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 5 {
		t.Errorf("got %d points, want 5", result[0].Points)
	}
	if !result[0].ResultCorrect {
		t.Error("expected ResultCorrect to be true")
	}
}

func TestComputeLiveTipps_ExactResult_KOPhase(t *testing.T) {
	m := &TippModel{}
	tipps := []Tipp{{TippA: 3, TippB: 0}}

	result, err := m.ComputeLiveTipps(tipps, 3, 0, scoring.PhaseKO)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 6 {
		t.Errorf("got %d points, want 6", result[0].Points)
	}
}

func TestComputeLiveTipps_CorrectTendencyAndDiff(t *testing.T) {
	m := &TippModel{}
	// Tipp: 3-1 (diff +2, team A wins), Result: 2-0 (diff +2, team A wins)
	tipps := []Tipp{{TippA: 3, TippB: 1}}

	result, err := m.ComputeLiveTipps(tipps, 2, 0, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 3 {
		t.Errorf("got %d points, want 3", result[0].Points)
	}
	if !result[0].TendencyCorrect {
		t.Error("expected TendencyCorrect to be true")
	}
	if !result[0].GoalDifferenceCorrect {
		t.Error("expected GoalDifferenceCorrect to be true")
	}
	if result[0].ResultCorrect {
		t.Error("expected ResultCorrect to be false")
	}
}

func TestComputeLiveTipps_CorrectTendencyOnly(t *testing.T) {
	m := &TippModel{}
	// Tipp: 1-0 (diff +1, team A wins), Result: 3-0 (diff +3, team A wins)
	tipps := []Tipp{{TippA: 1, TippB: 0}}

	result, err := m.ComputeLiveTipps(tipps, 3, 0, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 1 {
		t.Errorf("got %d points, want 1", result[0].Points)
	}
	if !result[0].TendencyCorrect {
		t.Error("expected TendencyCorrect to be true")
	}
	if result[0].GoalDifferenceCorrect {
		t.Error("expected GoalDifferenceCorrect to be false")
	}
}

func TestComputeLiveTipps_WrongEverything(t *testing.T) {
	m := &TippModel{}
	// Tipp: 2-0 (team A wins), Result: 0-1 (team B wins)
	tipps := []Tipp{{TippA: 2, TippB: 0}}

	result, err := m.ComputeLiveTipps(tipps, 0, 1, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 0 {
		t.Errorf("got %d points, want 0", result[0].Points)
	}
	if result[0].TendencyCorrect {
		t.Error("expected TendencyCorrect to be false")
	}
}

func TestComputeLiveTipps_Draw(t *testing.T) {
	m := &TippModel{}
	// Tipp: 1-1 (draw), Result: 1-1 (draw) — exact match
	tipps := []Tipp{{TippA: 1, TippB: 1}}

	result, err := m.ComputeLiveTipps(tipps, 1, 1, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Points != 5 {
		t.Errorf("got %d points, want 5 (exact result)", result[0].Points)
	}
}

func TestComputeLiveTipps_DrawTendencyOnly(t *testing.T) {
	m := &TippModel{}
	// Tipp: 0-0 (draw), Result: 2-2 (draw) — tendency correct, diff correct, but not exact
	tipps := []Tipp{{TippA: 0, TippB: 0}}

	result, err := m.ComputeLiveTipps(tipps, 2, 2, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}
	// diff is 0 in both cases, so tendency AND diff are correct
	if result[0].Points != 3 {
		t.Errorf("got %d points, want 3 (tendency + diff)", result[0].Points)
	}
}

func TestComputeLiveTipps_SortsByPointsDescending(t *testing.T) {
	m := &TippModel{}
	tipps := []Tipp{
		{TippA: 0, TippB: 3, UserId: 1}, // wrong tendency → 0 pts
		{TippA: 2, TippB: 1, UserId: 2}, // exact result → 5 pts
		{TippA: 1, TippB: 0, UserId: 3}, // correct tendency only → 1 pt
	}

	result, err := m.ComputeLiveTipps(tipps, 2, 1, scoring.PhaseGroup)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result[0].Points < result[1].Points || result[1].Points < result[2].Points {
		t.Errorf("results not sorted descending: %d, %d, %d",
			result[0].Points, result[1].Points, result[2].Points)
	}
}

func TestComputeLiveTipps_UnknownPhaseType_ReturnsError(t *testing.T) {
	m := &TippModel{}
	tipps := []Tipp{{TippA: 1, TippB: 0}}

	_, err := m.ComputeLiveTipps(tipps, 1, 0, "phase_invalid")
	if err == nil {
		t.Error("expected error for unknown phase type, got nil")
	}
}
