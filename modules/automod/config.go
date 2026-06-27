package automod

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes automod's scalar settings to the web
// dashboard. The banned-word list is list-based CRUD and stays on the slash
// commands; only scalar toggles/thresholds/channels are editable here.

var _ shared.Configurable = (*Module)(nil)

// Bounds mirror the validation constants in models.go.
const (
	cfgMaxMentionThreshold = 50
	cfgMaxSpamCount        = 50
)

// actionHelp is the shared help text for the delete/timeout enum fields.
const actionHelp = `"delete" or "timeout"`

// ConfigSchema describes the editable automod fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "AutoMod",
		Fields: []shared.Field{
			{Key: "log_channel_id", Label: "Log channel", Type: shared.FieldChannel,
				Help: "Where violations are reported; empty disables logging."},
			{Key: "exempt_role_id", Label: "Exempt role", Type: shared.FieldRole,
				Help: "Members with this role bypass automod."},
			{Key: "timeout_seconds", Label: "Timeout length (seconds)", Type: shared.FieldInt,
				Min: minTimeoutSecs, Max: maxTimeoutSecs},

			{Key: "words_enabled", Label: "Banned-word filter", Type: shared.FieldBool},
			{Key: "words_action", Label: "Banned-word action", Type: shared.FieldString, MaxLen: 16, Help: actionHelp},

			{Key: "invites_enabled", Label: "Invite filter", Type: shared.FieldBool},
			{Key: "invites_action", Label: "Invite action", Type: shared.FieldString, MaxLen: 16, Help: actionHelp},

			{Key: "mentions_enabled", Label: "Mention filter", Type: shared.FieldBool},
			{Key: "mentions_action", Label: "Mention action", Type: shared.FieldString, MaxLen: 16, Help: actionHelp},
			{Key: "mention_threshold", Label: "Mention threshold", Type: shared.FieldInt,
				Min: minMentionThreshold, Max: cfgMaxMentionThreshold},

			{Key: "spam_enabled", Label: "Spam filter", Type: shared.FieldBool},
			{Key: "spam_action", Label: "Spam action", Type: shared.FieldString, MaxLen: 16, Help: actionHelp},
			{Key: "spam_count", Label: "Spam message count", Type: shared.FieldInt,
				Min: minSpamCount, Max: cfgMaxSpamCount},
			{Key: "spam_window_seconds", Label: "Spam window (seconds)", Type: shared.FieldInt,
				Min: minSpamWindowSecs, Max: maxSpamWindowSecs},
		},
	}
}

// GetConfig returns the guild's current automod values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"log_channel_id":      idString(set.LogChannelID),
		"exempt_role_id":      idString(set.ExemptRoleID),
		"timeout_seconds":     set.TimeoutSecs,
		"words_enabled":       set.WordsEnabled,
		"words_action":        set.WordsAction,
		"invites_enabled":     set.InvitesEnabled,
		"invites_action":      set.InvitesAction,
		"mentions_enabled":    set.MentionsEnabled,
		"mentions_action":     set.MentionsAction,
		"mention_threshold":   set.MentionThreshold,
		"spam_enabled":        set.SpamEnabled,
		"spam_action":         set.SpamAction,
		"spam_count":          set.SpamCount,
		"spam_window_seconds": set.SpamWindowSecs,
	}, nil
}

// SetConfig applies a validated partial patch and persists it.
func (m *Module) SetConfig(ctx context.Context, guildID int64, patch map[string]any) error {
	norm, err := m.ConfigSchema().Normalize(patch)
	if err != nil {
		return shared.UserErr("%s", err.Error())
	}
	// Enum validation for the action fields (Normalize only checks length).
	for _, key := range []string{"words_action", "invites_action", "mentions_action", "spam_action"} {
		if v, ok := norm[key]; ok && !validAction(v.(string)) {
			return shared.UserErr("%s must be %q or %q.", key, ActionDelete, ActionTimeout)
		}
	}

	cur, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return err
	}
	next := *cur // copy: never mutate the cached pointer in place

	for key, v := range norm {
		switch key {
		case "log_channel_id":
			next.LogChannelID = pid(v.(string))
		case "exempt_role_id":
			next.ExemptRoleID = pid(v.(string))
		case "timeout_seconds":
			next.TimeoutSecs = v.(int)
		case "words_enabled":
			next.WordsEnabled = v.(bool)
		case "words_action":
			next.WordsAction = v.(string)
		case "invites_enabled":
			next.InvitesEnabled = v.(bool)
		case "invites_action":
			next.InvitesAction = v.(string)
		case "mentions_enabled":
			next.MentionsEnabled = v.(bool)
		case "mentions_action":
			next.MentionsAction = v.(string)
		case "mention_threshold":
			next.MentionThreshold = v.(int)
		case "spam_enabled":
			next.SpamEnabled = v.(bool)
		case "spam_action":
			next.SpamAction = v.(string)
		case "spam_count":
			next.SpamCount = v.(int)
		case "spam_window_seconds":
			next.SpamWindowSecs = v.(int)
		}
	}
	next.GuildID = guildID
	return m.svc.SaveSettings(ctx, &next)
}

// idString renders a stored snowflake for the dashboard ("" when unset).
func idString(id int64) string {
	if id == 0 {
		return ""
	}
	return sid(id)
}
