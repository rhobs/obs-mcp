package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/metrics"
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
	Registry               prom.Registerer
	clientMetrics          *metrics.ClientMetrics
	toolMetrics            *metrics.ToolMetrics
}

const (
	mcpEndpoint            = "/mcp"
	healthEndpoint         = "/health"
	serverName             = "obs-mcp"
	serverVersion          = "1.0.0"
	defaultShutdownTimeout = 10 * time.Second
)

func NewMCPServer(opts ObsMCPOptions) (*mcp.Server, error) {
	// Initialize shared HTTP client metrics once
	if opts.Registry != nil && opts.clientMetrics == nil {
		opts.clientMetrics = metrics.NewClientMetrics(opts.Registry)
	}

	if opts.Registry != nil && opts.toolMetrics == nil {
		opts.toolMetrics = metrics.NewToolMetrics(opts.Registry)
	}

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

func needsKubernetes(toolsets []Toolset) bool {
	for _, ts := range toolsets {
		if ts == ToolsetTraces || ts == ToolsetLogs {
			return true
		}
	}
	return false
}

func SetupTools(mcpServer *mcp.Server, opts ObsMCPOptions) error {
	var mgr *kubernetes.Manager
	if needsKubernetes(opts.Toolsets) {
		clientCmdConfig := k8s.GetClientCmdConfig()
		restConfig, err := clientCmdConfig.ClientConfig()
		if err != nil {
			return err
		}
		var mgrErr error
		mgr, mgrErr = kubernetes.NewManager(context.Background(), config.BaseDefault(), restConfig, clientCmdConfig)
		if mgrErr != nil {
			return mgrErr
		}
	}

	if slices.Contains(opts.Toolsets, ToolsetMetrics) {
		mcp.AddTool(mcpServer, tools.ListMetrics.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.ListMetrics.Name, opts.toolMetrics, ListMetricsHandler(opts)))
		mcp.AddTool(mcpServer, tools.ExecuteInstantQuery.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.ExecuteInstantQuery.Name, opts.toolMetrics, ExecuteInstantQueryHandler(opts)))
		mcp.AddTool(mcpServer, tools.ExecuteRangeQuery.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.ExecuteRangeQuery.Name, opts.toolMetrics, ExecuteRangeQueryHandler(opts)))
		mcp.AddTool(mcpServer, tools.ShowTimeseries.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.ShowTimeseries.Name, opts.toolMetrics, ShowTimeseriesHandler(opts)))
		mcp.AddTool(mcpServer, tools.GetLabelNames.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.GetLabelNames.Name, opts.toolMetrics, GetLabelNamesHandler(opts)))
		mcp.AddTool(mcpServer, tools.GetLabelValues.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.GetLabelValues.Name, opts.toolMetrics, GetLabelValuesHandler(opts)))
		mcp.AddTool(mcpServer, tools.GetSeries.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.GetSeries.Name, opts.toolMetrics, GetSeriesHandler(opts)))
		mcp.AddTool(mcpServer, tools.GetAlerts.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.GetAlerts.Name, opts.toolMetrics, GetAlertsHandler(opts)))
		mcp.AddTool(mcpServer, tools.GetSilences.ToMCPTool(),
			metrics.InstrumentToolHandler(tools.GetSilences.Name, opts.toolMetrics, GetSilencesHandler(opts)))
	}

	if slices.Contains(opts.Toolsets, ToolsetTraces) {
		if opts.Traces == nil {
			return errors.New("configuration for traces toolset is missing")
		}

		opts.Traces.ClientMetrics = opts.clientMetrics
		err := addToolset(mcpServer, mgr, &traces.Toolset{}, opts.Traces, opts.toolMetrics)
		if err != nil {
			return err
		}
	}

	if slices.Contains(opts.Toolsets, ToolsetOtelcol) {
		if opts.Otelcol == nil {
			return errors.New("configuration for otelcol toolset is missing")
		}
		err := addToolset(mcpServer, mgr, &otelcol.Toolset{}, opts.Otelcol, opts.toolMetrics)
		if err != nil {
			return err
		}
	}

	if slices.Contains(opts.Toolsets, ToolsetLogs) {
		if opts.Logs == nil {
			return errors.New("configuration for logs toolset is missing")
		}
		opts.Logs.ClientMetrics = opts.clientMetrics
		err := addToolset(mcpServer, mgr, &logs.Toolset{}, opts.Logs, opts.toolMetrics)
		if err != nil {
			return err
		}
	}
	return nil
}

func addToolset(mcpServer *mcp.Server, mgr *kubernetes.Manager, toolset api.Toolset, toolsetConfig api.ExtendedConfig, toolMetrics *metrics.ToolMetrics) error {
	baseConfig := &mcpBaseConfig{toolsetConfig: toolsetConfig}
	serverTools := toolset.GetTools(nil)
	for i := range serverTools {
		goSdkTool, goSdkHandler, err := ServerToolToGoSdkTool(mgr, baseConfig, serverTools[i])
		if err != nil {
			return err
		}
		mcpServer.AddTool(goSdkTool, metrics.InstrumentToolHandlerUntyped(goSdkTool.Name, toolMetrics, goSdkHandler))
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

// NewHTTPServer creates an HTTP server for MCP over SSE.
// Returns the server and a shutdown function to be used with run.Group.
func NewHTTPServer(mcpServer *mcp.Server, listenAddr string, registry prom.Registerer, authMode auth.AuthMode) (*http.Server, func(error)) {
	mux := http.NewServeMux()

	var instrMiddleware metrics.InstrumentationMiddleware
	if registry != nil {
		instrMiddleware = metrics.NewInstrumentationMiddleware(registry, nil)
	} else {
		instrMiddleware = metrics.NewNopInstrumentationMiddleware()
	}

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
	mux.Handle(mcpEndpoint, instrMiddleware.NewHandler("mcp", streamableHandler))
	mux.Handle("/", instrMiddleware.NewHandler("root", streamableHandler))

	mux.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	shutdown := func(err error) {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer shutdownCancel()

		slog.Info("Shutting down HTTP server gracefully")
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Error("HTTP server shutdown error", "error", shutdownErr)
		} else {
			slog.Info("HTTP server shutdown complete")
		}
	}

	return httpServer, shutdown
}
