package leveling

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// cooldown / XP bounds keep configuration sane.
const (
	minCooldown = 0
	maxCooldown = 3600
	maxXPGrant  = 100
)

func (m *Module) configCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "level-config",
			Description:              "Configure the leveling system",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				subCmd("status", "Show the current configuration"),
				subCmd("enable", "Enable XP gain"),
				subCmd("disable", "Disable XP gain"),
				subCmd("cooldown", "Set the per-user XP cooldown (seconds)", intOpt("seconds", "0–3600", true)),
				subCmd("xp-range", "Set XP awarded per message",
					intOpt("min", "Minimum XP", true), intOpt("max", "Maximum XP", true)),
				subCmd("announce", "Where to announce level-ups",
					&discordgo.ApplicationCommandOption{
						Type: discordgo.ApplicationCommandOptionString, Name: "mode", Description: "Announcement mode", Required: true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "In the active channel", Value: "here"},
							{Name: "In a specific channel", Value: "channel"},
							{Name: "Off", Value: "off"},
						},
					},
					channelOpt("channel", "Channel (when mode is 'channel')", false)),
				subCmd("stack", "Whether reward roles stack", boolOpt("enabled", "Stack roles", true)),
			},
		},
		Handler: m.handleConfig,
	}
}

func (m *Module) handleConfig(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}

	switch subName(c) {
	case "status":
		// no change
	case "enable":
		set.Enabled = true
	case "disable":
		set.Enabled = false
	case "cooldown":
		secs := int(subInt(c, "seconds"))
		if secs < minCooldown || secs > maxCooldown {
			return shared.UserErr("Cooldown must be between %d and %d seconds.", minCooldown, maxCooldown)
		}
		set.XPCooldownSeconds = secs
	case "xp-range":
		min, max := int(subInt(c, "min")), int(subInt(c, "max"))
		if min < 1 || max < min || max > 1000 {
			return shared.UserErr("Provide 1 ≤ min ≤ max ≤ 1000.")
		}
		set.XPMin, set.XPMax = min, max
	case "announce":
		if err := applyAnnounce(c, set); err != nil {
			return err
		}
	case "stack":
		set.StackRoles = subBool(c, "enabled")
	default:
		return shared.UserErr("Unknown subcommand.")
	}

	if subName(c) != "status" {
		if err := m.svc.SaveSettings(c.Ctx, set); err != nil {
			return err
		}
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{settingsEmbed(set)},
	}, true)
}

func applyAnnounce(c *shared.Context, set *Settings) error {
	switch subStr(c, "mode") {
	case "here":
		set.AnnounceEnabled, set.AnnounceChannelID = true, 0
	case "off":
		set.AnnounceEnabled = false
	case "channel":
		ch := subChannel(c, "channel")
		if ch == nil {
			return shared.UserErr("Pick a channel for the 'channel' mode.")
		}
		if ch.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("The announcement channel must be a text channel.")
		}
		set.AnnounceEnabled, set.AnnounceChannelID = true, pid(ch.ID)
	default:
		return shared.UserErr("Unknown mode.")
	}
	return nil
}

func subStr(c *shared.Context, name string) string {
	if o := subOpt(c, name); o != nil {
		return o.StringValue()
	}
	return ""
}

// --- level rewards ---

func (m *Module) roleCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "level-role",
			Description:              "Manage level-reward roles",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageRoles),
			Options: []*discordgo.ApplicationCommandOption{
				subCmd("add", "Grant a role when members reach a level",
					intOpt("level", "The level threshold", true), roleOpt("role", "The role to grant", true)),
				subCmd("remove", "Remove the reward at a level", intOpt("level", "The level threshold", true)),
				subCmd("list", "List configured level rewards"),
			},
		},
		Handler: m.handleRole,
	}
}

func (m *Module) handleRole(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageRoles); err != nil {
		return err
	}
	switch subName(c) {
	case "add":
		return m.handleRoleAdd(c)
	case "remove":
		return m.handleRoleRemove(c)
	case "list":
		rewards, err := m.svc.Rewards(c.Ctx, c.GuildID())
		if err != nil {
			return err
		}
		return c.Reply(&discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{rewardsEmbed(rewards)},
		}, true)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleRoleAdd(c *shared.Context) error {
	level := int(subInt(c, "level"))
	if level < 1 || level > 1000 {
		return shared.UserErr("Level must be between 1 and 1000.")
	}
	role := subRole(c, "role")
	if role == nil {
		return shared.UserErr("Pick a role.")
	}
	if role.Managed || role.ID == c.GuildID() {
		return shared.UserErr("That role can't be assigned as a reward.")
	}
	if err := m.svc.AddReward(c.Ctx, c.GuildID(), level, role.ID); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Reward added", fmt.Sprintf("Members reaching **level %d** now get <@&%s>.", level, role.ID))},
	}, true)
}

func (m *Module) handleRoleRemove(c *shared.Context) error {
	level := int(subInt(c, "level"))
	removed, err := m.svc.RemoveReward(c.Ctx, c.GuildID(), level)
	if err != nil {
		return err
	}
	if !removed {
		return shared.UserErr("No reward configured for level %d.", level)
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Reward removed", fmt.Sprintf("Level %d no longer grants a role.", level))},
	}, true)
}
