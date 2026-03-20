package tempo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

var (
	tempoStackGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}
	tempoMonolithicGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}
)

func newTempoStack(namespace, name string, tenants []string) *unstructured.Unstructured {
	auth := make([]any, 0, len(tenants))
	for _, t := range tenants {
		auth = append(auth, map[string]any{"tenantName": t})
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tempo.grafana.com",
		Version: "v1alpha1",
		Kind:    "TempoStack",
	})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	obj.Object["spec"] = map[string]any{
		"tenants": map[string]any{
			"mode":           "openshift",
			"authentication": auth,
		},
	}
	obj.Object["status"] = map[string]any{
		"conditions": []any{
			map[string]any{
				"type":   "Ready",
				"status": string(metav1.ConditionTrue),
			},
		},
	}
	return obj
}

func newMockK8sClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			tempoStackGVR:      "TempoStackList",
			tempoMonolithicGVR: "TempoMonolithicList",
		},
		objects...,
	)
}

// redirectTransport is a http.RoundTripper that redirects all requests to a
// target URL, preserving the original path and query string.
type redirectTransport struct {
	target *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to the test server, keeping path and query.
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.target.Listener.Addr().String()
	return http.DefaultTransport.RoundTrip(req)
}

// newTestMCPClient creates an in-process MCP server+client wired to the
// real SearchTracesHandler. The fake k8s dynamic client provides instance
// discovery and the redirectTransport sends Tempo HTTP calls to mockServer.
func newTestMCPClient(t *testing.T, mockServer *httptest.Server) *mcpclient.Client {
	t.Helper()

	fakeClient := newMockK8sClient(
		newTempoStack("ns1", "stack1", []string{"tenant-a", "tenant-b"}),
	)
	restCfg := &rest.Config{Transport: &redirectTransport{target: mockServer}}

	toolset := &Toolset{}
	cfg := &Config{UseRoute: false}
	mcpServer := server.NewMCPServer("test-tempo", "1.0.0",
		server.WithToolCapabilities(true),
	)
	mcpServer.AddTool(
		SearchTracesTool.ToMCPTool(),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result := toolset.SearchTracesHandler(ToolParams{
				context:       ctx,
				arguments:     request.GetArguments(),
				dynamicClient: fakeClient,
				restConfig:    restCfg,
				config:        cfg,
			})
			return result.ToMCPResult()
		},
	)

	client, err := mcpclient.NewInProcessClient(mcpServer)
	require.NoError(t, err)
	require.NoError(t, client.Start(t.Context()))
	t.Cleanup(func() { _ = client.Close() })

	_, err = client.Initialize(t.Context(), mcp.InitializeRequest{})
	require.NoError(t, err)

	return client
}

func TestSearchTraces(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, `{status=error}`, r.URL.Query().Get("q"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))
		require.Equal(t, "2", r.URL.Query().Get("spss"))
		require.NotEmpty(t, r.URL.Query().Get("start"))
		require.NotEmpty(t, r.URL.Query().Get("end"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"traces":[]}`))
	}))
	defer mockServer.Close()

	c := newTestMCPClient(t, mockServer)
	result, err := c.CallTool(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "tempo_search_traces",
			Arguments: map[string]any{
				"tempoNamespace": "ns1",
				"tempoName":      "stack1",
				"tenant":         "tenant-a",
				"query":          `{status=error}`,
				"limit":          int(5),
				"spss":           int(2),
				"start":          "2024-01-01T00:00:00Z",
				"end":            "2024-01-01T01:00:00Z",
			},
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected error: %v", result.Content)
	require.Equal(t, map[string]any{"traces": []any{}}, result.StructuredContent)
}

func TestSearchTraces_InstanceNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("tempo server should not be called for unknown instance")
	}))
	defer mockServer.Close()

	c := newTestMCPClient(t, mockServer)
	result, err := c.CallTool(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "tempo_search_traces",
			Arguments: map[string]any{
				"tempoNamespace": "ns1",
				"tempoName":      "nonexistent",
				"tenant":         "tenant-a",
				"query":          "{}",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, text.Text, "not found")
}

func TestSearchTraces_TenantNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("tempo server should not be called for unknown tenant")
	}))
	defer mockServer.Close()

	c := newTestMCPClient(t, mockServer)
	result, err := c.CallTool(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "tempo_search_traces",
			Arguments: map[string]any{
				"tempoNamespace": "ns1",
				"tempoName":      "stack1",
				"tenant":         "nonexistent",
				"query":          "{}",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, text.Text, "does not exist")
}
