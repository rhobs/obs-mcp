//go:build e2e && !openshift

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListAlertRules(t *testing.T) {
	skipIfMissingTool(t, "list_alert_rules")

	resp, err := mcpClient.CallTool(t, 210, "list_alert_rules", map[string]any{})
	require.NoError(t, err)
	if errText := mcpResultErrorText(resp); errText != "" {
		skipIfAlertManagementBackendUnavailable(t, errText)
		t.Fatalf("list_alert_rules failed: %s", errText)
	}

	content := structuredContent(t, resp)
	_, ok := content["rules"].([]any)
	require.True(t, ok, "expected rules array in structuredContent")
}

func TestListAlertRulesInvalidSource(t *testing.T) {
	skipIfMissingTool(t, "list_alert_rules")

	resp, err := mcpClient.CallTool(t, 211, "list_alert_rules", map[string]any{
		"source": "invalid",
	})
	require.NoError(t, err)
	errText := mcpResultErrorText(resp)
	if errText == "" {
		t.Fatal("expected error for invalid source")
	}
	skipIfAlertManagementBackendUnavailable(t, errText)
	require.Contains(t, errText, "invalid source")
}

func TestListAlerts(t *testing.T) {
	skipIfMissingTool(t, "list_alerts")

	resp, err := mcpClient.CallTool(t, 220, "list_alerts", map[string]any{})
	require.NoError(t, err)
	if errText := mcpResultErrorText(resp); errText != "" {
		skipIfAlertManagementBackendUnavailable(t, errText)
		t.Fatalf("list_alerts failed: %s", errText)
	}

	content := structuredContent(t, resp)
	_, ok := content["alerts"].([]any)
	require.True(t, ok, "expected alerts array in structuredContent")
}

func TestListAlertsInvalidState(t *testing.T) {
	skipIfMissingTool(t, "list_alerts")

	resp, err := mcpClient.CallTool(t, 221, "list_alerts", map[string]any{
		"state": "invalid",
	})
	require.NoError(t, err)
	errText := mcpResultErrorText(resp)
	if errText == "" {
		t.Fatal("expected error for invalid state")
	}
	skipIfAlertManagementBackendUnavailable(t, errText)
	require.Contains(t, errText, "invalid state")
}
