package automod

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Gateway event handlers run in discordgo's own goroutines, outside the
// interaction router's middleware — so each is wrapped in a recover guard to
// keep a handler panic from crashing the process.

func (m *Module) guard(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			m.deps.Log.Error("automod handler panic", zap.String("event", name), zap.Any("panic", r))
		}
	}()
	fn()
}

// onMessageCreate feeds every guild message through the automod filters.
func (m *Module) onMessageCreate(_ *discordgo.Session, e *discordgo.MessageCreate) {
	m.guard("MessageCreate", func() { m.svc.HandleMessage(e) })
}
