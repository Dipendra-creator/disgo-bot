package tickets

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) setupCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "ticket-setup",
			Description:              "Configure the ticket system for this server",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				channelOpt("category", "Category new ticket channels are created under", true, discordgo.ChannelTypeGuildCategory),
				roleOpt("staff_role", "Role granted access to every ticket", false),
				channelOpt("log_channel", "Channel where transcripts are posted on close", false, discordgo.ChannelTypeGuildText),
			},
		},
		Handler: m.handleSetup,
	}
}

func (m *Module) handleSetup(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	cat := optChannel(c, "category")
	if cat == nil || cat.Type != discordgo.ChannelTypeGuildCategory {
		return shared.UserErr("Pick a category for ticket channels.")
	}

	var staffID int64
	if role := optRole(c, "staff_role"); role != nil {
		staffID = pid(role.ID)
	}
	var logID int64
	if lc := optChannel(c, "log_channel"); lc != nil {
		if lc.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("The log channel must be a text channel.")
		}
		logID = pid(lc.ID)
	}

	if err := m.svc.Setup(c.Ctx, c.GuildID(), pid(cat.ID), staffID, logID); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Category: **%s**\n", cat.Name)
	if staffID != 0 {
		fmt.Fprintf(&b, "Staff role: <@&%s>\n", sid(staffID))
	} else {
		b.WriteString("Staff role: *none (Manage Channels required)*\n")
	}
	if logID != 0 {
		fmt.Fprintf(&b, "Transcript log: <#%s>\n", sid(logID))
	} else {
		b.WriteString("Transcript log: *none*\n")
	}
	b.WriteString("\nPost a panel with `/ticket-panel`.")

	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Tickets configured", b.String())},
	}, true)
}

func (m *Module) panelCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "ticket-panel",
			Description:              "Post a ticket panel with an Open Ticket button",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				channelOpt("channel", "Channel to post the panel in (defaults to here)", false, discordgo.ChannelTypeGuildText),
				strOpt("title", "Panel title", false),
				strOpt("description", "Panel description", false),
			},
		},
		Handler: m.handlePanel,
	}
}

func (m *Module) handlePanel(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}

	target := c.Event.ChannelID
	if ch := optChannel(c, "channel"); ch != nil {
		target = ch.ID
	}
	title := strings.TrimSpace(optStr(c, "title"))
	if title == "" {
		title = "Need help?"
	}
	desc := strings.TrimSpace(optStr(c, "description"))
	if desc == "" {
		desc = "Click the button below to open a private ticket with our staff team."
	}

	if err := m.svc.PostPanel(c.Ctx, c.GuildID(), target, title, desc); err != nil {
		return err
	}

	msg := fmt.Sprintf("Panel posted in <#%s>.", target)
	if set, _ := m.svc.Settings(c.Ctx, c.GuildID()); !set.Configured() {
		msg += "\n\n⚠️ Tickets aren't configured yet — run `/ticket-setup` so the button works."
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Panel posted", msg)},
	}, true)
}
