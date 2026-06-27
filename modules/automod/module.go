// Package automod provides automatic content moderation: configurable filters
// for banned words, invite links, mass mentions and message spam, each with its
// own action (delete the message, or delete and timeout the author). It follows
// the shared.Module contract and consumes gateway message events rather than
// interactions for the filter pipeline.
package automod

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

// New returns an uninitialised automod module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "automod" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.automodCommand(),
		m.wordsCommand(),
	}
}

// Events returns the gateway handler that drives the filter pipeline. Inspecting
// message content requires the privileged MessageContent intent (see config).
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
