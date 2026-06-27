package economy

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configurable implementation — exposes economy settings to the web dashboard.
// It delegates to the existing Service so the in-process settings cache stays
// authoritative (SaveSettings invalidates it).

var _ shared.Configurable = (*Module)(nil)

// Bounds mirror the validation in config_cmds.go.
const (
	cfgMaxName     = 32
	cfgMaxSymbol   = 16
	cfgMaxReward   = 1000000
	cfgMaxCooldown = 86400
)

// ConfigSchema describes the editable economy fields.
func (m *Module) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: m.Name(),
		Title:  "Economy",
		Fields: []shared.Field{
			{Key: "currency_name", Label: "Currency name", Type: shared.FieldString, MaxLen: cfgMaxName},
			{Key: "currency_symbol", Label: "Currency symbol", Type: shared.FieldString, MaxLen: cfgMaxSymbol},
			{Key: "daily_amount", Label: "Daily reward", Type: shared.FieldInt, Min: 1, Max: cfgMaxReward},
			{Key: "work_min", Label: "Work reward (min)", Type: shared.FieldInt, Min: 0, Max: cfgMaxReward},
			{Key: "work_max", Label: "Work reward (max)", Type: shared.FieldInt, Min: 0, Max: cfgMaxReward},
			{Key: "work_cooldown_seconds", Label: "Work cooldown (seconds)", Type: shared.FieldInt,
				Min: 0, Max: cfgMaxCooldown},
			{Key: "starting_balance", Label: "Starting balance", Type: shared.FieldInt, Min: 0, Max: cfgMaxReward},
		},
	}
}

// GetConfig returns the guild's current economy values.
func (m *Module) GetConfig(ctx context.Context, guildID int64) (map[string]any, error) {
	set, err := m.svc.Settings(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"currency_name":         set.CurrencyName,
		"currency_symbol":       set.CurrencySymbol,
		"daily_amount":          int(set.DailyAmount),
		"work_min":              int(set.WorkMin),
		"work_max":              int(set.WorkMax),
		"work_cooldown_seconds": set.WorkCooldownSec,
		"starting_balance":      int(set.StartingBalance),
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
		case "currency_name":
			next.CurrencyName = v.(string)
		case "currency_symbol":
			next.CurrencySymbol = v.(string)
		case "daily_amount":
			next.DailyAmount = int64(v.(int))
		case "work_min":
			next.WorkMin = int64(v.(int))
		case "work_max":
			next.WorkMax = int64(v.(int))
		case "work_cooldown_seconds":
			next.WorkCooldownSec = v.(int)
		case "starting_balance":
			next.StartingBalance = int64(v.(int))
		}
	}
	if next.WorkMax < next.WorkMin {
		return shared.UserErr("Work reward max must be greater than or equal to the min.")
	}
	if next.CurrencyName == "" {
		return shared.UserErr("Currency name can't be empty.")
	}
	if next.CurrencySymbol == "" {
		return shared.UserErr("Currency symbol can't be empty.")
	}
	next.GuildID = guildID
	return m.svc.SaveSettings(ctx, &next)
}
