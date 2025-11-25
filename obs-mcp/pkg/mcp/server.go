package mcp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// ObsMCPOptions contains configuration options for the MCP server
type ObsMCPOptions struct {
	AuthMode      AuthMode
	PromURL       string
	Insecure      bool
	UseGuardrails bool
}

const (
	mcpEndpoint    = "/mcp"
	healthEndpoint = "/health"
)

func NewMCPServer(opts ObsMCPOptions) (*server.MCPServer, error) {
	mcpServer := server.NewMCPServer(
		"obs-mcp",
		"1.0.0",
		server.WithLogging(),
		server.WithToolCapabilities(true),
	)

	if err := SetupTools(mcpServer, opts); err != nil {
		return nil, err
	}

	return mcpServer, nil
}

func SetupTools(mcpServer *server.MCPServer, opts ObsMCPOptions) error {
	// Create tool definitions
	listMetricsTool := CreateListMetricsTool()
	executeRangeQueryTool := CreateExecuteRangeQueryTool()

	// Create handlers
	listMetricsHandler := ListMetricsHandler(opts)
	executeRangeQueryHandler := ExecuteRangeQueryHandler(opts)

	// Add tools to server
	mcpServer.AddTool(listMetricsTool, listMetricsHandler)
	mcpServer.AddTool(executeRangeQueryTool, executeRangeQueryHandler)

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
		log.Printf("[DEBUG] Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("[DEBUG] Request headers: %v", r.Header)
		if r.ContentLength > 0 {
			log.Printf("[DEBUG] Content-Length: %d", r.ContentLength)
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
		w.Write([]byte("OK"))
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP server starting on %s with MCP endpoint at %s", listenAddr, mcpEndpoint)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, initiating graceful shutdown", sig)
		cancel()
	case <-ctx.Done():
		log.Printf("Context cancelled, initiating graceful shutdown")
	case err := <-serverErr:
		log.Printf("HTTP server error: %v", err)
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	log.Printf("Shutting down HTTP server gracefully...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
		return err
	}

	log.Printf("HTTP server shutdown complete")
	return nil
}
