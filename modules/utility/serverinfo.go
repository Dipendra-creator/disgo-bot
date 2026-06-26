package utility

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) serverinfoCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "serverinfo",
			Description: "Show statistics about this server",
		},
		Handler: m.handleServerinfo,
	}
}

// handleServerinfo renders the server overview as a Components-v2 container.
func (m *Module) handleServerinfo(c *shared.Context) error {
	data, err := m.renderServerinfo(c)
	if err != nil {
		return err
	}
	return c.Reply(data, false)
}

// handleServerinfoRefresh re-renders the overview in place when the refresh
// button is clicked, proving component routing through the framework.
func (m *Module) handleServerinfoRefresh(c *shared.Context) error {
	data, err := m.renderServerinfo(c)
	if err != nil {
		return err
	}
	return c.Update(data)
}

// renderServerinfo builds the shared Components-v2 response used by both the
// command and its refresh button.
func (m *Module) renderServerinfo(c *shared.Context) (*discordgo.InteractionResponseData, error) {
	guildID := c.GuildID()
	if guildID == "" {
		return nil, shared.UserErr("Use this command inside a server.")
	}

	st, err := m.svc.GuildStats(c.Session, guildID)
	if err != nil {
		return nil, err
	}

	body := ui.Text(fmt.Sprintf(
		"**👥 Members:** %s\n**💬 Channels:** %s\n**🎭 Roles:** %s",
		humanize.Comma(st.Members), humanize.Comma(st.Channels), humanize.Comma(st.Roles),
	))
	meta := ui.Text(fmt.Sprintf(
		"**👑 Owner:** <@%s>\n**📅 Created:** %s\n**🚀 Boosts:** %s (Tier %d)",
		st.OwnerID, humanize.RelativeTag(st.CreatedAt), humanize.Comma(st.Boosts), st.BoostTier,
	))
	refresh := ui.Row(ui.SecondaryButton(shared.BuildID(m.Name(), "refresh"), "Refresh", ui.EmojiRefresh))

	var header discordgo.MessageComponent
	if st.IconURL != "" {
		header = ui.Section(ui.Thumbnail(st.IconURL), "## "+st.Name, "Server overview • disgo")
	} else {
		header = ui.Text("## " + st.Name + "\nServer overview • disgo")
	}

	container := ui.Container(ui.ColorBrand,
		header,
		ui.Separator(),
		body,
		meta,
		refresh,
	)
	return ui.V2(container), nil
}
