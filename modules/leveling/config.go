package leveling

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes leveling settings to the web dashboard.
// It delegates to the existing Service so the in-process settings cache stays
// authoritative (SaveSettings invalidates it).

// ConfigSchema describes the editable leveling fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Leveling",
		Fields: []shared.Field{
			{Key: "enabled", Label: "Enabled", Type: shared.FieldBool,
				Help: "Award XP for messages."},
			{Key: "xp_cooldown_seconds", Label: "XP cooldown (seconds)", Type: shared.FieldInt,
				Min: 0, Max: 3600, Help: "Minimum gap between XP-earning messages."},
			{Key: "xp_min", Label: "XP per message (min)", Type: shared.FieldInt, Min: 1, Max: 1000},
			{Key: "xp_max", Label: "XP per message (max)", Type: shared.FieldInt, Min: 1, Max: 1000},
			{Key: "announce_enabled", Label: "Announce level-ups", Type: shared.FieldBool},
			{Key: "announce_channel_id", Label: "Announcement channel", Type: shared.FieldChannel,
				Help: "Where level-ups are posted; empty replies in the active channel."},
			{Key: "stack_roles", Label: "Stack reward roles", Type: shared.FieldBool,
				Help: "Keep every earned reward role instead of only the highest."},
		},
	}
}

// GetConfig returns the guild's current leveling values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"enabled":             set.Enabled,
		"xp_cooldown_seconds": set.XPCooldownSeconds,
		"xp_min":              set.XPMin,
		"xp_max":              set.XPMax,
		"announce_enabled":    set.AnnounceEnabled,
		"announce_channel_id": channelString(set.AnnounceChannelID),
		"stack_roles":         set.StackRoles,
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
	next := *cur // copy: never mutate the cached pointer in place

	for key, v := range norm {
		switch key {
		case "enabled":
			next.Enabled = v.(bool)
		case "xp_cooldown_seconds":
			next.XPCooldownSeconds = v.(int)
		case "xp_min":
			next.XPMin = v.(int)
		case "xp_max":
			next.XPMax = v.(int)
		case "announce_enabled":
			next.AnnounceEnabled = v.(bool)
		case "announce_channel_id":
			next.AnnounceChannelID = pid(v.(string))
		case "stack_roles":
			next.StackRoles = v.(bool)
		}
	}
	if next.XPMax < next.XPMin {
		return shared.UserErr("XP max must be greater than or equal to XP min.")
	}
	next.GuildID = guildID
	return m.svc.SaveSettings(ctx, &next)
}

// channelString renders a stored channel ID for the dashboard ("" when unset).
func channelString(id int64) string {
	if id == 0 {
		return ""
	}
	return sid(id)
}
