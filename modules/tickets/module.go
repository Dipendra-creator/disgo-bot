// Package tickets provides a support-ticket system: an admin posts a panel,
// users open private ticket channels via a button, staff claim and close them,
// and a transcript is logged on close. It follows the shared.Module contract.
package tickets

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

// New returns an uninitialised tickets module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "tickets" }

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
		m.closeCommand(),
		m.addCommand(),
		m.removeCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "tickets").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		actionOpen:         m.handleOpen,
		actionClaim:        m.handleClaim,
		actionClose:        m.handleCloseRequest,
		actionConfirmClose: m.handleConfirmClose,
		actionCancelClose:  m.handleCancelClose,
	}
}

// --- shared helpers ---

func (m *Module) getGuild(c *shared.Context) (*discordgo.Guild, error) {
	gid := c.GuildID()
	if c.Session.State != nil {
		if g, err := c.Session.State.Guild(gid); err == nil && g != nil {
			return g, nil
		}
	}
	return c.Session.Guild(gid)
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	guild, _ := m.getGuild(c)
	return shared.RequirePermission(guild, c.Member(), perm)
}

// isStaff reports whether the invoker is ticket staff: holds the configured
// staff role, or has Manage Channels.
func (m *Module) isStaff(c *shared.Context, set *Settings) bool {
	member := c.Member()
	if member == nil {
		return false
	}
	if set != nil && set.StaffRoleID != 0 {
		want := sid(set.StaffRoleID)
		for _, r := range member.Roles {
			if r == want {
				return true
			}
		}
	}
	guild, _ := m.getGuild(c)
	return shared.HasPermission(guild, member, discordgo.PermissionManageChannels)
}

// canManageTicket reports whether the invoker may act on a ticket: staff, or the
// ticket's opener.
func (m *Module) canManageTicket(c *shared.Context, set *Settings, t *Ticket) bool {
	return m.isStaff(c, set) || t.OpenerID == pid(c.User().ID)
}
