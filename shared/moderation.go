package shared

import (
	"context"
	"time"
)

// Moderation is an optional contract a Module implements to expose its case
// history and manual actions to the web dashboard. Like Configurable it is a
// purely additive seam: the web layer type-asserts each registered Module to
// Moderation, and modules that don't implement it simply aren't surfaced as a
// management console. Implementations stay transport-agnostic — they never
// import net/http and exchange only the plain types defined here.
type Moderation interface {
	// ListCases returns a page of a guild's moderation cases, newest first,
	// narrowed by the query and bounded by its Limit/Offset.
	ListCases(ctx context.Context, guildID int64, q ModCaseQuery) (ModCasePage, error)
	// EditCaseReason rewrites a case's reason and returns the updated case.
	EditCaseReason(ctx context.Context, guildID, number int64, reason string) (ModCase, error)
	// ApplyAction performs a manual moderation action (ban/kick/timeout/warn)
	// requested from the dashboard, recording the acting dashboard user as the
	// case moderator. It returns a UserError for unsupported actions or guard
	// violations (self/owner/bot targets).
	ApplyAction(ctx context.Context, action string, a ModAction) (ModCase, error)
}

// ModCase is one moderation case in the dashboard's transport-agnostic form.
// IDs are strings to preserve snowflake precision across JSON.
type ModCase struct {
	Number      int64     `json:"number"`
	Action      string    `json:"action"`
	TargetID    string    `json:"target_id"`
	ModeratorID string    `json:"moderator_id"` // "" / "0" = system/automatic
	Reason      string    `json:"reason"`
	Active      bool      `json:"active"`
	DurationMS  int64     `json:"duration_ms"`          // 0 = permanent/none
	ExpiresAt   time.Time `json:"expires_at,omitempty"` // zero when not time-limited
	CreatedAt   time.Time `json:"created_at"`
}

// ModCaseQuery narrows a case listing. Empty string filters match any value;
// Limit/Offset drive pagination (the implementation clamps unreasonable values).
type ModCaseQuery struct {
	TargetID string // snowflake; "" = any target
	Action   string // e.g. "ban"; "" = any action
	Limit    int
	Offset   int
}

// ModCasePage is one page of cases together with the total number of matches
// (ignoring Limit/Offset) so the dashboard can paginate.
type ModCasePage struct {
	Cases []ModCase `json:"cases"`
	Total int       `json:"total"`
}

// ModAction requests a manual moderation action from the dashboard. ModID and
// ModName identify the logged-in dashboard user performing it; they are taken
// from the server-side session, never from the client body.
type ModAction struct {
	GuildID    string
	TargetID   string
	ModID      string
	ModName    string
	Reason     string
	DurationMS int64 // timeout length / temp-ban length; 0 = none/permanent
}
