//go:build e2e && !openshift

package e2e

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const e2eSilenceAlertName = "obs-mcp-e2e-silence"

func TestSilenceCreateUpdateDelete(t *testing.T) {
	skipIfMissingTool(t, "create_silence")
	ensureAlertmanagerWriteRBAC(t)

	createResp, err := mcpClient.CallTool(t, 200, "create_silence", map[string]any{
		"comment":  "obs-mcp e2e silence",
		"duration": "1h",
		"labels": map[string]any{
			"alertname": e2eSilenceAlertName,
		},
	})
	require.NoError(t, err)
	if errText := mcpResultErrorText(createResp); errText != "" {
		t.Fatalf("create_silence failed: %s", errText)
	}
	silenceID, ok := structuredContent(t, createResp)["silence_id"].(string)
	require.True(t, ok, "expected silence_id string")
	require.NotEmpty(t, silenceID)

	t.Cleanup(func() {
		cleanupToolCall(t, 203, "delete_silence", map[string]any{
			"silence_id": silenceID,
		})
	})

	listResp, err := mcpClient.CallTool(t, 201, "get_silences", map[string]any{
		"filter": "alertname=" + e2eSilenceAlertName,
	})
	require.NoError(t, err)
	listJSON := mustResultJSON(t, listResp)
	require.Contains(t, listJSON, silenceID)

	updateResp, err := mcpClient.CallTool(t, 202, "update_silence", map[string]any{
		"silence_id": silenceID,
		"comment":    "obs-mcp e2e silence updated",
	})
	require.NoError(t, err)
	if errText := mcpResultErrorText(updateResp); errText != "" {
		t.Fatalf("update_silence failed: %s", errText)
	}

	deleteResp, err := mcpClient.CallTool(t, 204, "delete_silence", map[string]any{
		"silence_id": silenceID,
	})
	require.NoError(t, err)
	if errText := mcpResultErrorText(deleteResp); errText != "" {
		t.Fatalf("delete_silence failed: %s", errText)
	}
}

func TestCreateSilenceMissingComment(t *testing.T) {
	skipIfMissingTool(t, "create_silence")

	resp, err := mcpClient.CallTool(t, 205, "create_silence", map[string]any{
		"labels": map[string]any{"alertname": e2eSilenceAlertName},
	})
	require.NoError(t, err)
	errText := mcpResultErrorText(resp)
	require.NotEmpty(t, errText, "expected error for missing comment")
	require.Contains(t, strings.ToLower(errText), "comment")
}

func mustResultJSON(t *testing.T, resp *MCPResponse) string {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if errText := mcpResultErrorText(resp); errText != "" {
		t.Fatalf("tool returned isError: %s", errText)
	}
	raw, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	return string(raw)
}
