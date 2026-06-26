package config_test

import (
	"testing"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("DISCORD_APP_ID", "12345")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("LOG_LEVEL", "debug")

	// No file at this path; values must come from defaults + env.
	cfg, err := config.Load("testdata/does-not-exist.yaml")
	require.NoError(t, err)

	assert.Equal(t, "tok", cfg.Discord.Token)
	assert.Equal(t, "12345", cfg.Discord.AppID)
	assert.Equal(t, "secret", cfg.Postgres.Password)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "development", cfg.Env)
}

func TestValidateRejectsMissingSecrets(t *testing.T) {
	cfg := config.Default() // token + app_id empty
	err := config.Validate(&cfg)
	require.Error(t, err)
}

func TestConnStringFromFields(t *testing.T) {
	p := config.PostgresConfig{
		Host: "db", Port: 5432, User: "u", Password: "p", Database: "d", SSLMode: "require",
	}
	assert.Equal(t, "postgres://u:p@db:5432/d?sslmode=require", p.ConnString())
}

func TestConnStringPrefersDSN(t *testing.T) {
	p := config.PostgresConfig{DSN: "postgres://custom", Host: "ignored"}
	assert.Equal(t, "postgres://custom", p.ConnString())
}

func TestIsDev(t *testing.T) {
	assert.True(t, config.DiscordConfig{DevGuildID: "1"}.IsDev())
	assert.False(t, config.DiscordConfig{}.IsDev())
}
