package verification

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Component custom-ID action (namespace "verification").
const actionVerify = "verify"

func bid(action string) string { return shared.BuildID("verification", action) }

// verifyButton is the green call-to-action on the panel.
func verifyButton(label string) discordgo.Button {
	if label == "" {
		label = defaultButtonLabel
	}
	return ui.SuccessButton(bid(actionVerify), label, "✅")
}

// panelComponents builds the Components-v2 verification panel.
func panelComponents(title, desc, buttonLabel string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		ui.Container(ui.ColorPrimary,
			ui.Text("## "+title),
			ui.Text(desc),
			ui.Separator(),
			ui.Row(verifyButton(buttonLabel)),
		),
	}
}

// verifiedLogEmbed announces a member's first verification to the log channel.
func verifiedLogEmbed(u *discordgo.User) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Author(u.String(), u.AvatarURL("128")).
		Description(fmt.Sprintf("%s verified.", u.Mention())).
		Footer("disgo • verification", "").Timestamp().Build()
}

// statusEmbed summarises the verification configuration.
func statusEmbed(s *Settings, verified int) *discordgo.MessageEmbed {
	state := "🔴 Disabled"
	if s.Configured() {
		state = "🟢 Enabled"
	} else if s.Enabled {
		state = "🟡 Enabled (no role set)"
	}
	role := "*none*"
	if s.RoleID != 0 {
		role = fmt.Sprintf("<@&%s>", sid(s.RoleID))
	}
	logCh := "*none*"
	if s.LogChannelID != 0 {
		logCh = fmt.Sprintf("<#%s>", sid(s.LogChannelID))
	}
	return ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("🛡️ Verification configuration").
		Field("Status", state, true).
		Field("Verified role", role, true).
		Field("Log channel", logCh, true).
		Field("Members verified", humanize.Comma(verified), true).
		Field("Button label", s.ButtonLabel, true).
		Footer("disgo • verification", "").Build()
}
