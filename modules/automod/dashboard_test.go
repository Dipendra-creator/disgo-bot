package automod

import (
	"strings"
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/shared"
)

func TestValidateWord(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"trims and lowercases", "  BadWord  ", "badword", false},
		{"blank", "   ", "", true},
		{"empty", "", "", true},
		{"too long", strings.Repeat("x", maxWordLen+1), "", true},
		{"at limit", strings.Repeat("x", maxWordLen), strings.Repeat("x", maxWordLen), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := validateWord(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("validateWord(%q) = %q, want error", c.in, got)
				}
				if _, ok := shared.AsUserError(err); !ok {
					t.Fatalf("validateWord(%q) error %v, want UserError", c.in, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateWord(%q) unexpected error %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("validateWord(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestToViolationView(t *testing.T) {
	now := time.Now()
	v := toViolationView(&Violation{
		ID:        7,
		UserID:    123,
		UserName:  "spammer",
		ChannelID: 0,
		Filter:    FilterWords,
		Action:    ActionTimeout,
		Detail:    "banned word: x",
		CreatedAt: now,
	})
	if v.ID != 7 || v.UserID != "123" || v.UserName != "spammer" {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.ChannelID != "" {
		t.Fatalf("zero channel should map to empty string, got %q", v.ChannelID)
	}
	if v.Filter != FilterWords || v.Action != ActionTimeout {
		t.Fatalf("unexpected filter/action: %+v", v)
	}

	v2 := toViolationView(&Violation{ChannelID: 555})
	if v2.ChannelID != "555" {
		t.Fatalf("channel id = %q, want 555", v2.ChannelID)
	}
}

func TestClampPage(t *testing.T) {
	cases := []struct {
		name             string
		in               shared.PageQuery
		wantLim, wantOff int
	}{
		{"defaults from zero", shared.PageQuery{}, violationListLimit, 0},
		{"over cap", shared.PageQuery{Limit: 9999}, violationListLimit, 0},
		{"negative offset", shared.PageQuery{Limit: 10, Offset: -5}, 10, 0},
		{"passthrough", shared.PageQuery{Limit: 25, Offset: 50}, 25, 50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lim, off := clampPage(c.in, violationListLimit)
			if lim != c.wantLim || off != c.wantOff {
				t.Fatalf("clampPage(%+v) = (%d,%d), want (%d,%d)", c.in, lim, off, c.wantLim, c.wantOff)
			}
		})
	}
}
