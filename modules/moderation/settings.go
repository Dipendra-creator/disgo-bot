package moderation

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) modlogCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "modlog",
			Description:              "Set the channel where moderation actions are logged",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionChannel,
					Name:         "channel",
					Description:  "The text channel to post the mod-log into",
					Required:     true,
					ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				},
			},
		},
		Handler: m.handleModLog,
	}
}

func (m *Module) handleModLog(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	ch := optChannel(c, "channel")
	if ch == nil {
		return shared.UserErr("Pick a channel.")
	}
	if ch.Type != discordgo.ChannelTypeGuildText {
		return shared.UserErr("The mod-log must be a text channel.")
	}
	if err := m.svc.SetModLog(c.Ctx, pid(c.GuildID()), pid(ch.ID)); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Mod-log set", fmt.Sprintf("Moderation actions will be logged in <#%s>.", ch.ID)),
		},
	}, true)
}
