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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/client-go/dynamic"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/logs"
	lokiclient "github.com/rhobs/obs-mcp/pkg/logs/loki"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/tools"
	"github.com/rhobs/obs-mcp/pkg/traces"
	tempoclient "github.com/rhobs/obs-mcp/pkg/traces/tempo"
)

type Toolset string

const (
	ToolsetMetrics Toolset = "metrics"
	ToolsetTraces  Toolset = "traces"
	ToolsetLogs    Toolset = "logs"
	ToolsetOtelcol Toolset = "otelcol"
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
	LokiURL                string
	LokiUseRoute           bool
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

		tempoToolset := &traces.Toolset{}
		newTempoClient := func(ctx context.Context, url string) (tempoclient.Loader, error) {
			return getTempoHTTPClient(ctx, opts, url)
		}
		restConfig, err := k8s.GetClientConfig()
		if err != nil {
			return err
		}
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		mcp.AddTool(mcpServer, traces.ListInstancesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Traces, tempoToolset.ListInstancesHandler))
		mcp.AddTool(mcpServer, traces.GetTraceByIDTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Traces, tempoToolset.GetTraceByIDHandler))
		mcp.AddTool(mcpServer, traces.SearchTracesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Traces, tempoToolset.SearchTracesHandler))
		mcp.AddTool(mcpServer, traces.SearchTagsTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Traces, tempoToolset.SearchTagsHandler))
		mcp.AddTool(mcpServer, traces.SearchTagValuesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Traces, tempoToolset.SearchTagValuesHandler))
	}

	if slices.Contains(opts.Toolsets, ToolsetOtelcol) {
		if opts.Otelcol == nil {
			return errors.New("configuration for otelcol toolset is missing")
		}
		mcp.AddTool(mcpServer, otelcol.ListComponents.ToMCPTool(), otelcol.ToMCPHandler[otelcol.ListComponentsInput, otelcol.ListComponentsOutput](opts.Otelcol, otelcol.BuildListComponentsInput, otelcol.ListComponentsHandler))
		mcp.AddTool(mcpServer, otelcol.GetComponentSchema.ToMCPTool(), otelcol.ToMCPHandler[otelcol.GetComponentSchemaInput, otelcol.GetComponentSchemaOutput](opts.Otelcol, otelcol.BuildGetComponentSchemaInput, otelcol.GetComponentSchemaHandler))
		mcp.AddTool(mcpServer, otelcol.ValidateConfig.ToMCPTool(), otelcol.ToMCPHandler[otelcol.ValidateConfigInput, otelcol.ValidateConfigOutput](opts.Otelcol, otelcol.BuildValidateConfigInput, otelcol.ValidateConfigHandler))
		mcp.AddTool(mcpServer, otelcol.GetVersions.ToMCPTool(), otelcol.ToMCPHandler[otelcol.GetVersionsInput, otelcol.GetVersionsOutput](opts.Otelcol, otelcol.BuildGetVersionsInput, otelcol.GetVersionsHandler))
	}

	if slices.Contains(opts.Toolsets, ToolsetLogs) {
		logsCfg := &logs.Config{
			LokiURL:  opts.LokiURL,
			UseRoute: opts.LokiUseRoute,
		}
		var logsDynamicClient dynamic.Interface
		if restConfig, err := k8s.GetClientConfig(); err == nil {
			if c, err := dynamic.NewForConfig(restConfig); err == nil {
				logsDynamicClient = c
			} else {
				slog.Warn("LokiStack discovery disabled: failed to create Kubernetes dynamic client", "err", err)
			}
		} else {
			slog.Warn("LokiStack discovery disabled: failed to get Kubernetes client config", "err", err)
		}

		newLokiClient := func(ctx context.Context, url, tenant string) (lokiclient.Loader, error) {
			return getLokiClient(ctx, opts, url, tenant)
		}
		mcp.AddTool(mcpServer, logs.ListInstancesTool.ToMCPTool(), logs.ToMCPHandler(newLokiClient, logsDynamicClient, logsCfg, logs.ListInstancesHandler))
		mcp.AddTool(mcpServer, logs.LabelNamesTool.ToMCPTool(), logs.ToMCPHandler(newLokiClient, logsDynamicClient, logsCfg, logs.LabelNamesHandler))
		mcp.AddTool(mcpServer, logs.LabelValuesTool.ToMCPTool(), logs.ToMCPHandler(newLokiClient, logsDynamicClient, logsCfg, logs.LabelValuesHandler))
		mcp.AddTool(mcpServer, logs.QueryRangeTool.ToMCPTool(), logs.ToMCPHandler(newLokiClient, logsDynamicClient, logsCfg, logs.QueryRangeHandler))
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
