package leveling

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// onMessageCreate awards XP for qualifying guild messages. It runs in
// discordgo's goroutine, so it is wrapped in a recover guard.
func (m *Module) onMessageCreate(_ *discordgo.Session, e *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			m.deps.Log.Error("leveling handler panic", zap.Any("panic", r))
		}
	}()
	if e.Message == nil || e.GuildID == "" {
		return
	}
	m.svc.HandleMessage(e.GuildID, e.ChannelID, e.Author)
}
