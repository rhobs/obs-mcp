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

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/tempo"
	"github.com/rhobs/obs-mcp/pkg/tools"
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
)

func NewMCPServer(opts ObsMCPOptions) (*server.MCPServer, error) {
	hooks := &server.Hooks{}
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		slog.Debug("MCP tool call", "tool", message.Params.Name, "arguments", message.Params.Arguments)
	})
	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		slog.Debug("MCP tool result", "tool", message.Params.Name, "isError", result.IsError, "content", result.Content)
	})

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithLogging(),
		server.WithHooks(hooks),
		server.WithToolCapabilities(true),
		server.WithInstructions(tools.ServerPrompt),
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
	getAlertsTool := CreateGetAlertsTool()
	getSilencesTool := CreateGetSilencesTool()
	getCurrentTimeTool := CreateGetCurrentTimeTool()

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
	mcpServer.AddTool(getCurrentTimeTool, CurrentTimeHandler)

	k8sClient, err := k8s.GetDynamicClient()
	if err != nil {
		return err
	}

	// Workaround to break import cycle of mcp pkg imports tempo pkg (to register tools),
	// and tempo pkg imports mcp pkg (to use auth functionality).
	httpClientFactory := func(ctx context.Context) (*http.Client, error) {
		// This will be called for every MCP tool request.
		// When using "header" auth mode, it will forward the authorization header to the Tempo gateway.
		return getTempoHTTPClient(ctx, opts)
	}
	useRoute := opts.AuthMode == AuthModeKubeConfig

	tempoToolset := tempo.NewTempoToolset(k8sClient, useRoute, httpClientFactory)
	mcpServer.AddTool(tempo.ListInstancesTool(), tempoToolset.ListInstancesHandler)
	mcpServer.AddTool(tempo.GetTraceByIdTool(), tempoToolset.GetTraceByIdHandler)
	mcpServer.AddTool(tempo.SearchTracesTool(), tempoToolset.SearchTracesHandler)
	mcpServer.AddTool(tempo.SearchTagsTool(), tempoToolset.SearchTagsHandler)
	mcpServer.AddTool(tempo.SearchTagValuesTool(), tempoToolset.SearchTagValuesHandler)

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
