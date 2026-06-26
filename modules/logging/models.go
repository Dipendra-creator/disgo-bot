package logging

import (
	"time"

	"github.com/uptrace/bun"
)

// Category identifies a group of loggable events routed to a single channel.
const (
	CategoryMessage = "message" // message edits and deletions
	CategoryMember  = "member"  // member joins and leaves
	CategoryServer  = "server"  // bans/unbans, channel and role changes
)

// Categories lists every valid logging category.
var Categories = []string{CategoryMessage, CategoryMember, CategoryServer}

// ValidCategory reports whether name is a known category.
func ValidCategory(name string) bool {
	for _, c := range Categories {
		if c == name {
			return true
		}
	}
	return false
}

// Settings routes each logging category to a channel for one guild. A zero
// channel ID means that category is disabled.
type Settings struct {
	bun.BaseModel `bun:"table:logging_settings,alias:lg"`

	GuildID          int64     `bun:"guild_id,pk"`
	MessageChannelID int64     `bun:"message_channel_id,notnull"`
	MemberChannelID  int64     `bun:"member_channel_id,notnull"`
	ServerChannelID  int64     `bun:"server_channel_id,notnull"`
	CreatedAt        time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt        time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// channel returns the configured channel ID for a category (0 when disabled).
func (s *Settings) channel(category string) int64 {
	switch category {
	case CategoryMessage:
		return s.MessageChannelID
	case CategoryMember:
		return s.MemberChannelID
	case CategoryServer:
		return s.ServerChannelID
	default:
		return 0
	}
}
