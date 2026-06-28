package web

import "testing"

func TestCreateGiveawayRequestInput(t *testing.T) {
	b := createGiveawayRequest{
		ChannelID:  "555",
		Prize:      "Nitro",
		DurationMS: 3600000,
		Winners:    3,
	}
	in := b.input()
	if in.ChannelID != "555" || in.Prize != "Nitro" || in.DurationMS != 3600000 || in.Winners != 3 {
		t.Fatalf("unexpected mapping: %+v", in)
	}
}

func TestSessUID(t *testing.T) {
	if got := sessUID(&Session{UserID: "12345"}); got != 12345 {
		t.Fatalf("sessUID = %d, want 12345", got)
	}
	if got := sessUID(&Session{UserID: ""}); got != 0 {
		t.Fatalf("sessUID empty = %d, want 0", got)
	}
}
