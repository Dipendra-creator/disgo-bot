package shared

import (
	"context"
	"time"
)

// AutoMod is an optional contract a Module implements to expose its content
// moderation data to the web dashboard: a banned-word editor and a read-only log
// of enforced violations. Like Configurable, Moderation and the other console
// seams it is purely additive and transport-agnostic — the web layer type-asserts
// each registered Module to AutoMod, and modules that don't implement it simply
// aren't surfaced as a console. Implementations never import net/http and
// exchange only the plain types defined here.
//
// The scalar filter toggles (enable flags, thresholds, log channel) stay on the
// Configurable seam; AutoMod covers only the list-based word CRUD and the
// violation log, which don't fit the schema-driven config form.
type AutoMod interface {
	// ListWords returns the guild's banned terms, alphabetically.
	ListWords(ctx context.Context, guildID int64) ([]string, error)
	// AddWord adds a banned term and returns the updated list. It returns a
	// UserError for blank, over-long or duplicate terms.
	AddWord(ctx context.Context, guildID int64, word string) ([]string, error)
	// RemoveWord deletes a banned term and returns the updated list. It returns
	// a UserError when the term isn't in the list.
	RemoveWord(ctx context.Context, guildID int64, word string) ([]string, error)
	// ListViolations returns a page of the enforced-action log (newest first)
	// together with the total count ignoring the window.
	ListViolations(ctx context.Context, guildID int64, q PageQuery) (ViolationPage, error)
}

// ViolationView is one enforced automod action in the dashboard's
// transport-agnostic form. IDs are strings to preserve snowflake precision
// across JSON.
type ViolationView struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"` // captured at enforcement time
	ChannelID string    `json:"channel_id"`          // "" when unknown
	Filter    string    `json:"filter"`              // words | invites | mentions | spam
	Action    string    `json:"action"`              // delete | timeout
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// ViolationPage is one page of the violation log with the total row count.
type ViolationPage struct {
	Violations []ViolationView `json:"violations"`
	Total      int             `json:"total"`
}
