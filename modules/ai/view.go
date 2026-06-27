package ai

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
)

// answerEmbed renders a /ask response.
func answerEmbed(prompt, answer, model string) *discordgo.MessageEmbed {
	q := prompt
	if len([]rune(q)) > 120 {
		q = string([]rune(q)[:120]) + "…"
	}
	return ui.NewEmbed().
		Color(ui.ColorBrand).
		Author("🤖 "+q, "").
		Description(answer).
		Footer("disgo • "+model, "").
		Timestamp().Build()
}

// statusEmbed summarises the AI configuration and availability.
func statusEmbed(set *Settings, ready bool, model string) *discordgo.MessageEmbed {
	state := "🔴 Unavailable (no API key configured on this bot)"
	if ready {
		state = "🟢 Available · model `" + model + "`"
	}
	channel := "*none*"
	if set.AssistantChannelID != 0 {
		channel = fmt.Sprintf("<#%s>", sid(set.AssistantChannelID))
	}
	system := "*default*"
	if set.SystemPrompt != "" {
		s := set.SystemPrompt
		if len([]rune(s)) > 200 {
			s = string([]rune(s)[:200]) + "…"
		}
		system = s
	}
	return ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("🤖 AI assistant configuration").
		Field("Status", state, false).
		Field("Assistant channel", channel, true).
		Field("System prompt", system, false).
		Footer("disgo • ai", "").Build()
}
