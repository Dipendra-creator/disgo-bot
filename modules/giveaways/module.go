// Package giveaways provides timed prize giveaways: a host starts one with a
// prize, duration and winner count; members enter via a button; an in-process
// sweeper draws winners when the timer expires; and winners can be rerolled. It
// follows the shared.Module contract.
package giveaways

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Module implements shared.Module.
type Module struct {
	shared.Base
	deps *shared.Deps
	svc  *Service
}

// New returns an uninitialised giveaways module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "giveaways" }

// Init injects dependencies and starts the end-time sweeper.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	// The sweeper runs for the process lifetime; it stops when the process exits.
	go m.svc.runSweeper(context.Background())
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{m.giveawayCommand()}
}

// Components maps custom-ID actions to handlers (namespace "giveaways").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		actionEnter: m.handleEnter,
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

// guildName returns the interaction guild's name, or a neutral fallback.
func (m *Module) guildName(c *shared.Context) string {
	if g := m.getGuild(c); g != nil && g.Name != "" {
		return g.Name
	}
	return "This server"
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	return shared.RequirePermission(m.getGuild(c), c.Member(), perm)
}
