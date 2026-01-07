package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// ObsMCPOptions contains configuration options for the MCP server
type ObsMCPOptions struct {
	AuthMode   AuthMode
	PromURL    string
	Insecure   bool
	Guardrails *prometheus.Guardrails
}

const (
	mcpEndpoint            = "/mcp"
	healthEndpoint         = "/health"
	serverName             = "obs-mcp"
	serverVersion          = "1.0.0"
	defaultShutdownTimeout = 10 * time.Second

	serverInstructions = `You are an expert Kubernetes and OpenShift observability assistant with direct access to Prometheus metrics through this MCP server. Your role is to help users understand their system's health, performance, and behavior by querying and analyzing metrics.

## Available Tools

1. **list_metrics** - Discover all available metric names in Prometheus. Start here for ANY observability question.

2. **get_label_names** - Find available labels (dimensions) for filtering metrics by service, namespace, pod, etc.

3. **get_label_values** - Get possible values for a label to construct accurate filters.

4. **get_series** - Preview time series matching a selector and check cardinality before querying.

5. **execute_instant_query** - Run PromQL for current/point-in-time values ("What is the error rate RIGHT NOW?").

6. **execute_range_query** - Run PromQL over a time range for trends and historical analysis.

## Standard Workflow

For questions like "Why is my API service having high error rates?":

1. **Discover metrics**: Call list_metrics to find relevant metrics (http_requests_total, errors_total, etc.)
2. **Find label dimensions**: Call get_label_names to see how to filter (by service, status, namespace)
3. **Get exact values**: Use get_label_values to find the exact service name (e.g., "api-gateway" vs "api")
4. **Verify cardinality**: Optionally, use get_series to check your filters before querying:
   - Using regex label matchers (status=~"5..", pod=~"api.*")
   - Querying container/pod metrics without namespace filter
   - First time querying a metric you haven't seen before
   - User asks about "all pods", "all services", or broad scope
5. **Query metrics**: Execute instant or range queries with proper PromQL

## Key PromQL Patterns

- **Error rate**: (rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])) * 100
- **P95 latency**: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
- **Pod memory**: sum(container_memory_working_set_bytes{container!="",container!="POD"}) by (pod)
- **CPU usage**: sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod)
- **Pod restarts**: increase(kube_pod_container_status_restarts_total[1h])

## Important Notes

- Always use rate() or increase() for counter metrics
- For container metrics, filter container!="" and container!="POD" to avoid double-counting
- Choose appropriate time ranges: 5m for current state, 1h-6h for recent trends, 24h+ for patterns
- When cardinality is high (>1000 series), add more label filters or aggregate with sum/avg by()`
)

func NewMCPServer(opts ObsMCPOptions) (*server.MCPServer, error) {
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithLogging(),
		server.WithToolCapabilities(true),
		server.WithInstructions(serverInstructions),
	)

	if err := SetupTools(mcpServer, opts); err != nil {
		return nil, err
	}

	return mcpServer, nil
}

func SetupTools(mcpServer *server.MCPServer, opts ObsMCPOptions) error {
	// Create tool definitions
	listMetricsTool := CreateListMetricsTool()
	executeInstantQueryTool := CreateExecuteInstantQueryTool()
	executeRangeQueryTool := CreateExecuteRangeQueryTool()
	getLabelNamesTool := CreateGetLabelNamesTool()
	getLabelValuesTool := CreateGetLabelValuesTool()
	getSeriesTool := CreateGetSeriesTool()

	// Create handlers
	listMetricsHandler := ListMetricsHandler(opts)
	executeInstantQueryHandler := ExecuteInstantQueryHandler(opts)
	executeRangeQueryHandler := ExecuteRangeQueryHandler(opts)
	getLabelNamesHandler := GetLabelNamesHandler(opts)
	getLabelValuesHandler := GetLabelValuesHandler(opts)
	getSeriesHandler := GetSeriesHandler(opts)

	// Add tools to server
	mcpServer.AddTool(listMetricsTool, listMetricsHandler)
	mcpServer.AddTool(executeInstantQueryTool, executeInstantQueryHandler)
	mcpServer.AddTool(executeRangeQueryTool, executeRangeQueryHandler)
	mcpServer.AddTool(getLabelNamesTool, getLabelNamesHandler)
	mcpServer.AddTool(getLabelValuesTool, getLabelValuesHandler)
	mcpServer.AddTool(getSeriesTool, getSeriesHandler)

	return nil
}

func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	authHeaderValue := r.Header.Get(string(AuthHeaderKey))
	token, found := strings.CutPrefix(authHeaderValue, "Bearer ")
	if !found {
		return ctx
	}
	return context.WithValue(ctx, AuthHeaderKey, token)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Incoming request", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		slog.Debug("Request headers", "headers", r.Header)
		if r.ContentLength > 0 {
			slog.Info("Request content length", "content_length", r.ContentLength)
		}
		next.ServeHTTP(w, r)
	})
}

func Serve(ctx context.Context, mcpServer *server.MCPServer, listenAddr string) error {
	mux := http.NewServeMux()

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: loggingMiddleware(mux),
	}

	streamableHTTPServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithStreamableHTTPServer(httpServer),
		server.WithStateLess(true),
		server.WithHTTPContextFunc(authFromRequest),
	)
	mux.Handle(mcpEndpoint, streamableHTTPServer)

	mux.Handle("/", streamableHTTPServer)

	mux.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP server starting", "listen_addr", listenAddr, "mcp_endpoint", mcpEndpoint)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		slog.Warn("Received signal, initiating graceful shutdown", "signal", sig)
		cancel()
	case <-ctx.Done():
		slog.Warn("Context cancelled, initiating graceful shutdown")
	case err := <-serverErr:
		slog.Error("HTTP server error", "error", err)
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer shutdownCancel()

	slog.Info("Shutting down HTTP server gracefully")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
		return err
	}

	slog.Info("HTTP server shutdown complete")
	return nil
}
