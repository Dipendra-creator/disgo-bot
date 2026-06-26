// Package moderation provides the server-moderation feature set: ban, kick,
// timeout, warn and friends, each recorded as a numbered case with optional DM
// notification and a public mod-log. It follows the shared.Module contract and
// reuses the framework's UI, permission and custom-ID helpers.
package moderation

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

// New returns an uninitialised moderation module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "moderation" }

// Init injects dependencies and starts the temp-ban sweeper.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	// The sweeper runs for the process lifetime; it stops when the process exits.
	go m.svc.runTempbanSweeper(context.Background())
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.banCommand(),
		m.unbanCommand(),
		m.kickCommand(),
		m.timeoutCommand(),
		m.untimeoutCommand(),
		m.warnCommand(),
		m.warningsCommand(),
		m.caseCommand(),
		m.reasonCommand(),
		m.delwarnCommand(),
		m.purgeCommand(),
		m.modlogCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "moderation").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		"warnings": m.handleWarningsPage,
		"noop":     func(c *shared.Context) error { return c.DeferUpdate() },
	}
}

// --- shared handler helpers ---

// getGuild resolves the interaction's guild, preferring the gateway state cache
// and falling back to REST.
func (m *Module) getGuild(c *shared.Context) (*discordgo.Guild, error) {
	gid := c.GuildID()
	if c.Session.State != nil {
		if g, err := c.Session.State.Guild(gid); err == nil && g != nil {
			return g, nil
		}
	}
	return c.Session.Guild(gid)
}

// getMember resolves a guild member, preferring state then REST.
func (m *Module) getMember(c *shared.Context, userID string) (*discordgo.Member, error) {
	gid := c.GuildID()
	if c.Session.State != nil {
		if mem, err := c.Session.State.Member(gid, userID); err == nil && mem != nil {
			return mem, nil
		}
	}
	return c.Session.GuildMember(gid, userID)
}

// botID returns the bot user's ID, or "" when the session state isn't ready.
func (m *Module) botID(c *shared.Context) string {
	if c.Session.State != nil && c.Session.State.User != nil {
		return c.Session.State.User.ID
	}
	return ""
}

// guard runs the standard pre-action checks for a member-targeting command:
// in-guild, invoker holds perm, and identity + role-hierarchy rules hold.
func (m *Module) guard(c *shared.Context, target *discordgo.User, targetMember *discordgo.Member, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	guild, _ := m.getGuild(c)
	if err := shared.RequirePermission(guild, c.Member(), perm); err != nil {
		return err
	}
	bot := m.botID(c)
	if err := checkTarget(guild, c.User().ID, target.ID, bot); err != nil {
		return err
	}
	var botMember *discordgo.Member
	if bot != "" {
		botMember, _ = m.getMember(c, bot)
	}
	return checkHierarchy(guild, c.Member(), targetMember, botMember)
}

// requirePerm checks the invoker holds perm in the (guild) interaction.
func (m *Module) requirePerm(c *shared.Context, perm int64) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	guild, _ := m.getGuild(c)
	return shared.RequirePermission(guild, c.Member(), perm)
}

// guildName returns a guild's display name, or a neutral fallback.
func guildName(g *discordgo.Guild) string {
	if g != nil && g.Name != "" {
		return g.Name
	}
	return "the server"
}

// editActionResult replaces a deferred ephemeral response with the case card,
// confirming the action privately to the moderator.
func (m *Module) editActionResult(c *shared.Context, cs *Case, target *discordgo.User) error {
	embed := caseEmbed(cs, target, c.User())
	_, err := c.Edit(&discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}
