//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

const (
	mcpEndpoint = "/mcp"
)

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *MCPError      `json:"error,omitempty"`
}

// MCPError represents an MCP JSON-RPC error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPClient provides methods for interacting with the MCP server
type MCPClient struct {
	baseURL string
	client  *http.Client
}

// NewMCPClient creates a new MCP client with the given base URL
func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: defaultTimeout},
	}
}

// SendRequest sends an MCP request and returns the response
func (c *MCPClient) SendRequest(t *testing.T, req MCPRequest) (*MCPResponse, error) {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+mcpEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Validate HTTP status code - MCP/JSON-RPC should always return 200 OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, string(respBody))
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w (status: %d, body: %s)", err, resp.StatusCode, string(respBody))
	}

	return &mcpResp, nil
}

// CallTool is a convenience method for calling an MCP tool
func (c *MCPClient) CallTool(t *testing.T, id int, toolName string, args map[string]any) (*MCPResponse, error) {
	t.Helper()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}

	return c.SendRequest(t, req)
}
