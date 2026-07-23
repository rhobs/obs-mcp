package korrel8r

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	korrel8rmcp "github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const basePath = "/api/v1alpha1"

func korrel8rServer(t *testing.T, responses map[string]mockResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if resp, ok := responses[key]; ok {
			if resp.status != 0 {
				http.Error(w, resp.body, resp.status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, resp.body)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

type mockResponse struct {
	body   string
	status int
}

func callTool(t *testing.T, srv *httptest.Server, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	client := korrel8rmcp.NewClient(srv.URL, srv.Client())
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	AddTools(mcpServer, client)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	_, err := mcpServer.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })

	return session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
}

// --- list_domains ---

func TestListDomains_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/domains": {body: `[{"name":"k8s","description":"Kubernetes resources"},{"name":"log","description":"Log records"}]`},
	})
	result, err := callTool(t, srv, "list_domains", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError, "result: %#v", *result)
	text := textContent(t, result)
	assert.Contains(t, text, "k8s")
	assert.Contains(t, text, "log")
}

func TestListDomains_BackendError(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/domains": {body: "server error", status: 500},
	})
	result, err := callTool(t, srv, "list_domains", nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- list_domain_classes ---

func TestListDomainClasses_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/domain/k8s/classes": {body: `["Pod","Deployment","Service"]`},
	})
	result, err := callTool(t, srv, "list_domain_classes", map[string]any{"domain": "k8s"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "Pod")
}

// --- help ---

func TestHelp_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/help": {body: `{"documentation":"Query syntax: domain:class:selector"}`},
	})
	result, err := callTool(t, srv, "help", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "Query syntax")
}

func TestHelp_WithDomain(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/help/k8s": {body: `{"documentation":"k8s domain help"}`},
	})
	result, err := callTool(t, srv, "help", map[string]any{"domain": "k8s"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "k8s domain help")
}

// --- create_neighbors_graph ---

func TestCreateNeighborsGraph_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"POST " + basePath + "/graphs/neighbors": {body: `{"nodes":[],"edges":[]}`},
	})
	result, err := callTool(t, srv, "create_neighbors_graph", map[string]any{
		"depth": 2,
		"start": map[string]any{
			"queries": []string{`k8s:Pod:{"namespace":"default","name":"web-0"}`},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestCreateNeighborsGraph_BackendError(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"POST " + basePath + "/graphs/neighbors": {body: "correlation failed", status: 500},
	})
	result, err := callTool(t, srv, "create_neighbors_graph", map[string]any{
		"depth": 1,
		"start": map[string]any{"queries": []string{"k8s:Pod:{}"}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- create_goals_graph ---

func TestCreateGoalsGraph_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"POST " + basePath + "/graphs/goals": {body: `{"nodes":[{"class":"log:application","queries":[{"query":"log:application:{}","count":1}]}],"edges":[]}`},
	})
	result, err := callTool(t, srv, "create_goals_graph", map[string]any{
		"goals": []string{"log:application"},
		"start": map[string]any{
			"queries": []string{`k8s:Pod:{"namespace":"myapp","name":"web-0"}`},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "log:application")
}

// --- get_objects ---

func TestGetObjects_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/objects": {body: `[{"kind":"Pod","metadata":{"name":"web-0"}}]`},
	})
	result, err := callTool(t, srv, "get_objects", map[string]any{
		"query": `k8s:Pod:{"namespace":"default"}`,
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "web-0")
}

func TestGetObjects_BackendError(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/objects": {body: "query failed", status: 500},
	})
	result, err := callTool(t, srv, "get_objects", map[string]any{
		"query": "k8s:Pod:{}",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- get_console ---

func TestGetConsole_Success(t *testing.T) {
	srv := korrel8rServer(t, map[string]mockResponse{
		"GET " + basePath + "/console": {body: `{"view":"k8s:Pod:{}"}`},
	})
	result, err := callTool(t, srv, "get_console", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := textContent(t, result)
	assert.Contains(t, text, "k8s:Pod")
}

// --- show_in_console ---

func TestShowInConsole_Success(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && r.URL.Path == basePath+"/console/events" {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	result, err := callTool(t, srv, "show_in_console", map[string]any{
		"view": "k8s:Pod:{}",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, gotBody, "k8s:Pod")
}

// --- NewClient validation ---

func TestNewClient_EmptyURL(t *testing.T) {
	_, err := NewClient(&Config{Korrel8rURL: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "korrel8r URL is required")
}

// textContent extracts text from the first TextContent block of a CallToolResult.
func textContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no TextContent in result")
	return ""
}
