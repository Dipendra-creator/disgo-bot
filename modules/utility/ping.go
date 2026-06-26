package utility

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) pingCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "ping",
			Description: "Check the bot's gateway and round-trip latency",
		},
		Handler: m.handlePing,
	}
}

func (m *Module) handlePing(c *shared.Context) error {
	start := time.Now()
	if err := c.Defer(false); err != nil {
		return err
	}
	rtt := time.Since(start)
	gateway := c.Session.HeartbeatLatency()

	embed := ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title(ui.EmojiSparkles+" Pong!").
		Field("Gateway", fmt.Sprintf("`%d ms`", gateway.Milliseconds()), true).
		Field("Round-trip", fmt.Sprintf("`%d ms`", rtt.Milliseconds()), true).
		Footer("disgo", "").
		Timestamp().
		Build()

	_, err := c.Edit(&discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}
