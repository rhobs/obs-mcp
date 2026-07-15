package korrel8r

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInstructions = "Test instructions for korrel8r proxy."

// newMockUpstream creates a mock korrel8r MCP server with a few tools.
func newMockUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	server := mcp.NewServer(
		&mcp.Implementation{Name: "mock-korrel8r", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: testInstructions},
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_domains",
		Description: "List available domains.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return nil, map[string]any{"domains": []map[string]any{{"name": "mock"}}}, nil
	})

	type domainInput struct {
		Domain string `json:"domain"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_domain_classes",
		Description: "List classes in a domain.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input domainInput) (*mcp.CallToolResult, any, error) {
		if input.Domain != "mock" {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "unknown domain"}}}, nil, nil
		}
		return nil, map[string]any{"domain": "mock", "classes": []string{"a", "b"}}, nil
	})

	type helpInput struct {
		Domain string `json:"domain,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "help",
		Description: "Get help about domains.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input helpInput) (*mcp.CallToolResult, any, error) {
		return nil, map[string]any{"documentation": "mock help for " + input.Domain}, nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newProxyClient sets up a proxy to the mock upstream and returns a client session.
func newProxyClient(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	upstream := newMockUpstream(t)
	proxy, err := Connect(ctx, &Config{Endpoint: upstream.URL})
	require.NoError(t, err)
	t.Cleanup(proxy.Close)

	proxyServer := mcp.NewServer(
		&mcp.Implementation{Name: "test-proxy", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: proxy.Instructions()},
	)
	proxy.AddTools(proxyServer)

	ct, st := mcp.NewInMemoryTransports()
	ss, err := proxyServer.Connect(ctx, st, nil)
	require.NoError(t, err)
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close(); _ = ss.Wait() })
	return cs
}

func TestProxyListTools(t *testing.T) {
	cs := newProxyClient(t)
	tools, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	assert.ElementsMatch(t, names, []string{"list_domains", "list_domain_classes", "help"})
}

func TestProxyListDomains(t *testing.T) {
	cs := newProxyClient(t)
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_domains"})
	require.NoError(t, err)
	require.False(t, r.IsError)
	got := r.StructuredContent.(map[string]any)
	domains := got["domains"].([]any)
	require.Len(t, domains, 1)
	assert.Equal(t, "mock", domains[0].(map[string]any)["name"])
}

func TestProxyListDomainClasses(t *testing.T) {
	cs := newProxyClient(t)
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_domain_classes",
		Arguments: map[string]any{"domain": "mock"},
	})
	require.NoError(t, err)
	require.False(t, r.IsError)
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, "mock", got["domain"])
	assert.Equal(t, []any{"a", "b"}, got["classes"])
}

func TestProxyToolError(t *testing.T) {
	cs := newProxyClient(t)
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_domain_classes",
		Arguments: map[string]any{"domain": "nosuch"},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
}

func TestProxyInstructions(t *testing.T) {
	cs := newProxyClient(t)
	assert.Equal(t, testInstructions, cs.InitializeResult().Instructions)
}

func TestProxyToolDescriptionsAndSchemas(t *testing.T) {
	ctx := context.Background()

	// Connect directly to the upstream.
	upstream := newMockUpstream(t)
	directClient := mcp.NewClient(&mcp.Implementation{Name: "direct"}, nil)
	directCS, err := directClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: upstream.URL, DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = directCS.Close() })
	directTools, err := directCS.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)

	// Connect via proxy.
	proxy, err := Connect(ctx, &Config{Endpoint: upstream.URL})
	require.NoError(t, err)
	t.Cleanup(proxy.Close)
	proxyServer := mcp.NewServer(
		&mcp.Implementation{Name: "test-proxy", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: proxy.Instructions()},
	)
	proxy.AddTools(proxyServer)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := proxyServer.Connect(ctx, st, nil)
	require.NoError(t, err)
	proxyClient := mcp.NewClient(&mcp.Implementation{Name: "proxy-client"}, nil)
	proxyCS, err := proxyClient.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = proxyCS.Close(); _ = ss.Wait() })
	proxyTools, err := proxyCS.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)

	require.Equal(t, len(directTools.Tools), len(proxyTools.Tools))

	directByName := map[string]*mcp.Tool{}
	for _, tool := range directTools.Tools {
		directByName[tool.Name] = tool
	}
	proxyByName := map[string]*mcp.Tool{}
	for _, tool := range proxyTools.Tools {
		proxyByName[tool.Name] = tool
	}

	for name, direct := range directByName {
		proxied, ok := proxyByName[name]
		require.True(t, ok, "proxy missing tool %s", name)
		assert.Equal(t, direct.Description, proxied.Description, "description mismatch for %s", name)

		directSchema, err := json.Marshal(direct.InputSchema)
		require.NoError(t, err)
		proxySchema, err := json.Marshal(proxied.InputSchema)
		require.NoError(t, err)
		assert.JSONEq(t, string(directSchema), string(proxySchema), "input schema mismatch for %s", name)

		if direct.OutputSchema != nil {
			require.NotNil(t, proxied.OutputSchema, "proxy missing output schema for %s", name)
			directOut, _ := json.Marshal(direct.OutputSchema)
			proxyOut, _ := json.Marshal(proxied.OutputSchema)
			assert.JSONEq(t, string(directOut), string(proxyOut), "output schema mismatch for %s", name)
		}
	}
}
