package observability

import (
	"fmt"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/getsentry/sentry-go"
)

// InitSentry initializes the Sentry client when enabled. It returns a flush
// function the caller must defer to drain buffered events on shutdown. When
// Sentry is disabled it returns a no-op flush.
func InitSentry(cfg *config.Config) (flush func(), err error) {
	if !cfg.Sentry.Enabled || cfg.Sentry.DSN == "" {
		return func() {}, nil
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:           cfg.Sentry.DSN,
		Environment:   cfg.Env,
		EnableTracing: false,
	}); err != nil {
		return func() {}, fmt.Errorf("init sentry: %w", err)
	}
	return func() { sentry.Flush(2 * time.Second) }, nil
}

// CaptureError reports an error to Sentry if it has been initialized.
func CaptureError(err error) {
	if err != nil {
		sentry.CaptureException(err)
	}
}
