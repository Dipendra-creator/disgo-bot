package tickets

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

func (m *Module) closeCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "close",
			Description: "Close the ticket in this channel",
			Options: []*discordgo.ApplicationCommandOption{
				strOpt("reason", "Reason for closing", false),
			},
		},
		Handler: m.handleClose,
	}
}

func (m *Module) handleClose(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
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

	// Acknowledge before the channel is deleted — a response can't be posted
	// into a channel that no longer exists.
	if err := c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.LoadingEmbed("Closing this ticket…")},
	}, true); err != nil {
		return err
	}
	if _, err := m.svc.CloseTicket(c.Ctx, c.GuildID(), c.Event.ChannelID, c.User(), optStr(c, "reason")); err != nil {
		c.Log.Warn("close ticket failed", zap.Error(err))
	}
	return nil
}

func (m *Module) addCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "ticket-add",
			Description: "Add a user to this ticket",
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The user to add", true),
			},
		},
		Handler: m.handleAdd,
	}
}

func (m *Module) handleAdd(c *shared.Context) error {
	t, set, err := m.ticketHere(c)
	if err != nil {
		return err
	}
	if !m.canManageTicket(c, set, t) {
		return shared.UserErr("Only staff or the ticket opener can add users.")
	}
	user := optUser(c, "user")
	if user == nil {
		return shared.UserErr("You must specify a user.")
	}
	if err := m.svc.AddUser(c.Event.ChannelID, user.ID); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("User added", fmt.Sprintf("%s now has access to this ticket.", user.Mention()))},
	}, false)
}

func (m *Module) removeCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "ticket-remove",
			Description: "Remove a user from this ticket",
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The user to remove", true),
			},
		},
		Handler: m.handleRemove,
	}
}

func (m *Module) handleRemove(c *shared.Context) error {
	t, set, err := m.ticketHere(c)
	if err != nil {
		return err
	}
	if !m.isStaff(c, set) {
		return shared.UserErr("Only staff can remove users from a ticket.")
	}
	user := optUser(c, "user")
	if user == nil {
		return shared.UserErr("You must specify a user.")
	}
	if user.ID == sid(t.OpenerID) {
		return shared.UserErr("You can't remove the ticket's opener.")
	}
	if err := m.svc.RemoveUser(c.Event.ChannelID, user.ID); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("User removed", fmt.Sprintf("%s no longer has access to this ticket.", user.Mention()))},
	}, true)
}

// ticketHere resolves the ticket for the current channel plus guild settings,
// returning friendly errors when used outside a ticket.
func (m *Module) ticketHere(c *shared.Context) (*Ticket, *Settings, error) {
	if c.GuildID() == "" {
		return nil, nil, shared.UserErr("This command can only be used in a server.")
	}
	t, err := m.svc.TicketByChannel(c.Ctx, c.Event.ChannelID)
	if err != nil {
		if errors.Is(err, ErrNoTicket) {
			return nil, nil, shared.UserErr("This isn't a ticket channel.")
		}
		return nil, nil, err
	}
	if t.Status == StatusClosed {
		return nil, nil, shared.UserErr("This ticket is closed.")
	}
	set, _ := m.svc.Settings(c.Ctx, c.GuildID())
	return t, set, nil
}
