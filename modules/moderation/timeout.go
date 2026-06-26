package moderation

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/pkg/duration"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) timeoutCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "timeout",
			Description:              "Temporarily mute a member (Discord timeout, max 28 days)",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member to time out", true),
				strOpt("duration", "Timeout length, e.g. 10m, 2h or 7d", true),
				strOpt("reason", "Reason for the timeout", false),
			},
		},
		Handler: m.handleTimeout,
	}
}

func (m *Module) handleTimeout(c *shared.Context) error {
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user to time out.")
	}
	if err := m.guard(c, target, optMember(c, "user"), discordgo.PermissionModerateMembers); err != nil {
		return err
	}

	dur, err := duration.Parse(strings.TrimSpace(optStr(c, "duration")))
	if err != nil {
		return shared.UserErr("Invalid duration. Try formats like 10m, 2h or 7d.")
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Timeout(c.Ctx, actionInput{
		GuildID:   c.GuildID(),
		GuildName: guildName(guild),
		Target:    target,
		Mod:       c.User(),
		Reason:    optStr(c, "reason"),
		Duration:  dur,
	})
	if err != nil {
		return err
	}
	return m.editActionResult(c, cs, target)
}

func (m *Module) untimeoutCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "untimeout",
			Description:              "Clear a member's timeout",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member whose timeout to clear", true),
				strOpt("reason", "Reason", false),
			},
		},
		Handler: m.handleUntimeout,
	}
}

func (m *Module) handleUntimeout(c *shared.Context) error {
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user.")
	}
	if err := m.guard(c, target, optMember(c, "user"), discordgo.PermissionModerateMembers); err != nil {
		return err
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Untimeout(c.Ctx, actionInput{
		GuildID:   c.GuildID(),
		GuildName: guildName(guild),
		Target:    target,
		Mod:       c.User(),
		Reason:    optStr(c, "reason"),
	})
	if err != nil {
		return err
	}
	return m.editActionResult(c, cs, target)
}
