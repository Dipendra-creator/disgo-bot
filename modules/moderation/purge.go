package moderation

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) purgeCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "purge",
			Description:              "Bulk-delete recent messages in this channel",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageMessages),
			Options: []*discordgo.ApplicationCommandOption{
				func() *discordgo.ApplicationCommandOption {
					o := intOpt("count", "How many recent messages to scan (1-100)", true)
					o.MinValue = floatPtr(1)
					o.MaxValue = 100
					return o
				}(),
				userOpt("user", "Only delete messages from this user", false),
			},
		},
		Handler: m.handlePurge,
	}
}

func (m *Module) handlePurge(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageMessages); err != nil {
		return err
	}
	count := int(optInt(c, "count"))
	if count < 1 || count > 100 {
		return shared.UserErr("Choose a count between 1 and 100.")
	}
	var filter string
	if u := optUser(c, "user"); u != nil {
		filter = u.ID
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	n, err := m.svc.Purge(c.Event.ChannelID, count, filter)
	if err != nil {
		return shared.UserErr("Couldn't delete messages — they may be older than 14 days.")
	}
	embed := ui.SuccessEmbed("Purged", fmt.Sprintf("Deleted **%d** message(s).", n))
	_, err = c.Edit(&discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}
