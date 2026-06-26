package moderation

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/duration"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
)

// actionStyle maps an action to its presentation: a past-tense verb, an accent
// color and a glyph, used consistently across mod-log and DM embeds.
type actionStyle struct {
	verb  string
	color int
	emoji string
}

func styleFor(action string, temporary bool) actionStyle {
	switch action {
	case ActionBan:
		if temporary {
			return actionStyle{"Temporarily Banned", ui.ColorDanger, "🔨"}
		}
		return actionStyle{"Banned", ui.ColorDanger, "🔨"}
	case ActionUnban:
		return actionStyle{"Unbanned", ui.ColorSuccess, "♻️"}
	case ActionKick:
		return actionStyle{"Kicked", ui.ColorDanger, "👢"}
	case ActionTimeout:
		return actionStyle{"Timed Out", ui.ColorWarning, "⏲️"}
	case ActionUntimeout:
		return actionStyle{"Timeout Removed", ui.ColorSuccess, "🔈"}
	case ActionWarn:
		return actionStyle{"Warned", ui.ColorWarning, "⚠️"}
	default:
		return actionStyle{action, ui.ColorInfo, "•"}
	}
}

func reasonOrNone(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "*No reason provided.*"
	}
	return reason
}

// caseEmbed renders a full case card for the public mod-log and for /case.
func caseEmbed(c *Case, target, mod *discordgo.User) *discordgo.MessageEmbed {
	st := styleFor(c.Action, c.Temporary())
	e := ui.NewEmbed().
		Color(st.color).
		Title(fmt.Sprintf("%s %s • Case #%d", st.emoji, st.verb, c.CaseNumber))

	if target != nil {
		e.Field("User", fmt.Sprintf("%s (`%s`)", target.Mention(), target.ID), true)
		e.Thumbnail(target.AvatarURL("128"))
	} else {
		e.Field("User", fmt.Sprintf("<@%s> (`%s`)", sid(c.TargetID), sid(c.TargetID)), true)
	}

	switch {
	case mod != nil:
		e.Field("Moderator", fmt.Sprintf("%s (`%s`)", mod.Mention(), mod.ID), true)
	case c.ModeratorID == 0:
		e.Field("Moderator", "System", true)
	default:
		e.Field("Moderator", fmt.Sprintf("<@%s>", sid(c.ModeratorID)), true)
	}

	if c.Temporary() {
		e.Field("Duration", duration.Human(c.Duration()), true)
		if !c.ExpiresAt.IsZero() {
			e.Field("Expires", humanize.RelativeTag(c.ExpiresAt), true)
		}
	}

	e.Field("Reason", reasonOrNone(c.Reason), false)
	return e.Footer("disgo • moderation", "").Timestamp().Build()
}

// dmEmbed renders the private notice delivered to the affected user.
func dmEmbed(guildName string, c *Case) *discordgo.MessageEmbed {
	st := styleFor(c.Action, c.Temporary())
	e := ui.NewEmbed().
		Color(st.color).
		Title(st.emoji + " You were " + strings.ToLower(st.verb)).
		Description(fmt.Sprintf("This action was taken in **%s**.", guildName))

	if c.Temporary() {
		e.Field("Duration", duration.Human(c.Duration()), true)
		if !c.ExpiresAt.IsZero() {
			e.Field("Expires", humanize.RelativeTag(c.ExpiresAt), true)
		}
	}
	e.Field("Reason", reasonOrNone(c.Reason), false)
	return e.Footer("disgo • moderation", "").Timestamp().Build()
}
