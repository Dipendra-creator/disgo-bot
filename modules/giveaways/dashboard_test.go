package giveaways

import (
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/shared"
)

func TestValidateGiveaway(t *testing.T) {
	ok := shared.GiveawayInput{ChannelID: "123", Prize: "Nitro", DurationMS: int64(time.Hour / time.Millisecond), Winners: 2}

	cases := []struct {
		name    string
		in      shared.GiveawayInput
		wantErr bool
	}{
		{"valid", ok, false},
		{"no channel", shared.GiveawayInput{ChannelID: "", Prize: "x", DurationMS: ok.DurationMS, Winners: 1}, true},
		{"bad channel", shared.GiveawayInput{ChannelID: "abc", Prize: "x", DurationMS: ok.DurationMS, Winners: 1}, true},
		{"empty prize", shared.GiveawayInput{ChannelID: "123", Prize: "  ", DurationMS: ok.DurationMS, Winners: 1}, true},
		{"zero winners", shared.GiveawayInput{ChannelID: "123", Prize: "x", DurationMS: ok.DurationMS, Winners: 0}, true},
		{"too many winners", shared.GiveawayInput{ChannelID: "123", Prize: "x", DurationMS: ok.DurationMS, Winners: maxWinners + 1}, true},
		{"too short", shared.GiveawayInput{ChannelID: "123", Prize: "x", DurationMS: 1, Winners: 1}, true},
		{"too long", shared.GiveawayInput{ChannelID: "123", Prize: "x", DurationMS: int64(maxDuration/time.Millisecond) + 1000, Winners: 1}, true},
	}
	for _, c := range cases {
		_, _, _, err := validateGiveaway(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v, wantErr=%v", c.name, err, c.wantErr)
		}
		if err != nil {
			if _, isUser := shared.AsUserError(err); !isUser {
				t.Errorf("%s: expected UserError, got %T", c.name, err)
			}
		}
	}
}

func TestIsSnowflake(t *testing.T) {
	for in, want := range map[string]bool{"123": true, "0": false, "-1": false, "": false, "x": false} {
		if got := isSnowflake(in); got != want {
			t.Errorf("isSnowflake(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestClampPage(t *testing.T) {
	l, o := clampPage(shared.PageQuery{Limit: 0, Offset: -3}, 100)
	if l != 100 || o != 0 {
		t.Fatalf("clampPage zero/neg = (%d,%d), want (100,0)", l, o)
	}
	l, o = clampPage(shared.PageQuery{Limit: 25, Offset: 50}, 100)
	if l != 25 || o != 50 {
		t.Fatalf("clampPage in-range = (%d,%d), want (25,50)", l, o)
	}
}
