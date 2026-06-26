// Package logger constructs the application's structured zap logger.
package logger

import (
	"fmt"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a *zap.Logger from the given LogConfig. The "console" format is
// human-friendly for development; "json" emits machine-parseable lines for
// production log aggregation.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Level, err)
	}

	zc := zap.NewProductionConfig()
	zc.Level = zap.NewAtomicLevelAt(level)
	zc.Encoding = "json"
	zc.EncoderConfig.TimeKey = "ts"
	zc.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if cfg.Format == "console" {
		zc = zap.NewDevelopmentConfig()
		zc.Level = zap.NewAtomicLevelAt(level)
		zc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zc.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	log, err := zc.Build(zap.AddCaller())
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return log, nil
}
