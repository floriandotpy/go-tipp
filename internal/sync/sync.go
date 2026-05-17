package sync

import (
	"strings"
	"time"

	"tipp.casualcoding.com/internal/api"
	"tipp.casualcoding.com/internal/models"
)

// koKeywords are substrings that indicate a knockout phase.
var koKeywords = []string{"Finale", "Viertelfinale", "Halbfinale", "Achtelfinale"}

// InferPhaseType returns "phase_ko" if groupName contains any KO keyword,
// otherwise "phase_group".
func InferPhaseType(groupName string) string {
	for _, kw := range koKeywords {
		if strings.Contains(groupName, kw) {
			return "phase_ko"
		}
	}
	return "phase_group"
}

// GroupMatches groups API matches by their GroupOrderID.
func GroupMatches(matches []api.ApiMatch) map[int][]api.ApiMatch {
	groups := make(map[int][]api.ApiMatch)
	for _, m := range matches {
		groups[m.Group.GroupOrderID] = append(groups[m.Group.GroupOrderID], m)
	}
	return groups
}

// PhaseFromGroup constructs an EventPhase from a group of API matches.
// It sets Number, Title, PhaseType, Start, End, and ApiPath (empty string).
func PhaseFromGroup(eventID int, groupOrderID int, groupName string, matches []api.ApiMatch) (models.EventPhase, error) {
	var earliest, latest time.Time
	for _, m := range matches {
		t, err := time.Parse("2006-01-02T15:04:05", m.MatchDateTime)
		if err != nil {
			return models.EventPhase{}, err
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}

	return models.EventPhase{
		EventID:   eventID,
		Number:    groupOrderID,
		Title:     groupName,
		ApiPath:   "",
		PhaseType: InferPhaseType(groupName),
		Start:     earliest,
		End:       latestEnd(earliest, latest),
	}, nil
}

// latestEnd ensures end is always after start to satisfy the DB CHECK constraint.
// If start == end (single-match group), adds 24 hours to end.
func latestEnd(start, end time.Time) time.Time {
	if !end.After(start) {
		return start.Add(24 * time.Hour)
	}
	return end
}

// SyncPreviewPhase holds phase data plus its matches for the preview template.
type SyncPreviewPhase struct {
	Phase   models.EventPhase
	IsNew   bool // true = will be inserted, false = will be updated
	Matches []SyncPreviewMatch
}

// SyncPreviewMatch holds a single match for the preview template.
type SyncPreviewMatch struct {
	Date        string
	Time        string
	TeamA       string
	TeamB       string
	IsDuplicate bool // exists and unchanged
	IsUpdate    bool // exists but fields differ
	ApiMatchID  int
}
