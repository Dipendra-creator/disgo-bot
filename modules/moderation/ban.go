package moderation

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/pkg/duration"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) banCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "ban",
			Description:              "Ban a member from the server (optionally for a limited time)",
			DefaultMemberPermissions: permPtr(discordgo.PermissionBanMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member to ban", true),
				strOpt("reason", "Reason for the ban", false),
				func() *discordgo.ApplicationCommandOption {
					o := intOpt("delete_days", "Days of their recent messages to delete (0-7)", false)
					o.MinValue = floatPtr(0)
					o.MaxValue = 7
					return o
				}(),
				strOpt("duration", "Temp-ban length, e.g. 7d or 12h (omit for permanent)", false),
			},
		},
		Handler: m.handleBan,
	}
}

func (m *Module) handleBan(c *shared.Context) error {
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user to ban.")
	}
	if err := m.guard(c, target, optMember(c, "user"), discordgo.PermissionBanMembers); err != nil {
		return err
	}

	var dur time.Duration
	if ds := strings.TrimSpace(optStr(c, "duration")); ds != "" {
		d, err := duration.Parse(ds)
		if err != nil {
			return shared.UserErr("Invalid duration %q. Try formats like 10m, 2h or 7d.", ds)
		}
		dur = d
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Ban(c.Ctx, actionInput{
		GuildID:    c.GuildID(),
		GuildName:  guildName(guild),
		Target:     target,
		Mod:        c.User(),
		Reason:     optStr(c, "reason"),
		Duration:   dur,
		DeleteDays: int(optInt(c, "delete_days")),
	})
	if err != nil {
		return err
	}
	return m.editActionResult(c, cs, target)
}

func (m *Module) unbanCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "unban",
			Description:              "Lift a ban by user ID",
			DefaultMemberPermissions: permPtr(discordgo.PermissionBanMembers),
			Options: []*discordgo.ApplicationCommandOption{
				strOpt("user_id", "The ID of the banned user", true),
				strOpt("reason", "Reason for the unban", false),
			},
		},
		Handler: m.handleUnban,
	}
}

func (m *Module) handleUnban(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionBanMembers); err != nil {
		return err
	}
	id := strings.TrimSpace(optStr(c, "user_id"))
	if !isSnowflake(id) {
		return shared.UserErr("Provide a valid user ID.")
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	target, err := c.Session.User(id)
	if err != nil || target == nil {
		target = &discordgo.User{ID: id, Username: id}
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Unban(c.Ctx, actionInput{
		GuildID:   c.GuildID(),
		GuildName: guildName(guild),
		Target:    target,
		Mod:       c.User(),
		Reason:    optStr(c, "reason"),
	})
	if err != nil {
		return shared.UserErr("Couldn't unban that user — are they actually banned?")
	}
	return m.editActionResult(c, cs, target)
}
