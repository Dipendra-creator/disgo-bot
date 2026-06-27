package moderation

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes moderation settings to the web
// dashboard. It delegates to the existing Service, which upserts the row.

var _ shared.Configurable = (*Module)(nil)

// ConfigSchema describes the editable moderation fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Moderation",
		Fields: []shared.Field{
			{Key: "mod_log_channel_id", Label: "Mod-log channel", Type: shared.FieldChannel,
				Help: "Where moderation actions are logged; empty disables the log."},
			{Key: "dm_on_action", Label: "DM users on action", Type: shared.FieldBool,
				Help: "Notify the affected user when a ban/kick/timeout/warn is applied."},
		},
	}
}

// GetConfig returns the guild's current moderation values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mod_log_channel_id": idString(set.ModLogChannelID),
		"dm_on_action":       set.DMOnAction,
	}, nil
}

// SetConfig applies a validated partial patch and persists it.
func (m *Module) SetConfig(ctx context.Context, guildID int64, patch map[string]any) error {
	norm, err := m.ConfigSchema().Normalize(patch)
	if err != nil {
		return shared.UserErr("%s", err.Error())
	}
	cur, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return err
	}
	next := *cur

	for key, v := range norm {
		switch key {
		case "mod_log_channel_id":
			next.ModLogChannelID = pid(v.(string))
		case "dm_on_action":
			next.DMOnAction = v.(bool)
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
