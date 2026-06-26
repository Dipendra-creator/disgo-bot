package tickets

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Component custom-ID actions (namespace "tickets").
const (
	actionOpen         = "open"
	actionClaim        = "claim"
	actionClose        = "close"
	actionConfirmClose = "confirm_close"
	actionCancelClose  = "cancel_close"
)

func bid(action string) string { return shared.BuildID("tickets", action) }

func openButton() discordgo.Button  { return ui.PrimaryButton(bid(actionOpen), "Open Ticket", "🎫") }
func claimButton() discordgo.Button { return ui.SuccessButton(bid(actionClaim), "Claim", "✋") }
func closeButton() discordgo.Button { return ui.DangerButton(bid(actionClose), "Close", "🔒") }

func claimedButton(name string) discordgo.Button {
	b := ui.SuccessButton(bid(actionClaim), "Claimed by "+name, "✋")
	b.Disabled = true
	return b
}

// ticketControls is the Claim + Close button row on the welcome message.
func ticketControls() discordgo.ActionsRow { return ui.Row(claimButton(), closeButton()) }

// panelComponents builds the Components-v2 panel posted in a public channel.
func panelComponents(title, desc string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		ui.Container(ui.ColorPrimary,
			ui.Text("## "+title),
			ui.Text(desc),
			ui.Separator(),
			ui.Row(openButton()),
		),
	}
}

// welcomeEmbed greets the opener inside the new ticket channel.
func welcomeEmbed(t *Ticket, opener *discordgo.User) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorPrimary).
		Title(fmt.Sprintf("🎫 Ticket #%d", t.Number)).
		Description("Thanks for reaching out. Describe your issue in detail and a staff member will be with you shortly.")
	if t.Subject != "" {
		e.Field("Subject", t.Subject, false)
	}
	e.Field("Opened by", opener.Mention(), true)
	return e.Footer("disgo • tickets", "").Timestamp().Build()
}

// transcriptEmbed summarises a closed ticket for the log channel.
func transcriptEmbed(t *Ticket, closer *discordgo.User, reason string, messages int) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorInfo).
		Title(fmt.Sprintf("🎫 Ticket #%d closed", t.Number)).
		Field("Opened by", fmt.Sprintf("<@%s>", sid(t.OpenerID)), true)
	if t.ClaimerID != 0 {
		e.Field("Claimed by", fmt.Sprintf("<@%s>", sid(t.ClaimerID)), true)
	}
	if closer != nil {
		e.Field("Closed by", closer.Mention(), true)
	}
	e.Field("Messages", humanize.Comma(messages), true)
	if !t.CreatedAt.IsZero() {
		e.Field("Opened", humanize.RelativeTag(t.CreatedAt), true)
	}
	if reason == "" {
		reason = "*No reason provided.*"
	}
	e.Field("Reason", reason, false)
	return e.Footer("disgo • tickets", "").Timestamp().Build()
}

// closeConfirm is the ephemeral confirm/cancel prompt shown by the Close button.
func closeConfirm() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{ui.WarningEmbed("Close this ticket?", "The channel will be deleted. This can't be undone.")},
		Components: []discordgo.MessageComponent{
			ui.Row(
				ui.DangerButton(bid(actionConfirmClose), "Confirm Close", "🔒"),
				ui.SecondaryButton(bid(actionCancelClose), "Cancel", ""),
			),
		},
	}
}
