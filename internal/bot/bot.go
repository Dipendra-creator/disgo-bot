// Package bot owns the discordgo session lifecycle: configuring intents, wiring
// the interaction router and module event handlers, opening the gateway,
// syncing application commands on ready and shutting down gracefully.
package bot

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/dipu-sharma/disgo-bot/internal/router"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// DefaultIntents are the non-privileged gateway intents the bot requests.
// Privileged intents (GuildMembers, MessageContent) are intentionally excluded
// so the bot connects without portal configuration; member data is fetched via
// REST when needed. Add them here once enabled in the developer portal.
const DefaultIntents = discordgo.IntentsGuilds |
	discordgo.IntentsGuildMessages |
	discordgo.IntentsGuildBans |
	discordgo.IntentsGuildVoiceStates |
	discordgo.IntentsGuildEmojis

// Bot bundles the gateway session with the router and dependencies.
type Bot struct {
	cfg      *config.Config
	log      *zap.Logger
	session  *discordgo.Session
	registry *router.Registry
	deps     *shared.Deps

	ready    atomic.Bool
	syncOnce sync.Once
}

// New constructs a Bot around an already-created session.
func New(cfg *config.Config, log *zap.Logger, session *discordgo.Session, registry *router.Registry, deps *shared.Deps) *Bot {
	return &Bot{
		cfg:      cfg,
		log:      log,
		session:  session,
		registry: registry,
		deps:     deps,
	}
}

// Open wires handlers, sets intents and connects to the gateway. Command sync
// and presence are performed in the ready handler.
func (b *Bot) Open() error {
	b.session.Identify.Intents = DefaultIntents

	// Single dispatch entry-point for all interactions.
	b.session.AddHandler(b.registry.Dispatch(b.deps))

	// Module-provided gateway event handlers.
	for _, h := range b.registry.Events() {
		b.session.AddHandler(h)
	}

	// Ready: sync commands once and mark the bot ready for /readyz.
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		b.log.Info("gateway ready",
			zap.String("user", r.User.String()),
			zap.Int("guilds", len(r.Guilds)),
		)
		b.syncOnce.Do(func() {
			if err := b.registry.Sync(s, b.cfg.Discord.AppID, b.cfg.Discord.DevGuildID); err != nil {
				b.log.Error("command sync failed", zap.Error(err))
				return
			}
			_ = s.UpdateGameStatus(0, "/help • disgo")
			b.ready.Store(true)
		})
	})

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open gateway: %w", err)
	}
	return nil
}

// Ready reports whether the gateway is connected and commands are synced.
func (b *Bot) Ready() bool { return b.ready.Load() }

// Close disconnects from the gateway.
func (b *Bot) Close() error {
	b.ready.Store(false)
	return b.session.Close()
}
