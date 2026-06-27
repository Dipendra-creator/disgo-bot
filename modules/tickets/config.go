package tickets

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes the core ticket setup (category, staff
// role, log channel) to the web dashboard. Panel placement and the welcome
// message stay on the slash commands.

var _ shared.Configurable = (*Module)(nil)

// ConfigSchema describes the editable ticket fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Tickets",
		Fields: []shared.Field{
			{Key: "category_id", Label: "Ticket category", Type: shared.FieldChannel,
				Help: "Category new ticket channels are created under."},
			{Key: "staff_role_id", Label: "Staff role", Type: shared.FieldRole,
				Help: "Role granted access to every ticket."},
			{Key: "log_channel_id", Label: "Transcript channel", Type: shared.FieldChannel,
				Help: "Where transcripts are posted on close; empty disables them."},
		},
	}
}

// GetConfig returns the guild's current ticket values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"category_id":    idString(set.CategoryID),
		"staff_role_id":  idString(set.StaffRoleID),
		"log_channel_id": idString(set.LogChannelID),
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
		case "category_id":
			next.CategoryID = pid(v.(string))
		case "staff_role_id":
			next.StaffRoleID = pid(v.(string))
		case "log_channel_id":
			next.LogChannelID = pid(v.(string))
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
