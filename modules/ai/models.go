package ai

import (
	"time"

	"github.com/uptrace/bun"
)

// Bounds and defaults for the assistant.
const (
	maxPromptLen  = 2000 // Discord's own message length cap
	maxSystemLen  = 4000 // admin-set system prompt
	rateWindow    = 12 * time.Second
	defaultSystem = "You are a helpful, concise assistant in a Discord server. " +
		"Keep replies short and friendly, and use Discord markdown where useful."
	maxReplyChars = 1900 // leave headroom under Discord's 2000-char message limit
)

// Settings is per-guild AI configuration.
type Settings struct {
	bun.BaseModel `bun:"table:ai_settings,alias:ai"`

	GuildID            int64     `bun:"guild_id,pk"`
	AssistantChannelID int64     `bun:"assistant_channel_id,notnull"`
	SystemPrompt       string    `bun:"system_prompt,notnull"`
	CreatedAt          time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt          time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

func defaultSettings(guildID int64) *Settings {
	return &Settings{GuildID: guildID}
}

// system returns the effective system prompt (the guild override or the default).
func (s *Settings) system() string {
	if s != nil && s.SystemPrompt != "" {
		return s.SystemPrompt
	}
	return defaultSystem
}
