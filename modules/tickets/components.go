package tickets

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// handleOpen creates a ticket when the panel's Open button is clicked.
func (m *Module) handleOpen(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("Tickets can only be opened in a server.")
	}
	if err := c.Defer(true); err != nil {
		return err
	}
	t, err := m.svc.OpenTicket(c.Ctx, c.GuildID(), c.User(), "")
	if err != nil {
		return err
	}
	embed := ui.SuccessEmbed("Ticket opened", fmt.Sprintf("Your ticket is ready: <#%s>", sid(t.ChannelID)))
	_, err = c.Edit(&discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}

// handleClaim assigns the ticket to the clicking staff member.
func (m *Module) handleClaim(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This action can only be used in a server.")
	}
	set, _ := m.svc.Settings(c.Ctx, c.GuildID())
	if !m.isStaff(c, set) {
		return shared.UserErr("Only staff can claim tickets.")
	}
	if _, err := m.svc.ClaimTicket(c.Ctx, c.Event.ChannelID, c.User()); err != nil {
		return err
	}

	// Disable the Claim button in place, preserving the original content/embeds.
	if err := c.Update(&discordgo.InteractionResponseData{
		Content:    c.Event.Message.Content,
		Embeds:     c.Event.Message.Embeds,
		Components: []discordgo.MessageComponent{ui.Row(claimedButton(c.User().Username), closeButton())},
	}); err != nil {
		return err
	}
	_, err := c.Followup(&discordgo.WebhookParams{
		Content:         fmt.Sprintf("✋ Ticket claimed by %s", c.User().Mention()),
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}, false)
	return err
}

// handleCloseRequest shows an ephemeral confirm prompt for the Close button.
func (m *Module) handleCloseRequest(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This action can only be used in a server.")
	}
	set, _ := m.svc.Settings(c.Ctx, c.GuildID())
	t, err := m.svc.TicketByChannel(c.Ctx, c.Event.ChannelID)
	if err != nil {
		if errors.Is(err, ErrNoTicket) {
			return shared.UserErr("This isn't a ticket channel.")
		}
		return err
	}
	if !m.canManageTicket(c, set, t) {
		return shared.UserErr("Only staff or the ticket opener can close this ticket.")
	}
	return c.Reply(closeConfirm(), true)
}

// handleConfirmClose closes and deletes the ticket after confirmation.
func (m *Module) handleConfirmClose(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This action can only be used in a server.")
	}
	set, _ := m.svc.Settings(c.Ctx, c.GuildID())
	t, err := m.svc.TicketByChannel(c.Ctx, c.Event.ChannelID)
	if err != nil {
		if errors.Is(err, ErrNoTicket) {
			return shared.UserErr("This isn't a ticket channel.")
		}
		return err
	}
	if t.Status == StatusClosed {
		return shared.UserErr("This ticket is already closed.")
	}
	if !m.canManageTicket(c, set, t) {
		return shared.UserErr("Only staff or the ticket opener can close this ticket.")
	}

	// Acknowledge (editing the ephemeral prompt) before the channel is deleted.
	if err := c.Update(&discordgo.InteractionResponseData{
		Embeds:     []*discordgo.MessageEmbed{ui.LoadingEmbed("Closing this ticket…")},
		Components: []discordgo.MessageComponent{},
	}); err != nil {
		return err
	}
	_, err = m.svc.CloseTicket(c.Ctx, c.GuildID(), c.Event.ChannelID, c.User(), "")
	return err
}

// handleCancelClose dismisses the close confirmation.
func (m *Module) handleCancelClose(c *shared.Context) error {
	return c.Update(&discordgo.InteractionResponseData{
		Embeds:     []*discordgo.MessageEmbed{ui.EmptyEmbed("Cancelled", "This ticket was not closed.")},
		Components: []discordgo.MessageComponent{},
	})
}
