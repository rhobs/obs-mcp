package main

import (
	"testing"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/mcp"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

func TestApplyCardinalityLimits(t *testing.T) {
	tests := []struct {
		name                     string
		guardrailsFlag           string
		maxMetricCard            uint64
		maxLabelCard             uint64
		wantMaxMetricCardinality uint64
		wantMaxLabelCardinality  uint64
	}{
		{
			name:                     "all guardrails applies defaults",
			guardrailsFlag:           "all",
			maxMetricCard:            20000,
			maxLabelCard:             500,
			wantMaxMetricCardinality: 20000,
			wantMaxLabelCardinality:  500,
		},
		{
			name:                     "empty guardrails flag applies defaults",
			guardrailsFlag:           "",
			maxMetricCard:            20000,
			maxLabelCard:             500,
			wantMaxMetricCardinality: 20000,
			wantMaxLabelCardinality:  500,
		},
		{
			name:                     "explicit subset without cardinality flags does not apply defaults",
			guardrailsFlag:           "require-label-matcher,disallow-blanket-regex",
			maxMetricCard:            20000,
			maxLabelCard:             500,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "explicit subset with max-metric-cardinality in guardrails flag",
			guardrailsFlag:           "require-label-matcher,max-metric-cardinality",
			maxMetricCard:            10000,
			maxLabelCard:             500,
			wantMaxMetricCardinality: 10000,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "explicit subset with max-metric-cardinality flag using default value",
			guardrailsFlag:           "max-metric-cardinality",
			maxMetricCard:            20000,
			wantMaxMetricCardinality: 20000,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "explicit subset with both cardinality guardrails",
			guardrailsFlag:           "disallow-blanket-regex,max-metric-cardinality,max-label-cardinality",
			maxMetricCard:            15000,
			maxLabelCard:             300,
			wantMaxMetricCardinality: 15000,
			wantMaxLabelCardinality:  300,
		},
		{
			name:                     "all guardrails with custom cardinality values",
			guardrailsFlag:           "all",
			maxMetricCard:            50000,
			maxLabelCard:             1000,
			wantMaxMetricCardinality: 50000,
			wantMaxLabelCardinality:  1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &prometheus.Guardrails{}
			applyCardinalityLimits(g, tt.guardrailsFlag, tt.maxMetricCard, tt.maxLabelCard)
			if g.MaxMetricCardinality != tt.wantMaxMetricCardinality {
				t.Errorf("MaxMetricCardinality = %v, want %v", g.MaxMetricCardinality, tt.wantMaxMetricCardinality)
			}
			if g.MaxLabelCardinality != tt.wantMaxLabelCardinality {
				t.Errorf("MaxLabelCardinality = %v, want %v", g.MaxLabelCardinality, tt.wantMaxLabelCardinality)
			}
		})
	}
}

func TestApplyCardinalityLimits_NilGuardrails(t *testing.T) {
	applyCardinalityLimits(nil, "all", 20000, 500)
}

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
