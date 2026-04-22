package config

import (
	"context"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

func uint64Ptr(v uint64) *uint64 { return &v }

func TestGetGuardrails(t *testing.T) {
	tests := []struct {
		name                     string
		config                   Config
		expectNil                bool
		expectErr                bool
		wantDisallowExplicitName bool
		wantRequireLabelMatcher  bool
		wantDisallowBlanketRegex bool
		wantMaxMetricCardinality uint64
		wantMaxLabelCardinality  uint64
	}{
		{
			name:                     "default (empty) enables all guardrails with default cardinality",
			config:                   Config{},
			wantDisallowExplicitName: true,
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 20000,
			wantMaxLabelCardinality:  500,
		},
		{
			name:                     "all enables all guardrails with default cardinality",
			config:                   Config{Guardrails: "all"},
			wantDisallowExplicitName: true,
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 20000,
			wantMaxLabelCardinality:  500,
		},
		{
			name:      "none disables all guardrails",
			config:    Config{Guardrails: "none"},
			expectNil: true,
		},
		{
			name:                     "explicit subset does not enable cardinality guardrails",
			config:                   Config{Guardrails: prometheus.GuardrailRequireLabelMatcher + "," + prometheus.GuardrailDisallowBlanketRegex},
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "explicit subset with user-specified cardinality limits",
			config:                   Config{Guardrails: prometheus.GuardrailRequireLabelMatcher + "," + prometheus.GuardrailDisallowBlanketRegex, MaxMetricCardinality: uint64Ptr(10000), MaxLabelCardinality: uint64Ptr(200)},
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 10000,
			wantMaxLabelCardinality:  200,
		},
		{
			name:                     "single guardrail only enables that one",
			config:                   Config{Guardrails: prometheus.GuardrailRequireLabelMatcher},
			wantRequireLabelMatcher:  true,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "all with custom cardinality limits",
			config:                   Config{Guardrails: "all", MaxMetricCardinality: uint64Ptr(50000), MaxLabelCardinality: uint64Ptr(1000)},
			wantDisallowExplicitName: true,
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 50000,
			wantMaxLabelCardinality:  1000,
		},
		{
			name:                     "all with disabling the cardinality limits",
			config:                   Config{Guardrails: "all", MaxMetricCardinality: uint64Ptr(0), MaxLabelCardinality: uint64Ptr(0)},
			wantDisallowExplicitName: true,
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:      "unknown guardrail returns error",
			config:    Config{Guardrails: "not-a-guardrail"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := tt.config.GetGuardrails()
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.expectNil {
				if g != nil {
					t.Fatalf("expected nil guardrails, got %+v", g)
				}
				return
			}
			if g == nil {
				t.Fatal("expected non-nil guardrails, got nil")
			}
			if g.DisallowExplicitNameLabel != tt.wantDisallowExplicitName {
				t.Errorf("DisallowExplicitNameLabel = %v, want %v", g.DisallowExplicitNameLabel, tt.wantDisallowExplicitName)
			}
			if g.RequireLabelMatcher != tt.wantRequireLabelMatcher {
				t.Errorf("RequireLabelMatcher = %v, want %v", g.RequireLabelMatcher, tt.wantRequireLabelMatcher)
			}
			if g.DisallowBlanketRegex != tt.wantDisallowBlanketRegex {
				t.Errorf("DisallowBlanketRegex = %v, want %v", g.DisallowBlanketRegex, tt.wantDisallowBlanketRegex)
			}
			if g.MaxMetricCardinality != tt.wantMaxMetricCardinality {
				t.Errorf("MaxMetricCardinality = %v, want %v", g.MaxMetricCardinality, tt.wantMaxMetricCardinality)
			}
			if g.MaxLabelCardinality != tt.wantMaxLabelCardinality {
				t.Errorf("MaxLabelCardinality = %v, want %v", g.MaxLabelCardinality, tt.wantMaxLabelCardinality)
			}
		})
	}
}

func TestObsMCPToolsetParser(t *testing.T) {
	type wrapper struct {
		Toolset toml.Primitive `toml:"toolset"`
	}

	tests := []struct {
		name                     string
		tomlInput                string
		expectErr                bool
		wantPrometheusURL        string
		wantGuardrails           string
		wantMaxMetricCardinality *uint64
		wantMaxLabelCardinality  *uint64
	}{
		{
			name:              "cardinality fields omitted leaves pointers nil",
			tomlInput:         `[toolset]` + "\n" + `prometheus_url = "http://localhost:9090"`,
			wantPrometheusURL: "http://localhost:9090",
		},
		{
			name:                     "cardinality fields set to 0 produces non-nil pointers",
			tomlInput:                `[toolset]` + "\n" + `prometheus_url = "http://localhost:9090"` + "\n" + `max_metric_cardinality = 0` + "\n" + `max_label_cardinality = 0`,
			wantPrometheusURL:        "http://localhost:9090",
			wantMaxMetricCardinality: uint64Ptr(0),
			wantMaxLabelCardinality:  uint64Ptr(0),
		},
		{
			name:                     "cardinality fields set to specific values",
			tomlInput:                `[toolset]` + "\n" + `prometheus_url = "http://localhost:9090"` + "\n" + `max_metric_cardinality = 10000` + "\n" + `max_label_cardinality = 200`,
			wantPrometheusURL:        "http://localhost:9090",
			wantMaxMetricCardinality: uint64Ptr(10000),
			wantMaxLabelCardinality:  uint64Ptr(200),
		},
		{
			name:              "all config fields parsed",
			tomlInput:         `[toolset]` + "\n" + `prometheus_url = "http://localhost:9090"` + "\n" + `guardrails = "` + prometheus.GuardrailRequireLabelMatcher + `"` + "\n" + `insecure = true`,
			wantPrometheusURL: "http://localhost:9090",
			wantGuardrails:    prometheus.GuardrailRequireLabelMatcher,
		},
		{
			name: "full config with all guardrails and cardinality limits",
			tomlInput: `[toolset]` + "\n" +
				`prometheus_url = "http://localhost:9090"` + "\n" +
				`alertmanager_url = "http://localhost:9093"` + "\n" +
				`guardrails = "` + prometheus.GuardrailDisallowExplicitNameLabel + `,` + prometheus.GuardrailRequireLabelMatcher + `,` + prometheus.GuardrailDisallowBlanketRegex + `"`,
			wantPrometheusURL: "http://localhost:9090",
			wantGuardrails:    prometheus.GuardrailDisallowExplicitNameLabel + "," + prometheus.GuardrailRequireLabelMatcher + "," + prometheus.GuardrailDisallowBlanketRegex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w wrapper
			md, err := toml.Decode(tt.tomlInput, &w)
			if err != nil {
				t.Fatalf("failed to decode TOML: %v", err)
			}

			cfg, err := obsMCPToolsetParser(context.Background(), w.Toolset, md)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			c := cfg.(*Config)
			if c.PrometheusURL != tt.wantPrometheusURL {
				t.Errorf("PrometheusURL = %q, want %q", c.PrometheusURL, tt.wantPrometheusURL)
			}
			if c.Guardrails != tt.wantGuardrails {
				t.Errorf("Guardrails = %q, want %q", c.Guardrails, tt.wantGuardrails)
			}
			if tt.wantMaxMetricCardinality == nil {
				if c.MaxMetricCardinality != nil {
					t.Errorf("MaxMetricCardinality = %v, want nil", *c.MaxMetricCardinality)
				}
			} else {
				if c.MaxMetricCardinality == nil {
					t.Fatal("MaxMetricCardinality is nil, want non-nil")
				}
				if *c.MaxMetricCardinality != *tt.wantMaxMetricCardinality {
					t.Errorf("MaxMetricCardinality = %d, want %d", *c.MaxMetricCardinality, *tt.wantMaxMetricCardinality)
				}
			}
			if tt.wantMaxLabelCardinality == nil {
				if c.MaxLabelCardinality != nil {
					t.Errorf("MaxLabelCardinality = %v, want nil", *c.MaxLabelCardinality)
				}
			} else {
				if c.MaxLabelCardinality == nil {
					t.Fatal("MaxLabelCardinality is nil, want non-nil")
				}
				if *c.MaxLabelCardinality != *tt.wantMaxLabelCardinality {
					t.Errorf("MaxLabelCardinality = %d, want %d", *c.MaxLabelCardinality, *tt.wantMaxLabelCardinality)
				}
			}
		})
	}
}
