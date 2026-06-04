//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// runOtelcolToolsetTests verifies the otelcol toolset is registered on the deployed server.
// Schema and validation logic are covered by unit tests in pkg/otelcol.
func runOtelcolToolsetTests(t *testing.T, client *MCPClient) {
	t.Helper()

	resp, err := client.CallTool(t, 47, "otelcol_get_versions", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call otelcol_get_versions: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}

	structured, ok := resp.Result["structuredContent"].(map[string]any)
	require.True(t, ok, "expected structuredContent in result")
	require.NotEmpty(t, structured["versions"])
	require.NotEmpty(t, structured["latest_version"])
}
