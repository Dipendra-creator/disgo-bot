package logging

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/pkg/snowflake"
)

// maxFieldLen bounds message content rendered in a log field (Discord caps
// embed field values at 1024 characters).
const maxFieldLen = 1000

// truncate shortens s to at most maxFieldLen runes, appending an ellipsis.
func truncate(s string) string {
	if s == "" {
		return "*empty*"
	}
	r := []rune(s)
	if len(r) <= maxFieldLen {
		return s
	}
	return string(r[:maxFieldLen]) + "…"
}

func userLine(u *discordgo.User) string {
	if u == nil {
		return "unknown user"
	}
	return fmt.Sprintf("%s (%s)", u.Mention(), u.ID)
}

func messageDeleteEmbed(m *discordgo.Message) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorDanger).
		Title("🗑️ Message deleted").
		Field("Channel", fmt.Sprintf("<#%s>", m.ChannelID), true)
	if m.Author != nil {
		e.Field("Author", userLine(m.Author), true)
		e.Footer("User ID: "+m.Author.ID, "")
	}
	e.Field("Content", truncate(m.Content), false)
	return e.Timestamp().Build()
}

func messageEditEmbed(before, after *discordgo.Message) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorWarning).
		Title("✏️ Message edited").
		Field("Channel", fmt.Sprintf("<#%s>", after.ChannelID), true)
	if after.Author != nil {
		e.Field("Author", userLine(after.Author), true)
		e.Footer("User ID: "+after.Author.ID, "")
	}
	if before != nil {
		e.Field("Before", truncate(before.Content), false)
	}
	e.Field("After", truncate(after.Content), false)
	if after.ID != "" {
		e.Field("Jump", fmt.Sprintf("[Go to message](https://discord.com/channels/%s/%s/%s)", after.GuildID, after.ChannelID, after.ID), false)
	}
	return e.Timestamp().Build()
}

func memberJoinEmbed(u *discordgo.User) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title("📥 Member joined").
		Description(userLine(u))
	if u != nil {
		if created, err := snowflake.Timestamp(u.ID); err == nil {
			e.Field("Account created", humanize.RelativeTag(created), true)
		}
		e.Thumbnail(u.AvatarURL("128"))
		e.Footer("User ID: "+u.ID, "")
	}
	return e.Timestamp().Build()
}

func memberLeaveEmbed(m *discordgo.Member) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorMuted).
		Title("📤 Member left")
	if m != nil && m.User != nil {
		e.Description(userLine(m.User))
		e.Thumbnail(m.User.AvatarURL("128"))
		e.Footer("User ID: "+m.User.ID, "")
	}
	if m != nil && !m.JoinedAt.IsZero() {
		e.Field("Joined", humanize.RelativeTag(m.JoinedAt), true)
	}
	if m != nil && len(m.Roles) > 0 {
		e.Field("Roles", rolesMentions(m.Roles), false)
	}
	return e.Timestamp().Build()
}

func banEmbed(u *discordgo.User) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorDanger).
		Title("🔨 Member banned").
		Description(userLine(u)).
		Footer(footerID(u), "").
		Timestamp().Build()
}

func unbanEmbed(u *discordgo.User) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title("♻️ Member unbanned").
		Description(userLine(u)).
		Footer(footerID(u), "").
		Timestamp().Build()
}

func channelEmbed(ch *discordgo.Channel, created bool) *discordgo.MessageEmbed {
	title, color := "🗂️ Channel deleted", ui.ColorDanger
	if created {
		title, color = "🗂️ Channel created", ui.ColorSuccess
	}
	return ui.NewEmbed().
		Color(color).
		Title(title).
		Field("Channel", fmt.Sprintf("#%s (`%s`)", ch.Name, ch.ID), false).
		Timestamp().Build()
}

func roleEmbed(name, id string, created bool) *discordgo.MessageEmbed {
	title, color := "🎭 Role deleted", ui.ColorDanger
	if created {
		title, color = "🎭 Role created", ui.ColorSuccess
	}
	e := ui.NewEmbed().Color(color).Title(title)
	if name != "" {
		e.Field("Role", fmt.Sprintf("%s (`%s`)", name, id), false)
	} else {
		e.Field("Role", "`"+id+"`", false)
	}
	return e.Timestamp().Build()
}

// statusEmbed summarises which categories are routed where.
func statusEmbed(s *Settings) *discordgo.MessageEmbed {
	e := ui.NewEmbed().Color(ui.ColorInfo).Title("📋 Logging configuration")
	for _, cat := range Categories {
		val := "*disabled*"
		if ch := s.channel(cat); ch != 0 {
			val = fmt.Sprintf("<#%s>", sid(ch))
		}
		e.Field(categoryTitle(cat), val, false)
	}
	return e.Footer("disgo • logging", "").Build()
}

func categoryTitle(cat string) string {
	switch cat {
	case CategoryMessage:
		return "📝 Message events (edits, deletions)"
	case CategoryMember:
		return "👤 Member events (joins, leaves)"
	case CategoryServer:
		return "🛡️ Server events (bans, channels, roles)"
	default:
		return cat
	}
}

func rolesMentions(roleIDs []string) string {
	out := ""
	for i, r := range roleIDs {
		if i > 0 {
			out += " "
		}
		out += "<@&" + r + ">"
	}
	if out == "" {
		return "*none*"
	}
	return out
}

func footerID(u *discordgo.User) string {
	if u == nil {
		return ""
	}
	return "User ID: " + u.ID
}
