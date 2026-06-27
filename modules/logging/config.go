package logging

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes the per-category log channels to the web
// dashboard. It delegates to the existing Service (SetChannel invalidates the
// in-process cache). Field keys are the category names.

// ConfigSchema describes the editable logging fields (one channel per category).
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Audit Logging",
		Fields: []shared.Field{
			{Key: CategoryMessage, Label: "Message log channel", Type: shared.FieldChannel,
				Help: "Message edits and deletions. Empty disables. Needs MessageContent intent."},
			{Key: CategoryMember, Label: "Member log channel", Type: shared.FieldChannel,
				Help: "Member joins and leaves. Empty disables. Needs GuildMembers intent."},
			{Key: CategoryServer, Label: "Server log channel", Type: shared.FieldChannel,
				Help: "Bans, channel and role changes. Empty disables."},
		},
	}
}

// GetConfig returns the guild's current channel routing.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		CategoryMessage: channelString(set.MessageChannelID),
		CategoryMember:  channelString(set.MemberChannelID),
		CategoryServer:  channelString(set.ServerChannelID),
	}, nil
}

// SetConfig applies a validated partial patch, routing each provided category.
func (m *Module) SetConfig(ctx context.Context, guildID int64, patch map[string]any) error {
	norm, err := m.ConfigSchema().Normalize(patch)
	if err != nil {
		return shared.UserErr("%s", err.Error())
	}
	gid := sid(guildID)
	for category, v := range norm {
		// v is a snowflake string ("" clears) — SetChannel takes the string form.
		if err := m.svc.SetChannel(ctx, gid, category, v.(string)); err != nil {
			return err
		}
	}
	return nil
}

// channelString renders a stored channel ID for the dashboard ("" when unset).
func channelString(id int64) string {
	if id == 0 {
		return ""
	}
	return sid(id)
}
