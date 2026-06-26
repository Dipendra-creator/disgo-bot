package moderation

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) kickCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "kick",
			Description:              "Kick a member from the server",
			DefaultMemberPermissions: permPtr(discordgo.PermissionKickMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member to kick", true),
				strOpt("reason", "Reason for the kick", false),
			},
		},
		Handler: m.handleKick,
	}
}

func (m *Module) handleKick(c *shared.Context) error {
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user to kick.")
	}
	if err := m.guard(c, target, optMember(c, "user"), discordgo.PermissionKickMembers); err != nil {
		return err
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Kick(c.Ctx, actionInput{
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
