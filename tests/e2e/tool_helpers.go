//go:build e2e

package e2e

import (
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	e2eRBACRetryLimit    = 8
	e2eRBACRetryInterval = time.Second
)

func skipIfMissingTool(t *testing.T, name string) {
	t.Helper()
	names, err := mcpClient.ListToolNames(t)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if !slices.Contains(names, name) {
		t.Skipf("tool %s is not registered", name)
	}
}

func mcpResultErrorText(resp *MCPResponse) string {
	if resp == nil {
		return ""
	}
	if resp.Error != nil {
		return resp.Error.Message
	}
	if resp.Result == nil {
		return ""
	}
	isErr, _ := resp.Result["isError"].(bool)
	if !isErr {
		return ""
	}
	content, _ := resp.Result["content"].([]any)
	var b strings.Builder
	for _, item := range content {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, _ := entry["text"].(string)
		b.WriteString(text)
	}
	return b.String()
}

func skipIfAlertManagementBackendUnavailable(t *testing.T, errText string) {
	t.Helper()
	if alertManagementBackendUnavailable(errText) {
		t.Skipf("alert management API unavailable: %s", errText)
	}
}

// alertManagementBackendUnavailable reports whether errText is a transport or
// missing-API failure (stock monitoring-plugin returns "404 page not found").
// Generic HTTP 404, TLS errors, and 403 are not treated as unavailable.
func alertManagementBackendUnavailable(errText string) bool {
	lower := strings.ToLower(errText)
	needles := []string{
		"not configured",
		"connection refused",
		"no such host",
		"dial tcp",
		"i/o timeout",
		"no route to host",
		"server misbehaving",
	}
	if slices.ContainsFunc(needles, func(needle string) bool {
		return strings.Contains(lower, needle)
	}) {
		return true
	}
	return strings.Contains(lower, "404 page not found")
}

func isForbiddenText(errText string) bool {
	lower := strings.ToLower(errText)
	return strings.Contains(errText, "403") || strings.Contains(lower, "forbidden")
}

func isNotFoundText(errText string) bool {
	lower := strings.ToLower(errText)
	return strings.Contains(errText, "404") || strings.Contains(lower, "not found")
}

// cleanupToolCall best-effort expires a leftover write. Not-found is OK.
func cleanupToolCall(t *testing.T, id int, name string, args map[string]any) {
	t.Helper()
	cleanupMCP(t, id, name, args, true)
}

// cleanupRestore reports any tool error, including 404 (drop may still be in place).
func cleanupRestore(t *testing.T, id int, name string, args map[string]any) {
	t.Helper()
	cleanupMCP(t, id, name, args, false)
}

func cleanupMCP(t *testing.T, id int, name string, args map[string]any, allowNotFound bool) {
	t.Helper()
	resp, err := mcpClient.CallTool(t, id, name, args)
	if err != nil {
		t.Errorf("cleanup %s: %v", name, err)
		return
	}
	if errText := mcpResultErrorText(resp); errText != "" {
		if allowNotFound && isNotFoundText(errText) {
			return
		}
		t.Errorf("cleanup %s: %s", name, errText)
	}
}

func retryWhileForbidden(t *testing.T, op string, call func() (map[string]any, string)) map[string]any {
	t.Helper()
	var lastErr string
	for range e2eRBACRetryLimit {
		content, errText := call()
		if errText == "" {
			return content
		}
		if isForbiddenText(errText) {
			lastErr = errText
			time.Sleep(e2eRBACRetryInterval)
			continue
		}
		t.Fatalf("%s failed: %s", op, errText)
	}
	t.Fatalf("%s still forbidden after RBAC retries: %s", op, lastErr)
	return nil
}

func structuredContent(t *testing.T, resp *MCPResponse) map[string]any {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected result, got nil")
	}
	if errText := mcpResultErrorText(resp); errText != "" {
		t.Fatalf("tool returned isError: %s", errText)
	}
	content, ok := resp.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent, got %v", resp.Result)
	}
	return content
}
