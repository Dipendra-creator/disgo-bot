package verification

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// handleVerify grants the configured role when a member clicks the panel button.
func (m *Module) handleVerify(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("Verification can only be used in a server.")
	}
	if err := c.Defer(true); err != nil {
		return err
	}
	res, err := m.svc.Verify(c.Ctx, c.GuildID(), c.Member(), c.User())
	if err != nil {
		return err
	}

	var embed *discordgo.MessageEmbed
	if res.AlreadyHadRole {
		embed = ui.EmptyEmbed("Already verified", fmt.Sprintf("You already have <@&%s>.", sid(res.RoleID)))
	} else {
		embed = ui.SuccessEmbed("Verified", fmt.Sprintf("You've been granted <@&%s>. Welcome!", sid(res.RoleID)))
	}
	_, err = c.Edit(&discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}
