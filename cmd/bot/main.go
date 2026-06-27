// Command bot is the disgo Discord bot entry point. It loads configuration,
// constructs the dependency container, registers feature modules and runs the
// gateway until interrupted.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/bot"
	"github.com/dipu-sharma/disgo-bot/internal/cache"
	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/dipu-sharma/disgo-bot/internal/database"
	"github.com/dipu-sharma/disgo-bot/internal/logger"
	"github.com/dipu-sharma/disgo-bot/internal/observability"
	"github.com/dipu-sharma/disgo-bot/internal/router"
	"github.com/dipu-sharma/disgo-bot/modules/automod"
	"github.com/dipu-sharma/disgo-bot/modules/economy"
	"github.com/dipu-sharma/disgo-bot/modules/leveling"
	"github.com/dipu-sharma/disgo-bot/modules/logging"
	"github.com/dipu-sharma/disgo-bot/modules/moderation"
	"github.com/dipu-sharma/disgo-bot/modules/tickets"
	"github.com/dipu-sharma/disgo-bot/modules/utility"
	"github.com/dipu-sharma/disgo-bot/modules/verification"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "path to config.yaml (overrides DISGO_CONFIG)")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Log)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()
	log.Info("starting disgo", zap.String("env", cfg.Env))

	flush, err := observability.InitSentry(cfg)
	if err != nil {
		log.Warn("sentry init failed", zap.Error(err))
	}
	defer flush()

	ctx := context.Background()

	// Database + migrations.
	db, err := database.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(ctx, db, log); err != nil {
		return err
	}

	// Cache (Redis or in-memory fallback).
	c := cache.New(ctx, cfg.Redis, log)
	defer func() { _ = c.Close() }()

	// Gateway session.
	session, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	metrics := observability.NewMetrics()

	deps := &shared.Deps{
		Config:  cfg,
		Log:     log,
		DB:      db,
		Cache:   c,
		Session: session,
		Metrics: metrics,
	}

	// Register feature modules. New modules are added to this slice.
	registry := router.NewRegistry(log)
	modules := []shared.Module{
		utility.New(),
		moderation.New(),
		tickets.New(),
		logging.New(),
		leveling.New(),
		economy.New(),
		verification.New(),
		automod.New(),
	}
	for _, mod := range modules {
		if err := mod.Init(deps); err != nil {
			return fmt.Errorf("init module %q: %w", mod.Name(), err)
		}
		if err := registry.Register(mod); err != nil {
			return fmt.Errorf("register module %q: %w", mod.Name(), err)
		}
	}

	b := bot.New(cfg, log, session, registry, deps)

	// Observability HTTP servers (metrics + health); /readyz reflects the bot.
	servers := observability.Start(cfg, metrics, b.Ready, log)

	if err := b.Open(); err != nil {
		return err
	}
	log.Info("disgo is running — press CTRL-C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := b.Close(); err != nil {
		log.Warn("gateway close", zap.Error(err))
	}
	servers.Shutdown(shutdownCtx)
	return nil
}
