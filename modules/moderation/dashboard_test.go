package moderation

import (
	"context"
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/shared"
)

func TestToModCase(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	expires := created.Add(2 * time.Hour)
	c := &Case{
		CaseNumber:  7,
		Action:      ActionBan,
		TargetID:    123456789012345678,
		ModeratorID: 987654321098765432,
		Reason:      "spam",
		Active:      true,
		DurationMS:  expires.Sub(created).Milliseconds(),
		ExpiresAt:   expires,
		CreatedAt:   created,
	}
	mc := toModCase(c)
	if mc.Number != 7 || mc.Action != ActionBan {
		t.Fatalf("number/action wrong: %+v", mc)
	}
	if mc.TargetID != "123456789012345678" || mc.ModeratorID != "987654321098765432" {
		t.Errorf("ids wrong: %+v", mc)
	}
	if mc.Reason != "spam" || !mc.Active {
		t.Errorf("reason/active wrong: %+v", mc)
	}
	if !mc.ExpiresAt.Equal(expires) || !mc.CreatedAt.Equal(created) {
		t.Errorf("timestamps wrong: %+v", mc)
	}
}

func TestToModCaseSystemModerator(t *testing.T) {
	// A system/automatic case (moderator 0) should surface an empty moderator id.
	mc := toModCase(&Case{CaseNumber: 1, Action: ActionUnban, TargetID: 1, ModeratorID: 0})
	if mc.ModeratorID != "" {
		t.Errorf("system moderator should be empty, got %q", mc.ModeratorID)
	}
	if !mc.ExpiresAt.IsZero() {
		t.Errorf("non-temporary case should have zero ExpiresAt, got %v", mc.ExpiresAt)
	}
}

func TestApplyActionValidation(t *testing.T) {
	m := &Module{} // nil deps/svc: every case below must return before touching them
	const target = "123456789012345678"

	cases := []struct {
		name   string
		action string
		in     shared.ModAction
	}{
		{"unsupported action", "nuke", shared.ModAction{TargetID: target}},
		{"empty target", ActionBan, shared.ModAction{TargetID: ""}},
		{"non-snowflake target", ActionBan, shared.ModAction{TargetID: "nope"}},
		{"self target", ActionWarn, shared.ModAction{TargetID: target, ModID: target}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.ApplyAction(context.Background(), tc.action, tc.in)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if _, ok := shared.AsUserError(err); !ok {
				t.Errorf("expected UserError, got %T: %v", err, err)
			}
		})
	}
}
