package scoring

import "testing"

func TestPhasePointsMap_GroupPhaseValues(t *testing.T) {
	points, ok := PhasePointsMap[PhaseGroup]
	if !ok {
		t.Fatal("PhaseGroup not found in PhasePointsMap")
	}
	if points.CorrectResult != 3 {
		t.Errorf("CorrectResult: got %d, want 3", points.CorrectResult)
	}
	if points.CorrectTendencyAndDiff != 2 {
		t.Errorf("CorrectTendencyAndDiff: got %d, want 2", points.CorrectTendencyAndDiff)
	}
	if points.CorrectTendency != 1 {
		t.Errorf("CorrectTendency: got %d, want 1", points.CorrectTendency)
	}
}

func TestPhasePointsMap_KOPhaseValues(t *testing.T) {
	points, ok := PhasePointsMap[PhaseKO]
	if !ok {
		t.Fatal("PhaseKO not found in PhasePointsMap")
	}
	if points.CorrectResult != 6 {
		t.Errorf("CorrectResult: got %d, want 6", points.CorrectResult)
	}
	if points.CorrectTendencyAndDiff != 4 {
		t.Errorf("CorrectTendencyAndDiff: got %d, want 4", points.CorrectTendencyAndDiff)
	}
	if points.CorrectTendency != 2 {
		t.Errorf("CorrectTendency: got %d, want 2", points.CorrectTendency)
	}
}

func TestPhasePointsMap_UnknownPhaseReturnsZero(t *testing.T) {
	_, ok := PhasePointsMap["phase_unknown"]
	if ok {
		t.Error("expected unknown phase type to not exist in PhasePointsMap")
	}
}
