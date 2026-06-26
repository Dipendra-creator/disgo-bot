// Package logging provides server audit logging: it mirrors gateway events
// (message edits/deletions, member joins/leaves, bans, channel and role
// changes) into per-category log channels configured with /logging. It follows
// the shared.Module contract and uses gateway event handlers rather than
// interactions for the event stream.
package logging

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

// New returns an uninitialised logging module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "logging" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{m.loggingCommand()}
}

// Events returns the gateway event handlers that feed the audit log. Member and
// message-content events require the privileged intents (see config).
func (m *Module) Events() []interface{} {
	return []interface{}{
		m.onMessageDelete,
		m.onMessageUpdate,
		m.onMemberAdd,
		m.onMemberRemove,
		m.onBanAdd,
		m.onBanRemove,
		m.onChannelCreate,
		m.onChannelDelete,
		m.onRoleCreate,
		m.onRoleDelete,
	}
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	guild := m.getGuild(c)
	return shared.RequirePermission(guild, c.Member(), perm)
}

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
