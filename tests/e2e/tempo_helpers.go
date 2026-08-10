//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const (
	tempoRetryTimeout  = 60 * time.Second
	tempoRetryInterval = 5 * time.Second
)

// isTransientBackendError reports whether a tool error result is likely a
// temporary connectivity failure against the Tempo backend.
func isTransientBackendError(resultJSON []byte) bool {
	s := string(resultJSON)
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "temporary failure in name resolution") ||
		strings.Contains(s, "TLS handshake timeout")
}

// callTempoTool calls a Tempo tool and retries on transient backend connectivity
// errors. Setup waits for Tempo, but query-frontend can briefly refuse connections
// after the CR reports Ready.
func callTempoTool(t *testing.T, id int, toolName string, args map[string]any) *MCPResponse {
	t.Helper()

	deadline := time.Now().Add(tempoRetryTimeout)
	var lastResultJSON []byte
	attempt := 0

	for {
		attempt++
		resp, err := mcpClient.CallTool(t, id, toolName, args)
		if err != nil {
			t.Fatalf("Failed to call %s: %v", toolName, err)
		}
		if resp.Error != nil {
			t.Fatalf("MCP error from %s: %s", toolName, resp.Error.Message)
		}
		if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
			lastResultJSON, _ = json.Marshal(resp.Result)
			if isTransientBackendError(lastResultJSON) && time.Now().Before(deadline) {
				t.Logf("%s attempt %d: transient backend error, retrying in %s: %s",
					toolName, attempt, tempoRetryInterval, lastResultJSON)
				time.Sleep(tempoRetryInterval)
				continue
			}
			t.Fatalf("%s returned an error result: %s", toolName, lastResultJSON)
		}
		return resp
	}
}
