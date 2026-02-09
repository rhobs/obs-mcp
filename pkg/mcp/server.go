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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// ObsMCPOptions contains configuration options for the MCP server
type ObsMCPOptions struct {
	AuthMode          AuthMode
	MetricsBackendURL string
	AlertmanagerURL   string
	Insecure          bool
	Guardrails        *prometheus.Guardrails
}

const (
	mcpEndpoint            = "/mcp"
	healthEndpoint         = "/health"
	serverName             = "obs-mcp"
	serverVersion          = "1.0.0"
	defaultShutdownTimeout = 10 * time.Second

	serverInstructions = `You are an expert Kubernetes and OpenShift observability assistant with direct access to Prometheus metrics and Alertmanager alerts through this MCP server.

## INVESTIGATION STARTING POINT

When the user asks about issues, errors, failures, outages, or things going wrong - consider calling get_alerts first to see what's currently firing. Alert labels provide exact identifiers (namespaces, pods, services) useful for targeted metric queries.

If the user mentions a specific alert by name, use get_alerts with a filter to retrieve its full labels before investigating further.

## MANDATORY WORKFLOW FOR QUERYING - ALWAYS FOLLOW THIS ORDER

**STEP 1: ALWAYS call list_metrics FIRST**
- This is NON-NEGOTIABLE for EVERY question
- NEVER skip this step, even if you think you know the metric name
- NEVER guess metric names - they vary between environments
- Search the returned list to find the exact metric name that exists

**STEP 2: Call get_label_names for the metric you found**
- Discover available labels for filtering (namespace, pod, service, etc.)

**STEP 3: Call get_label_values if you need specific filter values**
- Find exact label values (e.g., actual namespace names, pod names)

**STEP 4: Execute your query using the EXACT metric name from Step 1**
- Use execute_instant_query for current state questions
- Use execute_range_query for trends/historical analysis

## CRITICAL RULES

1. **NEVER query a metric without first calling list_metrics** - You must verify the metric exists
2. **Use EXACT metric names from list_metrics output** - Do not modify or guess metric names
3. **If list_metrics doesn't return a relevant metric, tell the user** - Don't fabricate queries
4. **BE PROACTIVE** - Complete all steps automatically without asking for confirmation. When you find a relevant metric, proceed to query.
5. **UNDERSTAND TIME FRAMES** - Use the start and end parameters to specify the time frame for your queries. You can use NOW for current time liberally across parameters, and NOW±duration for relative time frames.

## Query Type Selection

- **execute_instant_query**: Current values, point-in-time snapshots, "right now" questions
- **execute_range_query**: Trends over time, rate calculations, historical analysis`
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

	// Register UI resources for MCP Apps
	mcpServer.AddResource(
		mcp.Resource{
			URI:         "ui://timeseries-chart",
			Name:        "Timeseries Chart",
			Description: "Interactive timeseries line chart for range query results",
			MIMEType:    "text/html;profile=mcp-app",
		},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "ui://timeseries-chart",
					MIMEType: "text/html;profile=mcp-app",
					Text:     chartHTML,
				},
			}, nil
		},
	)

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
	getAlertsTool := CreateGetAlertsTool()
	getSilencesTool := CreateGetSilencesTool()

	// Create handlers
	listMetricsHandler := ListMetricsHandler(opts)
	executeInstantQueryHandler := ExecuteInstantQueryHandler(opts)
	executeRangeQueryHandler := ExecuteRangeQueryHandler(opts)
	getLabelNamesHandler := GetLabelNamesHandler(opts)
	getLabelValuesHandler := GetLabelValuesHandler(opts)
	getSeriesHandler := GetSeriesHandler(opts)
	getAlertsHandler := GetAlertsHandler(opts)
	getSilencesHandler := GetSilencesHandler(opts)

	// Add tools to server
	mcpServer.AddTool(listMetricsTool, listMetricsHandler)
	mcpServer.AddTool(executeInstantQueryTool, executeInstantQueryHandler)
	mcpServer.AddTool(executeRangeQueryTool, executeRangeQueryHandler)
	mcpServer.AddTool(getLabelNamesTool, getLabelNamesHandler)
	mcpServer.AddTool(getLabelValuesTool, getLabelValuesHandler)
	mcpServer.AddTool(getSeriesTool, getSeriesHandler)
	mcpServer.AddTool(getAlertsTool, getAlertsHandler)
	mcpServer.AddTool(getSilencesTool, getSilencesHandler)

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
