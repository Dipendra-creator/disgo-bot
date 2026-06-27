package giveaways

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// handleEnter toggles the clicking member's entry for a giveaway.
func (m *Module) handleEnter(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("Giveaways can only be entered in a server.")
	}
	if len(c.Args) == 0 {
		return shared.UserErr("That button is malformed.")
	}
	id, err := strconv.ParseInt(c.Args[0], 10, 64)
	if err != nil {
		return shared.UserErr("That button is malformed.")
	}

	entered, count, err := m.svc.ToggleEntry(c.Ctx, id, c.GuildID(), c.User().ID)
	if err != nil {
		if errors.Is(err, ErrEnded) {
			return shared.UserErr("This giveaway has already ended.")
		}
		if errors.Is(err, ErrNotFound) {
			return shared.UserErr("This giveaway no longer exists.")
		}
		return err
	}

	var embed *discordgo.MessageEmbed
	if entered {
		embed = ui.SuccessEmbed("Entered", fmt.Sprintf("You're in! There are now **%d** entries.", count))
	} else {
		embed = ui.EmptyEmbed("Left", fmt.Sprintf("You've left this giveaway. **%d** entries remain.", count))
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}, true)
}
