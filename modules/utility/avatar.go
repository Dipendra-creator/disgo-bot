package utility

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// avatarSizes are the selectable resolutions, in display order.
var avatarSizes = []struct{ size, label string }{
	{"256", "Small"},
	{"1024", "Medium"},
	{"4096", "Large"},
}

func (m *Module) avatarCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "avatar",
			Description: "Show a user's avatar",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user (defaults to you)",
					Required:    false,
				},
			},
		},
		Handler: m.handleAvatar,
	}
}

func (m *Module) handleAvatar(c *shared.Context) error {
	user, _ := m.resolveTarget(c)
	if user == nil {
		return shared.UserErr("Couldn't resolve that user.")
	}
	return c.Reply(m.avatarData(user, "1024"), false)
}

// handleAvatarSize re-renders the avatar at a new resolution when a size button
// is clicked. Args: [userID, size].
func (m *Module) handleAvatarSize(c *shared.Context) error {
	if len(c.Args) < 2 {
		return shared.UserErr("Malformed request.")
	}
	userID, size := c.Args[0], c.Args[1]

	user, err := c.Session.User(userID)
	if err != nil {
		return shared.UserErr("Couldn't fetch that user.")
	}
	return c.Update(m.avatarData(user, size))
}

// avatarData builds the avatar embed plus size/open buttons for the given
// resolution, shared by the command and the size buttons.
func (m *Module) avatarData(user *discordgo.User, size string) *discordgo.InteractionResponseData {
	url := user.AvatarURL(size)

	embed := ui.NewEmbed().
		Color(ui.ColorPrimary).
		Title(user.Username+"'s avatar").
		Image(url).
		Footer("disgo", "").
		Timestamp().
		Build()

	buttons := make([]discordgo.MessageComponent, 0, len(avatarSizes)+1)
	for _, s := range avatarSizes {
		btn := ui.SecondaryButton(shared.BuildID(m.Name(), "avatar", user.ID, s.size), s.label, "")
		if s.size == size {
			btn.Style = discordgo.PrimaryButton
			btn.Disabled = true
		}
		buttons = append(buttons, btn)
	}
	buttons = append(buttons, ui.LinkButton("Open", user.AvatarURL("4096"), ""))

	return &discordgo.InteractionResponseData{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{ui.Row(buttons...)},
	}
}
