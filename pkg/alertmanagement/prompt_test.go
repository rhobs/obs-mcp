package alertmanagement

import (
	"strings"
	"testing"
)

func TestServerPromptRequiresDisambiguationAndConfirmation(t *testing.T) {
	checks := []string{
		"same expression",
		"Do not pass every matching id",
		"wait until the user explicitly agrees",
		"edit the owning PrometheusRule or AlertingRule in Git",
		"mute, silence, or stop paging",
		"Do not default to either, and do not invent a PrometheusRule name",
		"Omitting prometheus_rule_name creates a platform",
		"cannot change expr, alert name, for, or annotations",
		"get_alerts returns firing instances",
		"per-rule statusCode",
		"Do not put secrets",
		"Do not retry 400, 404, or 413",
		"list_metrics",
		"alerting_rule_enabled=false",
		"Write tools are always registered",
	}
	for _, want := range checks {
		if !strings.Contains(ServerPrompt, want) {
			t.Errorf("ServerPrompt missing %q", want)
		}
	}
}

func TestWriteToolPromptsCoverCapabilityLimits(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want []string
	}{
		{
			name: "create",
			got:  createAlertRulePrompt,
			want: []string{
				"Do not invent a PrometheusRule name",
				"list_metrics",
				"Do not put secrets",
			},
		},
		{
			name: "update",
			got:  updateAlertRulePrompt,
			want: []string{
				"Cannot change expr, alert name, for, or annotations",
				"platform-only",
				"create_silence",
				"returned id",
			},
		},
		{
			name: "delete",
			got:  deleteAlertRulesPrompt,
			want: []string{
				"not to mute notifications",
				"per-rule statusCode",
			},
		},
		{
			name: "list",
			got:  listAlertRulesPrompt,
			want: []string{
				"not firing instances",
				"do not invent an id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, want := range tt.want {
				if !strings.Contains(tt.got, want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}
