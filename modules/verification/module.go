// Package verification provides a member-gate: an admin configures a verified
// role and posts a panel, and members click a button to receive the role. The
// grant is idempotent and each first verification is audited and optionally
// logged. It follows the shared.Module contract.
package verification

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

// New returns an uninitialised verification module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "verification" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.setupCommand(),
		m.panelCommand(),
		m.disableCommand(),
		m.statusCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "verification").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		actionVerify: m.handleVerify,
	}
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
