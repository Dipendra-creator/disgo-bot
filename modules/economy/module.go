// Package economy provides a per-guild virtual currency system: members earn
// currency from /daily and /work, hold it in a wallet and bank, transfer it to
// each other, and spend it in a configurable shop that can grant roles. It is
// deliberately NON-GAMBLING — there are no betting or chance-based mechanics.
// It follows the shared.Module contract.
package economy

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

// New returns an uninitialised economy module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "economy" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.balanceCommand(),
		m.dailyCommand(),
		m.workCommand(),
		m.payCommand(),
		m.depositCommand(),
		m.withdrawCommand(),
		m.shopCommand(),
		m.buyCommand(),
		m.inventoryCommand(),
		m.richCommand(),
		m.configCommand(),
		m.adminCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "economy").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		"shop": m.handleShopPage,
		"rich": m.handleRichPage,
		"noop": func(c *shared.Context) error { return c.DeferUpdate() },
	}
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return errGuildOnly
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
