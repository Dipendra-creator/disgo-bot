package automod

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// AutoMod seam implementation — exposes the banned-word editor and the
// violation log to the web dashboard. The scalar filter settings are handled by
// the Configurable seam in config.go; this file covers only what doesn't fit the
// schema-driven form.

var _ shared.AutoMod = (*Module)(nil)

// violationListLimit caps a single violation-log page regardless of the request.
const violationListLimit = 100

// ListWords returns the guild's banned terms, alphabetically.
func (m *Module) ListWords(ctx context.Context, guildID int64) ([]string, error) {
	return m.svc.ListWords(ctx, sid(guildID))
}

// AddWord validates and adds a banned term, returning the updated list.
func (m *Module) AddWord(ctx context.Context, guildID int64, word string) ([]string, error) {
	norm, err := validateWord(word)
	if err != nil {
		return nil, err
	}
	added, err := m.svc.AddWord(ctx, sid(guildID), norm)
	if err != nil {
		return nil, err
	}
	if !added {
		return nil, shared.UserErr("%q is already on the banned list.", norm)
	}
	return m.svc.ListWords(ctx, sid(guildID))
}

// RemoveWord deletes a banned term, returning the updated list.
func (m *Module) RemoveWord(ctx context.Context, guildID int64, word string) ([]string, error) {
	norm := normalizeWord(word)
	if norm == "" {
		return nil, shared.UserErr("A word is required.")
	}
	removed, err := m.svc.RemoveWord(ctx, sid(guildID), norm)
	if err != nil {
		return nil, err
	}
	if !removed {
		return nil, shared.UserErr("%q is not on the banned list.", norm)
	}
	return m.svc.ListWords(ctx, sid(guildID))
}

// ListViolations returns a page of the enforced-action log (newest first).
func (m *Module) ListViolations(ctx context.Context, guildID int64, q shared.PageQuery) (shared.ViolationPage, error) {
	limit, offset := clampPage(q, violationListLimit)
	rows, total, err := m.svc.ListViolations(ctx, guildID, limit, offset)
	if err != nil {
		return shared.ViolationPage{}, err
	}
	out := make([]shared.ViolationView, len(rows))
	for i := range rows {
		out[i] = toViolationView(&rows[i])
	}
	return shared.ViolationPage{Violations: out, Total: total}, nil
}

// validateWord normalises a term (via normalizeWord) and enforces the
// non-empty and length bounds, returning a UserError on failure.
func validateWord(word string) (string, error) {
	norm := normalizeWord(word)
	if norm == "" {
		return "", shared.UserErr("A word is required.")
	}
	if len(norm) > maxWordLen {
		return "", shared.UserErr("Word must be at most %d characters.", maxWordLen)
	}
	return norm, nil
}

// toViolationView maps a stored row to its transport-agnostic form, rendering
// snowflakes as strings ("" for an unknown channel).
func toViolationView(v *Violation) shared.ViolationView {
	channel := ""
	if v.ChannelID != 0 {
		channel = sid(v.ChannelID)
	}
	return shared.ViolationView{
		ID:        v.ID,
		UserID:    sid(v.UserID),
		UserName:  v.UserName,
		ChannelID: channel,
		Filter:    v.Filter,
		Action:    v.Action,
		Detail:    v.Detail,
		CreatedAt: v.CreatedAt,
	}
}

// clampPage bounds a page window to sane limits.
func clampPage(q shared.PageQuery, max int) (limit, offset int) {
	limit = q.Limit
	if limit <= 0 || limit > max {
		limit = max
	}
	offset = q.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
