package leveling

import (
	"context"
	"testing"

	"github.com/dipu-sharma/disgo-bot/shared"
)

func TestIsSnowflake(t *testing.T) {
	for _, s := range []string{"1", "962587206604685342"} {
		if !isSnowflake(s) {
			t.Errorf("isSnowflake(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "0", "-3", "x9", "12.3"} {
		if isSnowflake(s) {
			t.Errorf("isSnowflake(%q) = true, want false", s)
		}
	}
}

// SetReward validates input before touching the service, so a nil-service
// Module still rejects bad level/role values with a UserError.
func TestSetRewardValidation(t *testing.T) {
	m := &Module{}
	ctx := context.Background()

	cases := []struct {
		name  string
		level int
		role  string
	}{
		{"level zero", 0, "123"},
		{"level too high", maxRewardLevel + 1, "123"},
		{"empty role", 5, ""},
		{"bad role", 5, "abc"},
		{"zero role", 5, "0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := m.SetReward(ctx, 1, c.level, c.role)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if _, ok := shared.AsUserError(err); !ok {
				t.Errorf("expected UserError, got %T: %v", err, err)
			}
		})
	}
}
