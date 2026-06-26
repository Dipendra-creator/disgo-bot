// Package config loads, merges and validates the bot's runtime configuration.
//
// Configuration is sourced in three layers, later layers overriding earlier:
//  1. built-in defaults (see Default)
//  2. a YAML file (path from DISGO_CONFIG env or the path passed to Load)
//  3. environment variables for secrets and common overrides
//
// The resulting Config is validated with go-playground/validator before use.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Config is the fully-resolved, validated bot configuration.
type Config struct {
	// Env is the deployment environment: "development" or "production".
	Env string `yaml:"env" validate:"required,oneof=development production"`

	Discord  DiscordConfig  `yaml:"discord" validate:"required"`
	Postgres PostgresConfig `yaml:"postgres" validate:"required"`
	Redis    RedisConfig    `yaml:"redis"`
	Log      LogConfig      `yaml:"log"`
	Sentry   SentryConfig   `yaml:"sentry"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	HTTP     HTTPConfig     `yaml:"http"`
}

// DiscordConfig holds gateway credentials and command-registration settings.
type DiscordConfig struct {
	// Token is the bot token. Provide via DISCORD_TOKEN in real deployments.
	Token string `yaml:"token" validate:"required"`
	// AppID is the application (client) ID, used for command registration.
	AppID string `yaml:"app_id" validate:"required"`
	// DevGuildID, when set, registers commands guild-scoped for instant updates
	// during development. Leave empty in production to register globally.
	DevGuildID string `yaml:"dev_guild_id"`
	// Shards is the total shard count. 0 lets discordgo decide (single shard now).
	Shards int `yaml:"shards" validate:"gte=0"`
}

// IsDev reports whether commands should be registered guild-scoped.
func (d DiscordConfig) IsDev() bool { return d.DevGuildID != "" }

// PostgresConfig configures the PostgreSQL connection. A full DSN takes
// precedence; otherwise one is assembled from the individual fields.
type PostgresConfig struct {
	DSN      string `yaml:"dsn"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
	// PoolMax is the maximum number of open connections.
	PoolMax int `yaml:"pool_max" validate:"gte=1"`
}

// ConnString returns a libpq-style connection URL.
func (p PostgresConfig) ConnString() string {
	if p.DSN != "" {
		return p.DSN
	}
	ssl := p.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, ssl)
}

// RedisConfig configures the optional Redis cache/queue. When Enabled is false
// the bot falls back to an in-process memory cache (dev-friendly).
type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db" validate:"gte=0"`
}

// LogConfig configures the structured logger.
type LogConfig struct {
	// Level is one of: debug, info, warn, error.
	Level string `yaml:"level" validate:"required,oneof=debug info warn error"`
	// Format is "json" (production) or "console" (development).
	Format string `yaml:"format" validate:"required,oneof=json console"`
}

// SentryConfig configures error tracking.
type SentryConfig struct {
	Enabled bool   `yaml:"enabled"`
	DSN     string `yaml:"dsn"`
}

// MetricsConfig configures the Prometheus metrics endpoint.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
	Path    string `yaml:"path"`
}

// HTTPConfig configures the health-check HTTP server.
type HTTPConfig struct {
	HealthAddr string `yaml:"health_addr"`
}

// Default returns a Config populated with sensible development defaults.
// Secrets (token, passwords) are intentionally left empty.
func Default() Config {
	return Config{
		Env: "development",
		Discord: DiscordConfig{
			Shards: 0,
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Database: "disgo",
			SSLMode:  "disable",
			PoolMax:  10,
		},
		Redis: RedisConfig{
			Enabled: false,
			Addr:    "localhost:6379",
			DB:      0,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "console",
		},
		Sentry: SentryConfig{Enabled: false},
		Metrics: MetricsConfig{
			Enabled: true,
			Addr:    ":9090",
			Path:    "/metrics",
		},
		HTTP: HTTPConfig{HealthAddr: ":8080"},
	}
}

// Load reads configuration from the given YAML path (falling back to the
// DISGO_CONFIG env var, then "config.yaml"), applies environment overrides and
// validates the result. A missing file is not an error when every required
// value is supplied through the environment.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		path = os.Getenv("DISGO_CONFIG")
	}
	if path == "" {
		path = "config.yaml"
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	applyEnv(&cfg)

	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate runs struct-tag validation against the config.
func Validate(cfg *Config) error {
	if err := validator.New().Struct(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	return nil
}

// applyEnv overrides config fields from environment variables. Only secrets and
// frequently-tuned values are wired here to keep the surface predictable.
func applyEnv(cfg *Config) {
	setStr(&cfg.Env, "DISGO_ENV")
	setStr(&cfg.Discord.Token, "DISCORD_TOKEN")
	setStr(&cfg.Discord.AppID, "DISCORD_APP_ID")
	setStr(&cfg.Discord.DevGuildID, "DISCORD_DEV_GUILD_ID")

	setStr(&cfg.Postgres.DSN, "DATABASE_URL")
	setStr(&cfg.Postgres.Host, "POSTGRES_HOST")
	setInt(&cfg.Postgres.Port, "POSTGRES_PORT")
	setStr(&cfg.Postgres.User, "POSTGRES_USER")
	setStr(&cfg.Postgres.Password, "POSTGRES_PASSWORD")
	setStr(&cfg.Postgres.Database, "POSTGRES_DB")

	setBool(&cfg.Redis.Enabled, "REDIS_ENABLED")
	setStr(&cfg.Redis.Addr, "REDIS_ADDR")
	setStr(&cfg.Redis.Password, "REDIS_PASSWORD")

	setStr(&cfg.Log.Level, "LOG_LEVEL")
	setStr(&cfg.Log.Format, "LOG_FORMAT")

	setBool(&cfg.Sentry.Enabled, "SENTRY_ENABLED")
	setStr(&cfg.Sentry.DSN, "SENTRY_DSN")
}

func setStr(dst *string, env string) {
	if v, ok := os.LookupEnv(env); ok && v != "" {
		*dst = v
	}
}

func setInt(dst *int, env string) {
	if v, ok := os.LookupEnv(env); ok && v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			*dst = n
		}
	}
}

func setBool(dst *bool, env string) {
	if v, ok := os.LookupEnv(env); ok && v != "" {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			*dst = b
		}
	}
}
