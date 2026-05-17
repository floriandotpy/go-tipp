package main

import (
	"testing"
	"time"
)

func TestGermanWeekday(t *testing.T) {
	tests := []struct {
		date time.Time
		want string
	}{
		{time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC), "Freitag"},
		{time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), "Samstag"},
		{time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC), "Sonntag"},
		{time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC), "Montag"},
		{time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC), "Dienstag"},
		{time.Date(2024, 6, 19, 0, 0, 0, 0, time.UTC), "Mittwoch"},
		{time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC), "Donnerstag"},
	}

	for _, tt := range tests {
		got := germanWeekday(tt.date)
		if got != tt.want {
			t.Errorf("germanWeekday(%v) = %q, want %q", tt.date, got, tt.want)
		}
	}
}

func TestGermanDate(t *testing.T) {
	tests := []struct {
		date time.Time
		want string
	}{
		{time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), "5. Januar 2024"},
		{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), "15. März 2024"},
		{time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), "31. Dezember 2024"},
	}

	for _, tt := range tests {
		got := germanDate(tt.date)
		if got != tt.want {
			t.Errorf("germanDate(%v) = %q, want %q", tt.date, got, tt.want)
		}
	}
}

func TestGermanYesNo(t *testing.T) {
	if germanYesNo(true) != "Ja" {
		t.Errorf("germanYesNo(true) = %q, want %q", germanYesNo(true), "Ja")
	}
	if germanYesNo(false) != "Nein" {
		t.Errorf("germanYesNo(false) = %q, want %q", germanYesNo(false), "Nein")
	}
}

func TestMatchResult(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		a    *int
		b    *int
		want string
	}{
		{intPtr(2), intPtr(1), "2:1"},
		{intPtr(0), intPtr(0), "0:0"},
		{nil, nil, "-:-"},
		{intPtr(3), nil, "3:-"},
		{nil, intPtr(1), "-:1"},
	}

	for _, tt := range tests {
		got := matchResult(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("matchResult(%v, %v) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAdd(t *testing.T) {
	if add(3, 4) != 7 {
		t.Errorf("add(3, 4) = %d, want 7", add(3, 4))
	}
	if add(-1, 1) != 0 {
		t.Errorf("add(-1, 1) = %d, want 0", add(-1, 1))
	}
}

func TestEvenOdd(t *testing.T) {
	// even(x) returns true when x%2 == 0 (used for table row striping)
	if !even(0) {
		t.Error("even(0) should be true")
	}
	if !even(2) {
		t.Error("even(2) should be true")
	}
	if even(1) {
		t.Error("even(1) should be false")
	}

	// odd(x) returns (x+1)%2 == 0, i.e. true for odd x values (1, 3, 5...)
	// Used for alternating row styles in templates
	if !odd(1) {
		t.Error("odd(1) should be true")
	}
	if !odd(3) {
		t.Error("odd(3) should be true")
	}
	if odd(0) {
		t.Error("odd(0) should be false")
	}
	if odd(2) {
		t.Error("odd(2) should be false")
	}
}
