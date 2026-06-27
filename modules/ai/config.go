package ai

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes the assistant channel and system-prompt
// override to the web dashboard. It delegates to the existing Service setters,
// each of which invalidates the in-process settings cache.

var _ shared.Configurable = (*Module)(nil)

// ConfigSchema describes the editable AI fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "AI Assistant",
		Fields: []shared.Field{
			{Key: "assistant_channel_id", Label: "Assistant channel", Type: shared.FieldChannel,
				Help: "Channel the bot replies in to every message; empty disables it."},
			{Key: "system_prompt", Label: "System prompt", Type: shared.FieldString, MaxLen: maxSystemLen,
				Help: "Custom instructions for the assistant; empty uses the default."},
		},
	}
}

// GetConfig returns the guild's current AI values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"assistant_channel_id": idString(set.AssistantChannelID),
		"system_prompt":        set.SystemPrompt,
	}, nil
}

// SetConfig applies a validated partial patch and persists it.
func (m *Module) SetConfig(ctx context.Context, guildID int64, patch map[string]any) error {
	norm, err := m.ConfigSchema().Normalize(patch)
	if err != nil {
		return shared.UserErr("%s", err.Error())
	}
	gid := sid(guildID)
	if v, ok := norm["assistant_channel_id"]; ok {
		if err := m.svc.SetChannel(ctx, gid, pid(v.(string))); err != nil {
			return err
		}
	}
	if v, ok := norm["system_prompt"]; ok {
		if err := m.svc.SetSystem(ctx, gid, v.(string)); err != nil {
			return err
		}
	}
	return nil
}

// idString renders a stored snowflake for the dashboard ("" when unset).
func idString(id int64) string {
	if id == 0 {
		return ""
	}
	return sid(id)
}
