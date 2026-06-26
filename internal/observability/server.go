package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Servers bundles the long-running HTTP servers (metrics + health) so the caller
// can shut them all down together.
type Servers struct {
	servers []*http.Server
	log     *zap.Logger
}

// readyFunc reports whether the bot is ready to serve (e.g. gateway connected).
type readyFunc func() bool

// Start launches the metrics and health HTTP servers per configuration. The
// ready callback backs the /readyz endpoint.
func Start(cfg *config.Config, m *Metrics, ready readyFunc, log *zap.Logger) *Servers {
	s := &Servers{log: log}

	if cfg.Metrics.Enabled {
		mux := http.NewServeMux()
		mux.Handle(cfg.Metrics.Path, promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
		s.run("metrics", cfg.Metrics.Addr, mux)
	}

	if cfg.HTTP.HealthAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
			if ready != nil && ready() {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ready"))
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		})
		s.run("health", cfg.HTTP.HealthAddr, mux)
	}

	return s
}

func (s *Servers) run(name, addr string, handler http.Handler) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.servers = append(s.servers, srv)
	go func() {
		s.log.Info("http server listening", zap.String("server", name), zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("http server error", zap.String("server", name), zap.Error(err))
		}
	}()
}

// Shutdown gracefully stops all servers.
func (s *Servers) Shutdown(ctx context.Context) {
	for _, srv := range s.servers {
		if err := srv.Shutdown(ctx); err != nil {
			s.log.Warn("http server shutdown", zap.Error(err))
		}
	}
}
