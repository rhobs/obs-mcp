package health

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultShutdownTimeout = 10 * time.Second
)

// Server provides an internal HTTP server for metrics, health, and pprof endpoints
type Server struct {
	registry prom.Gatherer
	mux      *http.ServeMux
}

// NewServer creates a new health server with metrics and pprof endpoints
func NewServer(registry prom.Gatherer) *Server {
	s := &Server{
		registry: registry,
		mux:      http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health endpoints
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	s.mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Metrics endpoint
	if s.registry != nil {
		s.mux.Handle("/metrics", promhttp.HandlerFor(
			s.registry,
			promhttp.HandlerOpts{},
		))
	}

	// pprof endpoints
	s.mux.HandleFunc("/debug/pprof/", pprof.Index)
	s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// ListenAndServe starts the internal HTTP server on the given address
// Returns a server and a shutdown function compatible with oklog/run.Group
func (s *Server) ListenAndServe(listenAddr string) (*http.Server, func(error)) {
	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: s.mux,
	}

	shutdown := func(err error) {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer shutdownCancel()

		slog.Info("Shutting down internal health server gracefully")
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Error("Internal health server shutdown error", "error", shutdownErr)
		} else {
			slog.Info("Internal health server shutdown complete")
		}
	}

	return httpServer, shutdown
}
