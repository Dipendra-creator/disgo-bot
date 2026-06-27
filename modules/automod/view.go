package automod

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
)

// contentSnippet caps how much of an offending message is echoed into the log.
const contentSnippet = 300

// actionLabel renders an action as a past-tense outcome for the log embed.
func actionLabel(action string) string {
	if action == ActionTimeout {
		return "Deleted message + timed out"
	}
	return "Deleted message"
}

// onOff renders a filter toggle with its action and any thresholds.
func onOff(enabled bool, action, extra string) string {
	if !enabled {
		return "🔴 Disabled"
	}
	s := "🟢 " + action
	if extra != "" {
		s += " · " + extra
	}
	return s
}

// violationEmbed describes an enforced automod action for the log channel.
func violationEmbed(mc *discordgo.MessageCreate, v *violation, set *Settings) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorWarning).
		Title("🛡️ AutoMod — "+v.filter).
		Field("Member", fmt.Sprintf("%s (`%s`)", mc.Author.Mention(), mc.Author.ID), true).
		Field("Channel", fmt.Sprintf("<#%s>", mc.ChannelID), true).
		Field("Action", actionLabel(v.action), true).
		Field("Trigger", v.detail, false)

	if v.action == ActionTimeout {
		e.Field("Timeout", fmt.Sprintf("%ds", set.TimeoutSecs), true)
	}
	if c := strings.TrimSpace(mc.Content); c != "" {
		if len(c) > contentSnippet {
			c = c[:contentSnippet] + "…"
		}
		e.Field("Content", c, false)
	}
	return e.Footer("disgo • automod", "").Timestamp().Build()
}

// statusEmbed summarises the automod configuration.
func statusEmbed(set *Settings, wordCount int) *discordgo.MessageEmbed {
	logCh := "*none*"
	if set.LogChannelID != 0 {
		logCh = fmt.Sprintf("<#%s>", sid(set.LogChannelID))
	}
	exempt := "*none*"
	if set.ExemptRoleID != 0 {
		exempt = fmt.Sprintf("<@&%s>", sid(set.ExemptRoleID))
	}
	return ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("🛡️ AutoMod configuration").
		Field("Banned words", onOff(set.WordsEnabled, set.WordsAction, fmt.Sprintf("%s listed", humanize.Comma(wordCount))), false).
		Field("Invite links", onOff(set.InvitesEnabled, set.InvitesAction, ""), false).
		Field("Mass mentions", onOff(set.MentionsEnabled, set.MentionsAction, fmt.Sprintf("≥ %d", set.MentionThreshold)), false).
		Field("Spam", onOff(set.SpamEnabled, set.SpamAction, fmt.Sprintf("%d / %ds", set.SpamCount, set.SpamWindowSecs)), false).
		Field("Log channel", logCh, true).
		Field("Exempt role", exempt, true).
		Field("Timeout", fmt.Sprintf("%ds", set.TimeoutSecs), true).
		Footer("disgo • automod • Manage Messages bypasses all filters", "").Build()
}

// wordsEmbed lists a guild's banned terms.
func wordsEmbed(words []string) *discordgo.MessageEmbed {
	e := ui.NewEmbed().Color(ui.ColorInfo).Title("🛡️ Banned words")
	if len(words) == 0 {
		return e.Description("No banned words yet. Add one with `/automod-words add`.").Build()
	}
	return e.
		Description("`"+strings.Join(words, "`, `")+"`").
		Footer(fmt.Sprintf("disgo • automod · %s words", humanize.Comma(len(words))), "").
		Build()
}
