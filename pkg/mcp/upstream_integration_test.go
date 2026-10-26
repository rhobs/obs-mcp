package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/obs/auth"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/logs"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/metrics"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/otelcol"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/traces"
	"github.com/google/jsonschema-go/jsonschema"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func standaloneTestOptions() ObsMCPOptions {
	return ObsMCPOptions{
		Metrics: &metrics.Config{AuthMode: auth.AuthModeKubeConfig},
		Logs:    &logs.Config{AuthMode: auth.AuthModeKubeConfig},
		Traces:  &traces.Config{AuthMode: auth.AuthModeKubeConfig},
		Otelcol: otelcol.NewDefaultConfig(),
		KubernetesClientConfig: clientcmd.NewDefaultClientConfig(clientcmdapi.Config{
			Clusters:       map[string]*clientcmdapi.Cluster{"test": {Server: "https://localhost", InsecureSkipTLSVerify: true}},
			AuthInfos:      map[string]*clientcmdapi.AuthInfo{"test": {Token: "kubeconfig-token"}},
			Contexts:       map[string]*clientcmdapi.Context{"test": {Cluster: "test", AuthInfo: "test"}},
			CurrentContext: "test",
		}, &clientcmd.ConfigOverrides{}),
	}
}

// Importing upstream toolsets must not expose unrelated Kubernetes tools or
// filter out Loki/Tempo tools just because the standalone inspector has no CRDs.
func TestUpstreamToolsetRegistration(t *testing.T) {
	toolNames := map[string][]string{
		metrics.ToolsetName: {"list_metrics", "execute_instant_query", "execute_range_query", "show_timeseries", "get_label_names", "get_label_values", "get_series", "get_alerts", "get_silences"},
		logs.ToolsetName:    {"loki_list_instances", "loki_label_names", "loki_label_values", "loki_query_range"},
		traces.ToolsetName:  {"tempo_list_instances", "tempo_get_trace_by_id", "tempo_search_traces", "tempo_search_tags", "tempo_search_tag_values"},
		otelcol.ToolsetName: {"otelcol_list_components", "otelcol_get_component_schema", "otelcol_validate_config", "otelcol_get_versions"},
	}
	prompts := map[string]string{
		metrics.ToolsetName: metrics.ServerPrompt,
		logs.ToolsetName:    logs.ServerPrompt,
		traces.ToolsetName:  traces.ServerPrompt,
		otelcol.ToolsetName: otelcol.ServerPrompt,
	}
	var allNames []string
	for _, name := range AllToolsets {
		allNames = append(allNames, toolNames[name]...)
	}
	var documented []string
	// Snapshot definitions before registration can mutate schema pointers.
	// Compare whole schemas so required fields, patterns, and nested properties
	// are checked across the standalone adapter and MCP wire serialization.
	expectedSchemas := make(map[string]map[string]json.RawMessage)
	for _, group := range GroupedTools() {
		for _, tool := range group.Tools {
			documented = append(documented, tool.Tool.Name)
			expectedSchemas[tool.Tool.Name] = make(map[string]json.RawMessage)
			require.NotNil(t, tool.Tool.InputSchema)
			input := *tool.Tool.InputSchema
			// The adapter deliberately adds empty input properties for OpenAI
			// compatibility; output schemas must remain untouched.
			if input.Properties == nil {
				input.Properties = make(map[string]*jsonschema.Schema)
			}
			for name, schema := range map[string]any{"input": &input, "output": tool.Tool.OutputSchema} {
				data, err := json.Marshal(schema)
				require.NoError(t, err)
				expectedSchemas[tool.Tool.Name][name] = data
			}
		}
	}
	require.ElementsMatch(t, allNames, documented)

	for _, enabled := range [][]string{nil, {metrics.ToolsetName}, {logs.ToolsetName}, {traces.ToolsetName}, {otelcol.ToolsetName}, AllToolsets} {
		t.Run(strings.Join(enabled, ","), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			opts := standaloneTestOptions()
			opts.Toolsets = enabled
			server, err := NewMCPServer(opts)
			require.NoError(t, err)
			clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
			serverSession, err := server.Connect(ctx, serverTransport, nil)
			require.NoError(t, err)
			defer serverSession.Close()
			client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client"}, nil)
			session, err := client.Connect(ctx, clientTransport, nil)
			require.NoError(t, err)
			defer session.Close()

			listed, err := session.ListTools(ctx, nil)
			require.NoError(t, err)
			var names, expected, instructions []string
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)
				require.NotEmpty(t, tool.Description, "tool %s must have a description", tool.Name)
				for name, schema := range map[string]any{"input": tool.InputSchema, "output": tool.OutputSchema} {
					data, err := json.Marshal(schema)
					require.NoError(t, err)
					require.JSONEq(t, string(expectedSchemas[tool.Name][name]), string(data),
						"tool %s: %s schema changed through the standalone adapter", tool.Name, name)
				}
			}
			for _, name := range enabled {
				expected = append(expected, toolNames[name]...)
			}
			// Match the existing standalone instruction ordering.
			for _, name := range []string{metrics.ToolsetName, traces.ToolsetName, logs.ToolsetName, otelcol.ToolsetName} {
				if slices.Contains(enabled, name) {
					instructions = append(instructions, prompts[name])
				}
			}
			require.ElementsMatch(t, expected, names)
			require.Equal(t, strings.Join(instructions, "\n"), session.InitializeResult().Instructions)
		})
	}
}

// Exercise the real HTTP host and upstream handlers, not just a synthetic
// toolcfg consumer. Explicit backend URLs, TLS options, guardrails, and schema
// files must reach the handlers, and callers must not inherit kubeconfig auth.
func TestUpstreamConfigurationOverHTTP(t *testing.T) {
	requests := make(chan string, 10)
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer caller-token" {
			t.Errorf("backend Authorization = %q, want caller token", got)
		}
		requests <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/prometheus/api/v1/label/__name__/values":
			_, _ = io.WriteString(w, `{"status":"success","data":["up"]}`)
		case "/prometheus/api/v1/query":
			_, _ = io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up"},"value":[1700000000,"1"]}]}}`)
		case "/api/v2/alerts":
			_, _ = io.WriteString(w, `[]`)
		case "/loki/loki/api/v1/labels":
			_, _ = io.WriteString(w, `{"status":"success","data":["service_name"]}`)
		case "/tempo/api/v2/search/tags":
			_, _ = io.WriteString(w, `{"scopes":[{"name":"resource","tags":["service.name"]}]}`)
		default:
			t.Errorf("unexpected backend request: %s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer backend.Close()

	opts := standaloneTestOptions()
	opts.Toolsets = AllToolsets
	opts.Metrics = &metrics.Config{
		AuthMode: auth.AuthModeHeader, Insecure: true, Guardrails: "none",
		PrometheusURL: backend.URL + "/prometheus", AlertmanagerURL: backend.URL,
	}
	opts.Logs = &logs.Config{AuthMode: auth.AuthModeHeader, Insecure: true, LokiURL: backend.URL + "/loki"}
	opts.Traces = &traces.Config{AuthMode: auth.AuthModeHeader, Insecure: true, TempoURL: backend.URL + "/tempo"}
	opts.Otelcol = &otelcol.Config{SchemaFS: fstest.MapFS{
		"schemas/0.999.0/receivers/test.json": {Data: []byte(`{}`)},
	}}
	registry := prom.NewRegistry()
	opts.Registry = registry
	server, err := NewMCPServer(opts)
	require.NoError(t, err)
	httpServer, _ := NewHTTPServer(server, "", registry, auth.AuthModeHeader)
	host := httptest.NewServer(httpServer.Handler)
	defer host.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// tools/call remains a protocol error without a caller token, despite a
	// valid token being present in the host's kubeconfig.
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "unauthenticated-client"}, nil)
	unauthenticated, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: host.URL + "/mcp"}, nil)
	require.NoError(t, err)
	defer unauthenticated.Close()
	_, err = unauthenticated.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_alerts"})
	require.ErrorContains(t, err, "oauth token required")
	require.Empty(t, requests)

	// Set the bearer header on every transport request, as an MCP caller does.
	client = mcpsdk.NewClient(&mcpsdk.Implementation{Name: "authenticated-client"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{
		Endpoint:   host.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport}},
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	for _, tc := range []struct {
		name  string
		args  map[string]any
		paths []string
		want  string
	}{
		{"execute_instant_query", map[string]any{"query": "up"}, []string{"/prometheus/api/v1/label/__name__/values", "/prometheus/api/v1/query"}, `"resultType":"vector"`},
		{"get_alerts", nil, []string{"/api/v2/alerts"}, `"alerts":[]`},
		{"loki_label_names", nil, []string{"/loki/loki/api/v1/labels"}, `"labels":["service_name"]`},
		{"tempo_search_tags", map[string]any{"tempoNamespace": "unused", "tempoName": "unused"}, []string{"/tempo/api/v2/search/tags"}, `"tags":["service.name"]`},
		{"otelcol_get_versions", nil, nil, `"latest_version":"0.999.0"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: tc.name, Arguments: tc.args})
			require.NoError(t, err)
			require.False(t, result.IsError, "result: %+v", result)
			structured, err := json.Marshal(result.StructuredContent)
			require.NoError(t, err)
			require.Contains(t, string(structured), tc.want)
			require.NotEmpty(t, result.Content)
			text, ok := result.Content[0].(*mcpsdk.TextContent)
			require.True(t, ok)
			require.JSONEq(t, string(structured), text.Text)
			for _, expectedPath := range tc.paths {
				select {
				case path := <-requests:
					require.Equal(t, expectedPath, path)
				case <-ctx.Done():
					t.Fatal("backend request not received")
				}
			}
		})
	}

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "execute_instant_query", Arguments: map[string]any{"query": ""}})
	require.NoError(t, err, "handler validation errors remain tool-result errors")
	require.True(t, result.IsError)
	require.Nil(t, result.StructuredContent)
	require.Contains(t, result.Content[0].(*mcpsdk.TextContent).Text, "query parameter is required")
	require.Empty(t, requests)

	for _, name := range []string{"http_requests_total", "mcp_tool_calls_total", "http_client_request_total"} {
		metric := getMetric(t, registry, name, nil)
		require.NotNil(t, metric, "upstream instrumentation missing %s", name)
		require.Positive(t, metric.GetCounter().GetValue())
	}
}

type bearerTransport struct {
	base http.RoundTripper
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer caller-token")
	return b.base.RoundTrip(r)
}
