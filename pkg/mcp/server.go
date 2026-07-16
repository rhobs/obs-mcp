package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/tools"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

type Toolset string

const (
	ToolsetMetrics Toolset = "observability/metrics"
	ToolsetTraces  Toolset = "observability/traces"
	ToolsetLogs    Toolset = "observability/logs"
	ToolsetOtelcol Toolset = "observability/otelcol"
)

var AllToolsets = []string{string(ToolsetMetrics), string(ToolsetTraces), string(ToolsetLogs), string(ToolsetOtelcol)}

// ObsMCPOptions contains configuration options for the MCP server
type ObsMCPOptions struct {
	Toolsets               []Toolset
	AuthMode               auth.AuthMode
	MetricsBackendURL      string
	AlertmanagerURL        string
	Insecure               bool
	Guardrails             *prometheus.Guardrails
	FullRangeQueryResponse bool
	Traces                 *traces.Config
	Otelcol                *otelcol.Config
	Logs                   *logs.Config
}

const (
	mcpEndpoint            = "/mcp"
	healthEndpoint         = "/health"
	serverName             = "obs-mcp"
	serverVersion          = "1.0.0"
	defaultShutdownTimeout = 10 * time.Second
)

func NewMCPServer(opts ObsMCPOptions) (*mcp.Server, error) {
	impl := &mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}

	var instructions []string
	if slices.Contains(opts.Toolsets, ToolsetMetrics) {
		instructions = append(instructions, tools.ServerPrompt)
	}
	if slices.Contains(opts.Toolsets, ToolsetTraces) {
		instructions = append(instructions, traces.ServerPrompt)
	}
	if slices.Contains(opts.Toolsets, ToolsetLogs) {
		instructions = append(instructions, logs.ServerPrompt)
	}
	if slices.Contains(opts.Toolsets, ToolsetOtelcol) {
		instructions = append(instructions, otelcol.ServerPrompt)
	}

	serverOpts := &mcp.ServerOptions{
		Instructions: strings.Join(instructions, "\n"),
	}

	mcpServer := mcp.NewServer(impl, serverOpts)

	if err := SetupTools(mcpServer, opts); err != nil {
		return nil, err
	}

	return mcpServer, nil
}

func SetupTools(mcpServer *mcp.Server, opts ObsMCPOptions) error {
	clientCmdConfig := k8s.GetClientCmdConfig()
	restConfig, err := clientCmdConfig.ClientConfig()
	if err != nil {
		return err
	}
	mgr, err := kubernetes.NewManager(context.Background(), config.BaseDefault(), restConfig, clientCmdConfig)
	if err != nil {
		return err
	}

	if slices.Contains(opts.Toolsets, ToolsetMetrics) {
		mcp.AddTool(mcpServer, tools.ListMetrics.ToMCPTool(), ListMetricsHandler(opts))
		mcp.AddTool(mcpServer, tools.ExecuteInstantQuery.ToMCPTool(), ExecuteInstantQueryHandler(opts))
		mcp.AddTool(mcpServer, tools.ExecuteRangeQuery.ToMCPTool(), ExecuteRangeQueryHandler(opts))
		mcp.AddTool(mcpServer, tools.ShowTimeseries.ToMCPTool(), ShowTimeseriesHandler(opts))
		mcp.AddTool(mcpServer, tools.GetLabelNames.ToMCPTool(), GetLabelNamesHandler(opts))
		mcp.AddTool(mcpServer, tools.GetLabelValues.ToMCPTool(), GetLabelValuesHandler(opts))
		mcp.AddTool(mcpServer, tools.GetSeries.ToMCPTool(), GetSeriesHandler(opts))
		mcp.AddTool(mcpServer, tools.GetAlerts.ToMCPTool(), GetAlertsHandler(opts))
		mcp.AddTool(mcpServer, tools.GetSilences.ToMCPTool(), GetSilencesHandler(opts))
	}

	if slices.Contains(opts.Toolsets, ToolsetTraces) {
		if opts.Traces == nil {
			return errors.New("configuration for traces toolset is missing")
		}

		err := addToolset(mcpServer, mgr, &traces.Toolset{}, opts.Traces)
		if err != nil {
			return err
		}
	}

	if slices.Contains(opts.Toolsets, ToolsetOtelcol) {
		if opts.Otelcol == nil {
			return errors.New("configuration for otelcol toolset is missing")
		}
		err := addToolset(mcpServer, mgr, &otelcol.Toolset{}, opts.Otelcol)
		if err != nil {
			return err
		}
	}

	if slices.Contains(opts.Toolsets, ToolsetLogs) {
		if opts.Logs == nil {
			return errors.New("configuration for logs toolset is missing")
		}
		err := addToolset(mcpServer, mgr, &logs.Toolset{}, opts.Logs)
		if err != nil {
			return err
		}
	}
	return nil
}

func addToolset(mcpServer *mcp.Server, mgr *kubernetes.Manager, toolset api.Toolset, toolsetConfig api.ExtendedConfig) error {
	baseConfig := &mcpBaseConfig{toolsetConfig: toolsetConfig}
	serverTools := toolset.GetTools(nil)
	for i := range serverTools {
		goSdkTool, goSdkHandler, err := ServerToolToGoSdkTool(mgr, baseConfig, serverTools[i])
		if err != nil {
			return err
		}
		mcpServer.AddTool(goSdkTool, goSdkHandler)
	}
	return nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.ContextWithAuthFromRequest(r.Context(), r)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
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

func Serve(ctx context.Context, mcpServer *mcp.Server, listenAddr string, authMode auth.AuthMode) error {
	mux := http.NewServeMux()

	handler := loggingMiddleware(mux)
	if authMode == auth.AuthModeHeader {
		handler = authMiddleware(handler)
	}

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	opts := &mcp.StreamableHTTPOptions{
		Stateless: true,
	}

	streamableHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, opts)
	mux.Handle(mcpEndpoint, streamableHandler)
	mux.Handle("/", streamableHandler)

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
