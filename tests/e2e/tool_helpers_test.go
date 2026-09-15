//go:build e2e

package e2e

import "testing"

func TestPreviewDisablesEnabledRule(t *testing.T) {
	tests := []struct {
		name    string
		changes []any
		want    bool
	}{
		{name: "unknown state"},
		{name: "malformed change", changes: []any{"invalid"}},
		{name: "enabled", changes: []any{map[string]any{
			"field": "alertingRuleEnabled", "currentValue": true, "newValue": false,
		}}, want: true},
		{name: "already disabled", changes: []any{map[string]any{
			"field": "alertingRuleEnabled", "currentValue": false, "newValue": false,
		}}},
		{name: "restore preview", changes: []any{map[string]any{
			"field": "alertingRuleEnabled", "currentValue": false, "newValue": true,
		}}},
		{name: "unrelated change", changes: []any{map[string]any{
			"field": "severity", "currentValue": true, "newValue": false,
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview := map[string]any{"resources": []any{map[string]any{"changes": tt.changes}}}
			if got := previewDisablesEnabledRule(preview); got != tt.want {
				t.Fatalf("previewDisablesEnabledRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertManagementBackendUnavailable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		errText string
		want    bool
	}{
		{name: "connection refused", errText: "dial tcp 127.0.0.1:9443: connect: connection refused", want: true},
		{name: "stock plugin mux", errText: "404 page not found", want: true},
		{name: "generic page not found", errText: "page not found", want: false},
		{name: "http status 404", errText: "GET /api/v1/alerting/rules: unexpected status 404", want: false},
		{name: "tls certificate", errText: "tls: failed to verify certificate", want: false},
		{name: "generic timeout", errText: "context deadline exceeded: timeout", want: false},
		{name: "i/o timeout", errText: "dial tcp 10.0.0.1:9443: i/o timeout", want: true},
		{name: "invalid source", errText: "invalid source: want platform or user", want: false},
		{name: "forbidden", errText: "error posting silence: [POST /silences] postSilences (status 403): {}", want: false},
		{name: "empty", errText: "", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := alertManagementBackendUnavailable(tc.errText)
			if got != tc.want {
				t.Fatalf("alertManagementBackendUnavailable(%q) = %v, want %v", tc.errText, got, tc.want)
			}
		})
	}
}
