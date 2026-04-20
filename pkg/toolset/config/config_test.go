package config

import (
	"testing"
)

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
			config:                   Config{Guardrails: "require-label-matcher,disallow-blanket-regex"},
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "explicit subset with user-specified cardinality limits",
			config:                   Config{Guardrails: "require-label-matcher,disallow-blanket-regex", MaxMetricCardinality: 10000, MaxLabelCardinality: 200},
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 10000,
			wantMaxLabelCardinality:  200,
		},
		{
			name:                     "single guardrail only enables that one",
			config:                   Config{Guardrails: "require-label-matcher"},
			wantRequireLabelMatcher:  true,
			wantMaxMetricCardinality: 0,
			wantMaxLabelCardinality:  0,
		},
		{
			name:                     "all with custom cardinality limits",
			config:                   Config{Guardrails: "all", MaxMetricCardinality: 50000, MaxLabelCardinality: 1000},
			wantDisallowExplicitName: true,
			wantRequireLabelMatcher:  true,
			wantDisallowBlanketRegex: true,
			wantMaxMetricCardinality: 50000,
			wantMaxLabelCardinality:  1000,
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
