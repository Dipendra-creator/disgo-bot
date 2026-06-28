package automod

import (
	"time"

	"github.com/uptrace/bun"
)

// Filter identifiers (also the values of the command's filter-name choice).
const (
	FilterWords    = "words"
	FilterInvites  = "invites"
	FilterMentions = "mentions"
	FilterSpam     = "spam"
)

// Actions a filter can take on a match. The offending message is always deleted;
// the action describes any escalation beyond that.
const (
	ActionDelete  = "delete"  // delete the message only
	ActionTimeout = "timeout" // delete the message and timeout the author
)

// Bounds for configurable thresholds.
const (
	minMentionThreshold = 2
	minSpamCount        = 2
	minSpamWindowSecs   = 1
	maxSpamWindowSecs   = 60
	minTimeoutSecs      = 10
	maxTimeoutSecs      = 2419200 // 28 days — Discord's timeout ceiling
	maxWordLen          = 100
)

// validAction reports whether a is a known action.
func validAction(a string) bool { return a == ActionDelete || a == ActionTimeout }

// Settings is per-guild automod configuration.
type Settings struct {
	bun.BaseModel `bun:"table:automod_settings,alias:am"`

	GuildID      int64 `bun:"guild_id,pk"`
	LogChannelID int64 `bun:"log_channel_id,notnull"`
	ExemptRoleID int64 `bun:"exempt_role_id,notnull"`
	TimeoutSecs  int   `bun:"timeout_secs,notnull"`

	WordsEnabled bool   `bun:"words_enabled,notnull"`
	WordsAction  string `bun:"words_action,notnull"`

	InvitesEnabled bool   `bun:"invites_enabled,notnull"`
	InvitesAction  string `bun:"invites_action,notnull"`

	MentionsEnabled  bool   `bun:"mentions_enabled,notnull"`
	MentionsAction   string `bun:"mentions_action,notnull"`
	MentionThreshold int    `bun:"mention_threshold,notnull"`

	SpamEnabled    bool   `bun:"spam_enabled,notnull"`
	SpamAction     string `bun:"spam_action,notnull"`
	SpamCount      int    `bun:"spam_count,notnull"`
	SpamWindowSecs int    `bun:"spam_window_secs,notnull"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// defaultSettings returns the in-memory defaults for a guild with no stored row.
func defaultSettings(guildID int64) *Settings {
	return &Settings{
		GuildID:          guildID,
		TimeoutSecs:      300,
		WordsAction:      ActionDelete,
		InvitesAction:    ActionDelete,
		MentionsAction:   ActionDelete,
		MentionThreshold: 5,
		SpamAction:       ActionDelete,
		SpamCount:        5,
		SpamWindowSecs:   5,
	}
}

// AnyEnabled reports whether at least one filter is active (the message hot path
// can bail early otherwise).
func (s *Settings) AnyEnabled() bool {
	return s != nil && (s.WordsEnabled || s.InvitesEnabled || s.MentionsEnabled || s.SpamEnabled)
}

// Word is a single banned term for a guild.
type Word struct {
	bun.BaseModel `bun:"table:automod_words,alias:aw"`

	GuildID int64  `bun:"guild_id,notnull"`
	Word    string `bun:"word,notnull"`
}

// Violation is an audit-trail row for one enforced automod action. The author's
// display name is denormalised at write time since the member or message may be
// gone when the dashboard reads the log.
type Violation struct {
	bun.BaseModel `bun:"table:automod_violations,alias:av"`

	ID        int64     `bun:"id,pk,autoincrement"`
	GuildID   int64     `bun:"guild_id,notnull"`
	UserID    int64     `bun:"user_id,notnull"`
	UserName  string    `bun:"user_name,notnull"`
	ChannelID int64     `bun:"channel_id,notnull"`
	Filter    string    `bun:"filter,notnull"`
	Action    string    `bun:"action,notnull"`
	Detail    string    `bun:"detail,notnull"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
}
