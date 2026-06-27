package ai

import (
	"context"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Gateway event handlers run in discordgo's own goroutines, outside the
// interaction router — so each is wrapped in a recover guard.

func (m *Module) guard(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			m.deps.Log.Error("ai handler panic", zap.String("event", name), zap.Any("panic", r))
		}
	}()
	fn()
}

// onMessageCreate drives the opt-in assistant channel.
func (m *Module) onMessageCreate(_ *discordgo.Session, e *discordgo.MessageCreate) {
	m.guard("MessageCreate", func() { m.handleAssistant(e) })
}

func (m *Module) handleAssistant(e *discordgo.MessageCreate) {
	if e.GuildID == "" || e.Author == nil || e.Author.Bot || e.Author.System {
		return
	}
	if !m.svc.Ready() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), askTimeout+5*time.Second)
	defer cancel()

	set, err := m.svc.getSettings(ctx, pid(e.GuildID))
	if err != nil || set.AssistantChannelID == 0 || sid(set.AssistantChannelID) != e.ChannelID {
		return
	}
	content := strings.TrimSpace(e.Content)
	if content == "" {
		return
	}
	// Silently ignore messages from members still within their cooldown.
	if m.svc.rateLimited(ctx, e.GuildID, e.Author.ID) {
		return
	}

	_ = m.deps.Session.ChannelTyping(e.ChannelID)
	answer, err := m.svc.Ask(ctx, e.GuildID, content)
	if err != nil {
		m.deps.Log.Warn("assistant reply failed", zap.Error(err), zap.String("guild", e.GuildID))
		return
	}

	_, err = m.deps.Session.ChannelMessageSendComplex(e.ChannelID, &discordgo.MessageSend{
		Content:         answer,
		Reference:       e.Reference(),
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	})
	if err != nil {
		m.deps.Log.Warn("assistant send failed", zap.Error(err), zap.String("channel", e.ChannelID))
	}
}
