package logging

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
			m.deps.Log.Error("logging handler panic", zap.String("event", name), zap.Any("panic", r))
		}
	}()
	fn()
}

func (m *Module) onMessageDelete(_ *discordgo.Session, e *discordgo.MessageDelete) {
	m.guard("MessageDelete", func() {
		if e.GuildID == "" {
			return
		}
		msg := e.Message
		if e.BeforeDelete != nil {
			msg = e.BeforeDelete // cached copy carries author + content
		}
		if msg.Author != nil && (msg.Author.Bot || msg.Author.ID == m.svc.botUserID()) {
			return
		}
		m.svc.emit(e.GuildID, CategoryMessage, messageDeleteEmbed(msg))
	})
}

func (m *Module) onMessageUpdate(_ *discordgo.Session, e *discordgo.MessageUpdate) {
	m.guard("MessageUpdate", func() {
		if e.GuildID == "" || e.Message == nil {
			return
		}
		if e.Author != nil && (e.Author.Bot || e.Author.ID == m.svc.botUserID()) {
			return
		}
		// Embed-only updates (link unfurls) carry no content change.
		if e.Content == "" {
			return
		}
		if e.BeforeUpdate != nil && e.BeforeUpdate.Content == e.Content {
			return
		}
		m.svc.emit(e.GuildID, CategoryMessage, messageEditEmbed(e.BeforeUpdate, e.Message))
	})
}

func (m *Module) onMemberAdd(_ *discordgo.Session, e *discordgo.GuildMemberAdd) {
	m.guard("GuildMemberAdd", func() {
		if e.Member == nil {
			return
		}
		m.svc.emit(e.GuildID, CategoryMember, memberJoinEmbed(e.User))
	})
}

func (m *Module) onMemberRemove(_ *discordgo.Session, e *discordgo.GuildMemberRemove) {
	m.guard("GuildMemberRemove", func() {
		if e.Member == nil {
			return
		}
		m.svc.emit(e.GuildID, CategoryMember, memberLeaveEmbed(e.Member))
	})
}

func (m *Module) onBanAdd(_ *discordgo.Session, e *discordgo.GuildBanAdd) {
	m.guard("GuildBanAdd", func() {
		m.svc.emit(e.GuildID, CategoryServer, banEmbed(e.User))
	})
}

func (m *Module) onBanRemove(_ *discordgo.Session, e *discordgo.GuildBanRemove) {
	m.guard("GuildBanRemove", func() {
		m.svc.emit(e.GuildID, CategoryServer, unbanEmbed(e.User))
	})
}

func (m *Module) onChannelCreate(_ *discordgo.Session, e *discordgo.ChannelCreate) {
	m.guard("ChannelCreate", func() {
		if e.Channel == nil {
			return
		}
		m.svc.emit(e.GuildID, CategoryServer, channelEmbed(e.Channel, true))
	})
}

func (m *Module) onChannelDelete(_ *discordgo.Session, e *discordgo.ChannelDelete) {
	m.guard("ChannelDelete", func() {
		if e.Channel == nil {
			return
		}
		m.svc.emit(e.GuildID, CategoryServer, channelEmbed(e.Channel, false))
	})
}

func (m *Module) onRoleCreate(_ *discordgo.Session, e *discordgo.GuildRoleCreate) {
	m.guard("GuildRoleCreate", func() {
		if e.GuildRole == nil || e.Role == nil {
			return
		}
		m.svc.emit(e.GuildID, CategoryServer, roleEmbed(e.Role.Name, e.Role.ID, true))
	})
}

func (m *Module) onRoleDelete(_ *discordgo.Session, e *discordgo.GuildRoleDelete) {
	m.guard("GuildRoleDelete", func() {
		m.svc.emit(e.GuildID, CategoryServer, roleEmbed("", e.RoleID, false))
	})
}
