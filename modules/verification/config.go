package verification

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes verification settings to the web
// dashboard. It delegates to the existing Service so the in-process settings
// cache stays authoritative (SaveSettings invalidates it). Panel references are
// managed by /verify-panel and are intentionally not editable here.

var _ shared.Configurable = (*Module)(nil)

const (
	cfgMaxMessage = 500
	cfgMaxButton  = 80 // Discord's button-label ceiling
)

// ConfigSchema describes the editable verification fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Verification",
		Fields: []shared.Field{
			{Key: "enabled", Label: "Enabled", Type: shared.FieldBool,
				Help: "Allow members to verify via the panel button."},
			{Key: "role_id", Label: "Verified role", Type: shared.FieldRole,
				Help: "Role granted on verification."},
			{Key: "log_channel_id", Label: "Log channel", Type: shared.FieldChannel,
				Help: "Where first-time verifications are recorded; empty disables logging."},
			{Key: "message", Label: "Panel message", Type: shared.FieldString, MaxLen: cfgMaxMessage},
			{Key: "button_label", Label: "Button label", Type: shared.FieldString, MaxLen: cfgMaxButton},
		},
	}
}

// GetConfig returns the guild's current verification values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"enabled":        set.Enabled,
		"role_id":        idString(set.RoleID),
		"log_channel_id": idString(set.LogChannelID),
		"message":        set.Message,
		"button_label":   set.ButtonLabel,
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
		case "role_id":
			next.RoleID = pid(v.(string))
		case "log_channel_id":
			next.LogChannelID = pid(v.(string))
		case "message":
			next.Message = v.(string)
		case "button_label":
			next.ButtonLabel = v.(string)
		}
	}
	if next.Enabled && next.RoleID == 0 {
		return shared.UserErr("Set a verified role before enabling verification.")
	}
	if next.ButtonLabel == "" {
		next.ButtonLabel = defaultButtonLabel
	}
	if next.Message == "" {
		next.Message = defaultMessage
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
