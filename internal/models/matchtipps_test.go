package models

import "testing"

import "time"

func TestMatchStatus_Future(t *testing.T) {
	m := &MatchTippModel{}
	match := Match{Start: time.Now().Add(1 * time.Hour), Finished: false}
	now := time.Now()

	status := m.MatchStatus(match, now)
	if status != MatchFuture {
		t.Errorf("got %q, want %q", status, MatchFuture)
	}
}

func TestMatchStatus_Live(t *testing.T) {
	m := &MatchTippModel{}
	match := Match{Start: time.Now().Add(-1 * time.Hour), Finished: false}
	now := time.Now()

	status := m.MatchStatus(match, now)
	if status != MatchLive {
		t.Errorf("got %q, want %q", status, MatchLive)
	}
}

func TestMatchStatus_Done(t *testing.T) {
	m := &MatchTippModel{}
	match := Match{Start: time.Now().Add(-3 * time.Hour), Finished: true}
	now := time.Now()

	status := m.MatchStatus(match, now)
	if status != MatchDone {
		t.Errorf("got %q, want %q", status, MatchDone)
	}
}

func TestMatchStatus_FutureAndFinished(t *testing.T) {
	// Edge case: match in the future but marked finished (shouldn't happen, but test the logic)
	m := &MatchTippModel{}
	match := Match{Start: time.Now().Add(1 * time.Hour), Finished: true}
	now := time.Now()

	status := m.MatchStatus(match, now)
	// Before start → future takes precedence
	if status != MatchFuture {
		t.Errorf("got %q, want %q", status, MatchFuture)
	}
}
