package mcp

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/require"

	"github.com/rhobs/obs-mcp/pkg/alertmanagement"
	"github.com/rhobs/obs-mcp/pkg/metrics"
)

func TestIsToolApplicable(t *testing.T) {
	tools := append((&metrics.Toolset{}).GetTools(nil), (&alertmanagement.Toolset{}).GetTools(nil)...)
	byName := toolByNameIndex(t, tools)

	tests := []struct {
		name               string
		tool               string
		readOnly           bool
		disableDestructive bool
		want               bool
	}{
		{name: "list_metrics stays in read-only", tool: "list_metrics", readOnly: true, want: true},
		{name: "get_silences stays in read-only", tool: "get_silences", readOnly: true, want: true},
		{name: "list_alert_rules stays in read-only", tool: "list_alert_rules", readOnly: true, want: true},
		{name: "list_alerts stays in read-only", tool: "list_alerts", readOnly: true, want: true},
		{name: "preview_alert_rule stays in read-only", tool: "preview_alert_rule", readOnly: true, want: true},
		{name: "create_silence hidden in read-only", tool: "create_silence", readOnly: true, want: false},
		{name: "update_silence hidden in read-only", tool: "update_silence", readOnly: true, want: false},
		{name: "delete_silence hidden in read-only", tool: "delete_silence", readOnly: true, want: false},
		{name: "create_alert_rule hidden in read-only", tool: "create_alert_rule", readOnly: true, want: false},
		{name: "update_alert_rule hidden in read-only", tool: "update_alert_rule", readOnly: true, want: false},
		{name: "delete_alert_rules hidden in read-only", tool: "delete_alert_rules", readOnly: true, want: false},
		{name: "create_silence registered when writes enabled", tool: "create_silence", want: true},
		{name: "create_silence kept when only destructive is disabled", tool: "create_silence", disableDestructive: true, want: true},
		{name: "create_alert_rule kept when only destructive is disabled", tool: "create_alert_rule", disableDestructive: true, want: true},
		{name: "delete_silence hidden when destructive is disabled", tool: "delete_silence", disableDestructive: true, want: false},
		{name: "update_alert_rule hidden when destructive is disabled", tool: "update_alert_rule", disableDestructive: true, want: false},
		{name: "delete_alert_rules hidden when destructive is disabled", tool: "delete_alert_rules", disableDestructive: true, want: false},
		{name: "preview_alert_rule stays when destructive is disabled", tool: "preview_alert_rule", disableDestructive: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isToolApplicable(byName[tt.tool], tt.readOnly, tt.disableDestructive)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAllToolsSetReadOnlyHint(t *testing.T) {
	for _, group := range GroupedTools() {
		for i := range group.Tools {
			if group.Tools[i].Tool.Annotations.ReadOnlyHint == nil {
				t.Errorf("%s: nil ReadOnlyHint (read-only mode would hide the tool)", group.Tools[i].Tool.Name)
			}
		}
	}
}

func toolByNameIndex(t *testing.T, tools []api.ServerTool) map[string]api.ServerTool {
	t.Helper()
	byName := make(map[string]api.ServerTool, len(tools))
	for i := range tools {
		byName[tools[i].Tool.Name] = tools[i]
	}
	for _, name := range []string{
		"list_metrics", "get_silences", "create_silence", "update_silence", "delete_silence",
		"list_alert_rules", "list_alerts", "preview_alert_rule", "create_alert_rule", "update_alert_rule", "delete_alert_rules",
	} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("tool %q missing from GetTools", name)
		}
	}
	return byName
}
