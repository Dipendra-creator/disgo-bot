package utility

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/pkg/snowflake"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) userinfoCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "userinfo",
			Description: "Show information about a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to inspect (defaults to you)",
					Required:    false,
				},
			},
		},
		Handler: m.handleUserinfo,
	}
}

// userinfoContextCommand exposes the same handler via the right-click user menu.
func (m *Module) userinfoContextCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name: "User Info",
			Type: discordgo.UserApplicationCommand,
		},
		Handler: m.handleUserinfo,
	}
}

func (m *Module) handleUserinfo(c *shared.Context) error {
	user, member := m.resolveTarget(c)
	if user == nil {
		return shared.UserErr("Couldn't resolve that user.")
	}

	// Backfill member details from REST when not provided in the interaction.
	if member == nil && c.GuildID() != "" {
		if fetched, err := c.Session.GuildMember(c.GuildID(), user.ID); err == nil {
			member = fetched
		}
	}

	created, _ := snowflake.Timestamp(user.ID)
	e := ui.NewEmbed().
		Color(ui.ColorInfo).
		Author(user.String(), user.AvatarURL("128")).
		Thumbnail(user.AvatarURL("256")).
		Field("User", fmt.Sprintf("<@%s>", user.ID), true).
		Field("ID", "`"+user.ID+"`", true).
		Field("Bot", yesNo(user.Bot), true).
		Field("Account Created",
			humanize.TimeTag(created)+" • "+humanize.RelativeTag(created), false)

	if member != nil {
		if !member.JoinedAt.IsZero() {
			e.Field("Joined Server",
				humanize.TimeTag(member.JoinedAt)+" • "+humanize.RelativeTag(member.JoinedAt), false)
		}
		if len(member.Roles) > 0 {
			e.Field(fmt.Sprintf("Roles [%d]", len(member.Roles)), roleMentions(member.Roles), false)
		}
		if member.Nick != "" {
			e.Field("Nickname", member.Nick, true)
		}
	}

	return c.Reply(e.Footer("disgo", "").Timestamp().Reply(), false)
}

// resolveTarget figures out which user a userinfo/avatar interaction is about:
// the context-menu target, the "user" option, or the invoker as a fallback.
func (m *Module) resolveTarget(c *shared.Context) (*discordgo.User, *discordgo.Member) {
	data := c.Event.ApplicationCommandData()

	// Context-menu (user command) target.
	if data.TargetID != "" && data.Resolved != nil {
		u := data.Resolved.Users[data.TargetID]
		var mem *discordgo.Member
		if data.Resolved.Members != nil {
			mem = data.Resolved.Members[data.TargetID]
		}
		if mem != nil && mem.User == nil {
			mem.User = u
		}
		return u, mem
	}

	// Slash "user" option.
	for _, o := range data.Options {
		if o.Name == "user" {
			u := o.UserValue(c.Session)
			var mem *discordgo.Member
			if data.Resolved != nil && data.Resolved.Members != nil {
				mem = data.Resolved.Members[o.Value.(string)]
			}
			if mem != nil && mem.User == nil {
				mem.User = u
			}
			return u, mem
		}
	}

	return c.User(), c.Member()
}

// roleMentions renders up to 15 role mentions to stay within embed limits.
func roleMentions(ids []string) string {
	const max = 15
	mentions := make([]string, 0, len(ids))
	for i, id := range ids {
		if i >= max {
			mentions = append(mentions, fmt.Sprintf("… +%d more", len(ids)-max))
			break
		}
		mentions = append(mentions, "<@&"+id+">")
	}
	return strings.Join(mentions, " ")
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
