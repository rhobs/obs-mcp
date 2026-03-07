package main

import (
	"testing"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/mcp"
)

// TestParseMetricsBackend verifies the --metrics-backend flag parsing logic
func TestParseMetricsBackend(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected k8s.MetricsBackend
		wantErr  bool
	}{
		{
			name:     "thanos lowercase",
			input:    "thanos",
			expected: k8s.MetricsBackendThanos,
			wantErr:  false,
		},
		{
			name:     "thanos uppercase",
			input:    "THANOS",
			expected: k8s.MetricsBackendThanos,
			wantErr:  false,
		},
		{
			name:     "prometheus lowercase",
			input:    "prometheus",
			expected: k8s.MetricsBackendPrometheus,
			wantErr:  false,
		},
		{
			name:     "prometheus mixed case",
			input:    "Prometheus",
			expected: k8s.MetricsBackendPrometheus,
			wantErr:  false,
		},
		{
			name:     "empty defaults to thanos",
			input:    "",
			expected: k8s.MetricsBackendThanos,
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
// serviceaccount and header modes return an error when PROMETHEUS_URL is not set,
// rather than silently falling back to localhost.
func TestDetermineMetricsBackendURL_RequiresURLForNonKubeconfigModes(t *testing.T) {
	t.Setenv("PROMETHEUS_URL", "")

	tests := []struct {
		name     string
		authMode mcp.AuthMode
		backend  k8s.MetricsBackend
	}{
		{
			name:     "serviceaccount mode with thanos backend",
			authMode: mcp.AuthModeServiceAccount,
			backend:  k8s.MetricsBackendThanos,
		},
		{
			name:     "serviceaccount mode with prometheus backend",
			authMode: mcp.AuthModeServiceAccount,
			backend:  k8s.MetricsBackendPrometheus,
		},
		{
			name:     "header mode with thanos backend",
			authMode: mcp.AuthModeHeader,
			backend:  k8s.MetricsBackendThanos,
		},
		{
			name:     "header mode with prometheus backend",
			authMode: mcp.AuthModeHeader,
			backend:  k8s.MetricsBackendPrometheus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := determineMetricsBackendURL(tt.authMode, tt.backend)
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

	authModes := []mcp.AuthMode{
		mcp.AuthModeKubeConfig,
		mcp.AuthModeServiceAccount,
		mcp.AuthModeHeader,
	}

	for _, authMode := range authModes {
		t.Run(string(authMode), func(t *testing.T) {
			url, source, err := determineMetricsBackendURL(authMode, k8s.MetricsBackendThanos)
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
