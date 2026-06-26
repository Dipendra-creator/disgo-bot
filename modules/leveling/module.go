// Package leveling provides an XP and ranking system: members earn XP for
// chatting (rate-limited per user), level up along a fixed curve, optionally
// receive reward roles, and can view rank cards and a leaderboard. It follows
// the shared.Module contract, consuming MessageCreate events to award XP.
package leveling

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

// New returns an uninitialised leveling module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "leveling" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.rankCommand(),
		m.leaderboardCommand(),
		m.configCommand(),
		m.roleCommand(),
		m.xpCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "leveling").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		"lb":   m.handleLeaderboardPage,
		"noop": func(c *shared.Context) error { return c.DeferUpdate() },
	}
}

// Events returns the gateway handlers that feed XP gain.
func (m *Module) Events() []interface{} {
	return []interface{}{m.onMessageCreate}
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	return shared.RequirePermission(m.getGuild(c), c.Member(), perm)
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
