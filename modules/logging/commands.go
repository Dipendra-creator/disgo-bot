package logging

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// loggingCommand defines /logging with set/disable/status subcommands, gated by
// Manage Server.
func (m *Module) loggingCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "logging",
			Description:              "Configure server audit logging",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set",
					Description: "Route an event category to a log channel",
					Options: []*discordgo.ApplicationCommandOption{
						categoryOpt(),
						channelOpt("channel", "The channel to post logs into", true),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "disable",
					Description: "Stop logging an event category",
					Options:     []*discordgo.ApplicationCommandOption{categoryOpt()},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "status",
					Description: "Show the current logging configuration",
				},
			},
		},
		Handler: m.handleLogging,
	}
}

func (m *Module) handleLogging(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "set":
		return m.handleSet(c)
	case "disable":
		return m.handleDisable(c)
	case "status":
		return m.handleStatus(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleSet(c *shared.Context) error {
	category := subStr(c, "category")
	if !ValidCategory(category) {
		return shared.UserErr("Unknown category.")
	}
	ch := subChannel(c, "channel")
	if ch == nil {
		return shared.UserErr("Pick a channel.")
	}
	if ch.Type != discordgo.ChannelTypeGuildText {
		return shared.UserErr("Log channels must be text channels.")
	}
	if err := m.svc.SetChannel(c.Ctx, c.GuildID(), category, ch.ID); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Logging updated", fmt.Sprintf("%s will now be logged in <#%s>.", categoryTitle(category), ch.ID)),
		},
	}, true)
}

func (m *Module) handleDisable(c *shared.Context) error {
	category := subStr(c, "category")
	if !ValidCategory(category) {
		return shared.UserErr("Unknown category.")
	}
	if err := m.svc.SetChannel(c.Ctx, c.GuildID(), category, ""); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Logging disabled", fmt.Sprintf("%s will no longer be logged.", categoryTitle(category))),
		},
	}, true)
}

func (m *Module) handleStatus(c *shared.Context) error {
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{statusEmbed(set)},
	}, true)
}
