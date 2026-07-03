package sync

import (
	"testing"

	"tipp.casualcoding.com/internal/scoring"
)

func TestInferPhaseType_GroupPhases(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Gruppenphase 1", scoring.PhaseGroup},
		{"Gruppenphase 2", scoring.PhaseGroup},
		{"Gruppenphase 3", scoring.PhaseGroup},
		{"Group A", scoring.PhaseGroup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferPhaseType(tt.name)
			if got != tt.want {
				t.Errorf("InferPhaseType(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestInferPhaseType_KOPhases(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Sechzehntelfinale", scoring.PhaseKO},
		{"Achtelfinale", scoring.PhaseKO},
		{"Viertelfinale", scoring.PhaseKO},
		{"Halbfinale", scoring.PhaseKO},
		{"Finale", scoring.PhaseKO},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferPhaseType(tt.name)
			if got != tt.want {
				t.Errorf("InferPhaseType(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
