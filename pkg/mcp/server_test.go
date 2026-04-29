package mcp

import (
	"net/http"
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	original := http.Header{
		"Authorization": []string{"Bearer secret-token"},
		"Cookie":        []string{"session=abc123"},
		"Content-Type":  []string{"application/json"},
		"X-Custom":      []string{"safe-value"},
	}

	redacted := redactHeaders(original)

	if got := redacted.Get("Authorization"); got != "[REDACTED]" {
		t.Errorf("Authorization: got %q, want [REDACTED]", got)
	}
	if got := redacted.Get("Cookie"); got != "[REDACTED]" {
		t.Errorf("Cookie: got %q, want [REDACTED]", got)
	}
	if got := redacted.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}
	if got := redacted.Get("X-Custom"); got != "safe-value" {
		t.Errorf("X-Custom: got %q, want safe-value", got)
	}

	if got := original.Get("Authorization"); got != "Bearer secret-token" {
		t.Error("original header was mutated")
	}
}
