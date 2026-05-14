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

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/tools"
	"github.com/rhobs/obs-mcp/pkg/traces"
	tempoclient "github.com/rhobs/obs-mcp/pkg/traces/tempo"
)

type Toolset string

const (
	ToolsetMetrics Toolset = "metrics"
	ToolsetTraces  Toolset = "traces"
)

var AllToolsets = []string{string(ToolsetMetrics), string(ToolsetTraces)}

// ObsMCPOptions contains configuration options for the MCP server
type ObsMCPOptions struct {
	Toolsets               []Toolset
	AuthMode               AuthMode
	MetricsBackendURL      string
	AlertmanagerURL        string
	Insecure               bool
	Guardrails             *prometheus.Guardrails
	FullRangeQueryResponse bool
	Tempo                  *traces.Config
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
		if opts.Tempo == nil {
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
		mcp.AddTool(mcpServer, traces.ListInstancesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Tempo, tempoToolset.ListInstancesHandler))
		mcp.AddTool(mcpServer, traces.GetTraceByIDTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Tempo, tempoToolset.GetTraceByIDHandler))
		mcp.AddTool(mcpServer, traces.SearchTracesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Tempo, tempoToolset.SearchTracesHandler))
		mcp.AddTool(mcpServer, traces.SearchTagsTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Tempo, tempoToolset.SearchTagsHandler))
		mcp.AddTool(mcpServer, traces.SearchTagValuesTool.ToMCPTool(), traces.ToMCPHandler(newTempoClient, dynamicClient, opts.Tempo, tempoToolset.SearchTagValuesHandler))
	}
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

var sensitiveHeaders = map[string]bool{
	"Authorization": true,
	"Cookie":        true,
	"Set-Cookie":    true,
}

func redactHeaders(h http.Header) http.Header {
	redacted := h.Clone()
	for name := range redacted {
		if sensitiveHeaders[http.CanonicalHeaderKey(name)] {
			redacted.Set(name, "[REDACTED]")
		}
	}
	return redacted
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Incoming request", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		slog.Debug("Request headers", "headers", redactHeaders(r.Header))
		if r.ContentLength > 0 {
			slog.Info("Request content length", "content_length", r.ContentLength)
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := authFromRequest(r.Context(), r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Serve(ctx context.Context, mcpServer *mcp.Server, listenAddr string, authMode AuthMode) error {
	mux := http.NewServeMux()

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: loggingMiddleware(mux),
	}

	opts := &mcp.StreamableHTTPOptions{
		Stateless: true,
	}

	var mcpHandler http.Handler = mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, opts)

	if authMode == AuthModeHeader {
		mcpHandler = authMiddleware(mcpHandler)
	}

	mux.Handle(mcpEndpoint, mcpHandler)
	mux.Handle("/", mcpHandler)

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
