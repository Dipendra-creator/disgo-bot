package giveaways

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Component custom-ID action (namespace "giveaways").
const actionEnter = "enter"

func bid(action string, args ...string) string {
	return shared.BuildID("giveaways", action, args...)
}

// enterButton is the entry call-to-action, labelled with the live entry count.
func enterButton(id int64, count int) discordgo.Button {
	return ui.SuccessButton(bid(actionEnter, sid(id)), fmt.Sprintf("Enter · %d", count), "🎉")
}

// endedButton is the disabled placeholder shown after a giveaway closes.
func endedButton() discordgo.Button {
	b := ui.SecondaryButton(bid(actionEnter, "0"), "Giveaway ended", "🎉")
	b.Disabled = true
	return b
}

// mentions renders a winner-ID list as user mentions, or "" when empty.
func mentions(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = "<@" + id + ">"
	}
	return strings.Join(out, ", ")
}

// activePanel renders the live giveaway message.
func activePanel(g *Giveaway, count int) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		ui.Container(ui.ColorBrand,
			ui.Text("# 🎉 Giveaway"),
			ui.Text("## "+g.Prize),
			ui.Text(fmt.Sprintf("Ends %s · **%d** winner(s) · hosted by <@%s>",
				humanize.RelativeTag(g.EndsAt), g.Winners, sid(g.HostID))),
			ui.Separator(),
			ui.Row(enterButton(g.ID, count)),
		),
	}
}

// endedPanel renders the giveaway message after the draw.
func endedPanel(g *Giveaway, count int, winners []string) []discordgo.MessageComponent {
	result := "No valid entries — nobody won."
	if len(winners) > 0 {
		result = "Winners: " + mentions(winners)
	}
	return []discordgo.MessageComponent{
		ui.Container(ui.ColorMuted,
			ui.Text("# 🎉 Giveaway ended"),
			ui.Text("## "+g.Prize),
			ui.Text(result),
			ui.Text(fmt.Sprintf("%s entries · hosted by <@%s>", humanize.Comma(count), sid(g.HostID))),
			ui.Separator(),
			ui.Row(endedButton()),
		),
	}
}

// announceText is the winner ping posted in the channel when a giveaway ends.
func announceText(g *Giveaway, winners []string) string {
	if len(winners) == 0 {
		return fmt.Sprintf("🎉 The giveaway for **%s** ended, but there weren't enough entries to pick a winner.", g.Prize)
	}
	return fmt.Sprintf("🎉 Congratulations %s — you won **%s**!", mentions(winners), g.Prize)
}

// rerollText is the winner ping posted when a giveaway is rerolled.
func rerollText(g *Giveaway, winners []string) string {
	return fmt.Sprintf("🔄 Reroll for **%s** — new winner(s): %s", g.Prize, mentions(winners))
}

// listEmbed renders a guild's active giveaways.
func listEmbed(guildName string, rows []Giveaway) *discordgo.MessageEmbed {
	e := ui.NewEmbed().Color(ui.ColorBrand).Title("🎉 " + guildName + " — Active giveaways")
	if len(rows) == 0 {
		return e.Description("No giveaways are running. Start one with `/giveaway start`.").Build()
	}
	var b strings.Builder
	for i := range rows {
		g := &rows[i]
		fmt.Fprintf(&b, "`#%d` **%s** — %d winner(s), ends %s · <#%s>\n",
			g.ID, g.Prize, g.Winners, humanize.RelativeTag(g.EndsAt), sid(g.ChannelID))
	}
	return e.Description(b.String()).Footer("disgo • giveaways", "").Build()
}
