//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLokiListInstances(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 100, "loki_list_instances", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call loki_list_instances: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("loki_list_instances returned an error result: %s", resultJSON)
	}

	structured := resp.Result["structuredContent"].(map[string]any)
	instances := structured["instances"].([]any)
	require.Len(t, instances, 1, "expected exactly one Loki instance")

	inst := instances[0].(map[string]any)
	require.Equal(t, "obs-mcp-loki", inst["lokiName"], "unexpected lokiName")
	require.Equal(t, "obs-mcp-loki", inst["lokiNamespace"], "unexpected lokiNamespace")
	require.NotEmpty(t, inst["status"], "expected non-empty status")
	require.NotEmpty(t, inst["url"], "expected non-empty url")

	t.Logf("loki_list_instances returned successfully (status=%s, url=%s)", inst["status"], inst["url"])
}

// lokiBaseArgs returns the common instance/tenant args shared by all loki tool tests.
func lokiBaseArgs() map[string]any {
	return map[string]any{
		"lokiNamespace": "obs-mcp-loki",
		"lokiName":      "obs-mcp-loki",
		"tenant":        "network",
	}
}

// callLokiTool calls the named loki tool with base args merged with extra.
func callLokiTool(t *testing.T, id int, tool string, extra map[string]any) *MCPResponse {
	t.Helper()
	base := lokiBaseArgs()
	args := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		args[k] = v
	}
	for k, v := range extra {
		args[k] = v
	}
	resp, err := mcpClient.CallTool(t, id, tool, args)
	require.NoError(t, err, "CallTool %s failed", tool)
	return resp
}

func requireLokiSuccess(t *testing.T, resp *MCPResponse) map[string]any {
	t.Helper()
	require.Nil(t, resp.Error, "unexpected MCP error")
	if isErr, ok := resp.Result["isError"].(bool); ok {
		require.False(t, isErr, "unexpected isError in result: %v", resp.Result)
	}
	return resp.Result["structuredContent"].(map[string]any)
}

func requireLokiError(t *testing.T, resp *MCPResponse) {
	t.Helper()
	if resp.Error != nil {
		return
	}
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			return
		}
	}
	t.Error("expected an error response")
}

func TestLokiLabelNames(t *testing.T) {
	// --- happy path: default time range, explicit time range, known labels ---
	t.Run("basic", func(t *testing.T) {
		structured := requireLokiSuccess(t, callLokiTool(t, 110, "loki_label_names", nil))
		labels := structured["labels"].([]any)
		require.NotEmpty(t, labels, "expected at least one label")

		// The log generator pushes with app, level, SrcK8S_Namespace, DstK8S_Namespace
		labelStrs := make([]string, len(labels))
		for i, l := range labels {
			labelStrs[i] = l.(string)
		}
		require.Contains(t, labelStrs, "app")
		require.Contains(t, labelStrs, "level")

		t.Logf("returned %d labels", len(labels))
	})

	t.Run("with_time_range", func(t *testing.T) {
		structured := requireLokiSuccess(t, callLokiTool(t, 111, "loki_label_names", map[string]any{
			"start": "NOW-1h",
			"end":   "NOW",
		}))
		labels := structured["labels"].([]any)
		require.NotEmpty(t, labels)
	})

	// --- error: start without end ---
	t.Run("start_without_end", func(t *testing.T) {
		requireLokiError(t, callLokiTool(t, 112, "loki_label_names", map[string]any{
			"start": "NOW-1h",
		}))
	})
}

func TestLokiLabelValues(t *testing.T) {
	// --- happy path: known label, time range ---
	t.Run("basic", func(t *testing.T) {
		structured := requireLokiSuccess(t, callLokiTool(t, 120, "loki_label_values", map[string]any{
			"label": "app",
		}))
		values := structured["values"].([]any)
		require.NotEmpty(t, values, "expected at least one value for label app")

		valueStrs := make([]string, len(values))
		for i, v := range values {
			valueStrs[i] = v.(string)
		}
		require.Contains(t, valueStrs, "obs-mcp-log-generator")

		t.Logf("label app has %d values", len(values))
	})

	t.Run("with_time_range", func(t *testing.T) {
		structured := requireLokiSuccess(t, callLokiTool(t, 121, "loki_label_values", map[string]any{
			"label": "level",
			"start": "NOW-1h",
			"end":   "NOW",
		}))
		values := structured["values"].([]any)
		require.NotEmpty(t, values)

		valueStrs := make([]string, len(values))
		for i, v := range values {
			valueStrs[i] = v.(string)
		}
		require.Contains(t, valueStrs, "info")
	})

	// --- error cases ---
	t.Run("missing_label", func(t *testing.T) {
		requireLokiError(t, callLokiTool(t, 122, "loki_label_values", nil))
	})

	t.Run("start_without_end", func(t *testing.T) {
		requireLokiError(t, callLokiTool(t, 123, "loki_label_values", map[string]any{
			"label": "app",
			"start": "NOW-1h",
		}))
	})
}

func TestLokiQueryRange(t *testing.T) {
	callQR := func(t *testing.T, id int, extra map[string]any) *MCPResponse {
		t.Helper()
		return callLokiTool(t, id, "loki_query_range", extra)
	}

	// --- happy path: basic query with duration, line filter, limit, direction, explicit start/end ---
	t.Run("basic", func(t *testing.T) {
		structured := requireLokiSuccess(t, callQR(t, 101, map[string]any{
			"query":    `{app="obs-mcp-log-generator"} |= "obs-mcp-loki-hack"`,
			"duration": "15m",
			"limit":    3,
		}))

		require.Equal(t, "streams", structured["resultType"])
		streams := structured["streams"].([]any)
		require.NotEmpty(t, streams, "expected at least one stream from log generator")

		stream := streams[0].(map[string]any)
		labels := stream["labels"].(map[string]any)
		require.Equal(t, "obs-mcp-log-generator", labels["app"])

		entries := stream["entries"].([]any)
		require.NotEmpty(t, entries)

		// Verify entry structure and line filter content
		for _, e := range entries {
			entry := e.(map[string]any)
			require.NotEmpty(t, entry["timestamp"])
			require.Contains(t, entry["line"], "obs-mcp-loki-hack")
		}

		// Verify limit is respected
		totalEntries := 0
		for _, s := range streams {
			totalEntries += len(s.(map[string]any)["entries"].([]any))
		}
		require.LessOrEqual(t, totalEntries, 3, "limit=3 should cap total entries")

		t.Logf("returned %d streams, %d total entries", len(streams), totalEntries)
	})

	t.Run("forward_and_explicit_time_range", func(t *testing.T) {
		structured := requireLokiSuccess(t, callQR(t, 102, map[string]any{
			"query":     `{app="obs-mcp-log-generator"}`,
			"start":     "NOW-30m",
			"end":       "NOW",
			"limit":     5,
			"direction": "forward",
		}))
		streams := structured["streams"].([]any)
		require.NotEmpty(t, streams)
	})

	t.Run("empty_result", func(t *testing.T) {
		structured := requireLokiSuccess(t, callQR(t, 103, map[string]any{
			"query":    `{app="nonexistent-app-xyz-12345"}`,
			"duration": "5m",
		}))
		if streams, ok := structured["streams"].([]any); ok {
			require.Empty(t, streams)
		}
	})

	// --- error cases ---
	t.Run("missing_query", func(t *testing.T) {
		requireLokiError(t, callQR(t, 104, nil))
	})

	t.Run("invalid_direction", func(t *testing.T) {
		requireLokiError(t, callQR(t, 105, map[string]any{
			"query":     `{app="obs-mcp-log-generator"}`,
			"direction": "invalid",
		}))
	})
}
