package main

import "testing"

func TestExtractUserDeleteToken(t *testing.T) {
	lines := []string{`To confirm user deletion, send "user delete alice a1b2c3"`}
	token, ok := extractUserDeleteToken(lines, "alice")
	if !ok || token != "a1b2c3" {
		t.Fatalf("unexpected token result: %q %v", token, ok)
	}
}

func TestExtractUserDeleteTokenRejectsWrongUser(t *testing.T) {
	lines := []string{`To confirm user deletion, send "user delete bob a1b2c3"`}
	if token, ok := extractUserDeleteToken(lines, "alice"); ok || token != "" {
		t.Fatalf("unexpected token result: %q %v", token, ok)
	}
}

func TestValidUserDeleteToken(t *testing.T) {
	for _, tc := range []struct {
		token string
		want  bool
	}{
		{"a1b2c3", true},
		{"A1B2C3", false},
		{"a1b2", false},
		{"zzzzzz", false},
	} {
		if got := validUserDeleteToken(tc.token); got != tc.want {
			t.Fatalf("validUserDeleteToken(%q) = %v, want %v", tc.token, got, tc.want)
		}
	}
}
