package validator

import "testing"

func TestNotBlank(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello", true},
		{"  hello  ", true},
		{"", false},
		{"   ", false},
		{"\t\n", false},
	}

	for _, tt := range tests {
		got := NotBlank(tt.input)
		if got != tt.want {
			t.Errorf("NotBlank(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMaxChars(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  bool
	}{
		{"abc", 3, true},
		{"ab", 3, true},
		{"abcd", 3, false},
		{"", 0, true},
		{"über", 4, true},  // 4 runes
		{"über", 3, false}, // 4 runes > 3
	}

	for _, tt := range tests {
		got := MaxChars(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("MaxChars(%q, %d) = %v, want %v", tt.input, tt.max, got, tt.want)
		}
	}
}

func TestMinChars(t *testing.T) {
	tests := []struct {
		input string
		min   int
		want  bool
	}{
		{"abc", 3, true},
		{"abcd", 3, true},
		{"ab", 3, false},
		{"", 1, false},
		{"ä", 1, true}, // 1 rune
	}

	for _, tt := range tests {
		got := MinChars(tt.input, tt.min)
		if got != tt.want {
			t.Errorf("MinChars(%q, %d) = %v, want %v", tt.input, tt.min, got, tt.want)
		}
	}
}

func TestMatches_Email(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"user@example.com", true},
		{"user+tag@example.co.uk", true},
		{"@example.com", false},
		{"user@", false},
		{"not-an-email", false},
		{"", false},
	}

	for _, tt := range tests {
		got := Matches(tt.input, EmailRX)
		if got != tt.want {
			t.Errorf("Matches(%q, EmailRX) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMatches_Username(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"flo", true},
		{"user_123", true},
		{"ab", false},                    // too short (< 3)
		{"a_very_long_username_here!", false}, // too long / invalid chars
		{"has space", false},
		{"has-dash", false},
	}

	for _, tt := range tests {
		got := Matches(tt.input, UsernameRX)
		if got != tt.want {
			t.Errorf("Matches(%q, UsernameRX) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPermittedValue(t *testing.T) {
	if !PermittedValue("a", "a", "b", "c") {
		t.Error("expected 'a' to be permitted")
	}
	if PermittedValue("d", "a", "b", "c") {
		t.Error("expected 'd' to not be permitted")
	}
	if !PermittedValue(1, 1, 2, 3) {
		t.Error("expected 1 to be permitted")
	}
	if PermittedValue(4, 1, 2, 3) {
		t.Error("expected 4 to not be permitted")
	}
}

func TestValidator_CheckField_Valid(t *testing.T) {
	v := Validator{}
	v.CheckField(true, "name", "must not be blank")

	if !v.Valid() {
		t.Error("expected validator to be valid when check passes")
	}
}

func TestValidator_CheckField_Invalid(t *testing.T) {
	v := Validator{}
	v.CheckField(false, "name", "must not be blank")

	if v.Valid() {
		t.Error("expected validator to be invalid when check fails")
	}
	if v.FieldErrors["name"] != "must not be blank" {
		t.Errorf("got error %q, want %q", v.FieldErrors["name"], "must not be blank")
	}
}

func TestValidator_AddFieldError_NoDuplicates(t *testing.T) {
	v := Validator{}
	v.AddFieldError("email", "first error")
	v.AddFieldError("email", "second error")

	if v.FieldErrors["email"] != "first error" {
		t.Errorf("expected first error to be kept, got %q", v.FieldErrors["email"])
	}
}

func TestValidator_NonFieldErrors(t *testing.T) {
	v := Validator{}
	v.AddNonFieldError("something went wrong")

	if v.Valid() {
		t.Error("expected validator to be invalid with non-field errors")
	}
	if len(v.NonFieldErrors) != 1 {
		t.Fatalf("expected 1 non-field error, got %d", len(v.NonFieldErrors))
	}
	if v.NonFieldErrors[0] != "something went wrong" {
		t.Errorf("got %q, want %q", v.NonFieldErrors[0], "something went wrong")
	}
}
