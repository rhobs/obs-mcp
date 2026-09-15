package main

import (
	"context"
	"testing"

	"github.com/rhobs/obs-mcp/pkg/auth"
)

// TestParseMetricsBackend verifies the --metrics-backend flag parsing logic
func TestParseMetricsBackend(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "thanos lowercase",
			input:    "thanos",
			expected: "thanos",
			wantErr:  false,
		},
		{
			name:     "thanos uppercase",
			input:    "THANOS",
			expected: "thanos",
			wantErr:  false,
		},
		{
			name:     "prometheus lowercase",
			input:    "prometheus",
			expected: "prometheus",
			wantErr:  false,
		},
		{
			name:     "prometheus mixed case",
			input:    "Prometheus",
			expected: "prometheus",
			wantErr:  false,
		},
		{
			name:     "empty defaults to thanos",
			input:    "",
			expected: "thanos",
			wantErr:  false,
		},
		{
			name:     "invalid backend",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMetricsBackend(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestDetermineMetricsBackendURL_RequiresURLForNonKubeconfigModes verifies that
// header mode returns an error when PROMETHEUS_URL is not set,
// rather than silently falling back to localhost.
func TestDetermineMetricsBackendURL_RequiresURLForNonKubeconfigModes(t *testing.T) {
	t.Setenv("PROMETHEUS_URL", "")

	tests := []struct {
		name     string
		authMode auth.AuthMode
		backend  string
	}{
		{
			name:     "header mode with thanos backend",
			authMode: auth.AuthModeHeader,
			backend:  "thanos",
		},
		{
			name:     "header mode with prometheus backend",
			authMode: auth.AuthModeHeader,
			backend:  "prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := determineMetricsBackendURL(context.Background(), tt.authMode, tt.backend, nil, nil)
			if err == nil {
				t.Errorf("expected error for auth mode %q without PROMETHEUS_URL, got nil", tt.authMode)
			}
		})
	}
}

// TestDetermineMetricsBackendURL_EnvVarOverridesAll verifies that the
// PROMETHEUS_URL environment variable takes highest precedence and
// overrides all other configuration (auth mode, metrics-backend flag).
func TestDetermineMetricsBackendURL_EnvVarOverridesAll(t *testing.T) {
	customURL := "https://custom-prometheus.example.com:9090"
	t.Setenv("PROMETHEUS_URL", customURL)

	authModes := []auth.AuthMode{
		auth.AuthModeKubeConfig,
		auth.AuthModeHeader,
	}

	for _, authMode := range authModes {
		t.Run(string(authMode), func(t *testing.T) {
			url, source, err := determineMetricsBackendURL(context.Background(), authMode, "thanos", nil, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if url != customURL {
				t.Errorf("expected env var URL %q, got %q", customURL, url)
			}
			if source != "PROMETHEUS_URL env var" {
				t.Errorf("expected source %q, got %q", "PROMETHEUS_URL env var", source)
			}
		})
	}
}

func TestDetermineLokiURL(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		t.Setenv("LOKI_URL", "http://from-env:3100")
		got, source, err := determineLokiURL("http://from-flag:3100")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "http://from-flag:3100" || source != "--loki-url flag" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("env used when flag missing", func(t *testing.T) {
		t.Setenv("LOKI_URL", "http://from-env:3100")
		got, source, err := determineLokiURL("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "http://from-env:3100" || source != "LOKI_URL env var" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("without URL returns unset", func(t *testing.T) {
		t.Setenv("LOKI_URL", "")
		got, source, err := determineLokiURL("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" || source != "unset" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})
}

func TestDetermineTempoURL(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		t.Setenv("TEMPO_URL", "http://from-env:3200")
		got, source := determineTempoURL("http://from-flag:3200")
		if got != "http://from-flag:3200" || source != "--traces.tempo-url flag" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("env used when flag missing", func(t *testing.T) {
		t.Setenv("TEMPO_URL", "http://from-env:3200")
		got, source := determineTempoURL("")
		if got != "http://from-env:3200" || source != "TEMPO_URL env var" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("no URL returns unset", func(t *testing.T) {
		t.Setenv("TEMPO_URL", "")
		got, source := determineTempoURL("")
		if got != "" || source != "unset" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})
}

func TestDetermineAlertMgmtAPIURL(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		t.Setenv("ALERT_MGMT_API_URL", "http://from-env:9444")
		got, source, err := determineAlertMgmtAPIURL(auth.AuthModeHeader, "http://from-flag:9444")
		if err != nil {
			t.Fatal(err)
		}
		if got != "http://from-flag:9444" || source != "--alert-mgmt-api-url flag" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("env used when flag missing", func(t *testing.T) {
		t.Setenv("ALERT_MGMT_API_URL", "http://from-env:9444")
		got, source, err := determineAlertMgmtAPIURL(auth.AuthModeHeader, "")
		if err != nil {
			t.Fatal(err)
		}
		if got != "http://from-env:9444" || source != "ALERT_MGMT_API_URL env var" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("kubeconfig falls back to default", func(t *testing.T) {
		t.Setenv("ALERT_MGMT_API_URL", "")
		got, source, err := determineAlertMgmtAPIURL(auth.AuthModeKubeConfig, "")
		if err != nil {
			t.Fatal(err)
		}
		if got != defaultAlertMgmtAPIURL || source != "default" {
			t.Fatalf("unexpected result: %s (%s)", got, source)
		}
	})

	t.Run("header mode requires URL", func(t *testing.T) {
		t.Setenv("ALERT_MGMT_API_URL", "")
		_, _, err := determineAlertMgmtAPIURL(auth.AuthModeHeader, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
