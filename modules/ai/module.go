// Package ai provides an opt-in AI assistant: a public, rate-limited /ask
// command plus an admin-configurable assistant channel where the bot replies to
// every message. Completions go through a Provider interface (an Anthropic
// Messages API implementation by default) so the backend stays swappable. It
// follows the shared.Module contract and is inert until the bot owner supplies
// an API key (config.AI.Ready()).
package ai

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Module implements shared.Module.
type Module struct {
	shared.Base
	deps *shared.Deps
	svc  *Service
}

// New returns an uninitialised AI module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "ai" }

// Init injects dependencies and wires the service (and provider, when an API key
// is configured).
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.askCommand(),
		m.aiCommand(),
	}
}

// Events returns the gateway handler driving the opt-in assistant channel.
// Reading message content requires the privileged MessageContent intent.
func (m *Module) Events() []interface{} {
	return []interface{}{m.onMessageCreate}
}

// --- shared helpers ---

func (m *Module) getGuild(c *shared.Context) *discordgo.Guild {
	gid := c.GuildID()
	if c.Session.State != nil {
		if g, err := c.Session.State.Guild(gid); err == nil && g != nil {
			return g
		}
	}
	g, _ := c.Session.Guild(gid)
	return g
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	return shared.RequirePermission(m.getGuild(c), c.Member(), perm)
}
