//go:build e2e

package e2e

import (
	"slices"
	"testing"
)

func TestWriteToolsRegistered(t *testing.T) {
	names, err := mcpClient.ListToolNames(t)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if slices.Contains(names, "get_silences") && !slices.Contains(names, "create_silence") {
		t.Fatal("create_silence missing while get_silences is registered: e2e deploy must set read-only=false")
	}
	if slices.Contains(names, "list_alert_rules") && !slices.Contains(names, "create_alert_rule") {
		t.Fatal("create_alert_rule missing while list_alert_rules is registered: e2e deploy must set read-only=false")
	}
}
