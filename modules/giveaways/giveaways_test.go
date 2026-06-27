package giveaways

import "testing"

func TestDrawWinners(t *testing.T) {
	entrants := []int64{1, 2, 3, 4, 5}

	got := drawWinners(entrants, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	seen := map[int64]bool{}
	for _, w := range got {
		if seen[w] {
			t.Fatalf("winner %d drawn twice", w)
		}
		seen[w] = true
		if w < 1 || w > 5 {
			t.Fatalf("winner %d not from the entrant pool", w)
		}
	}

	// Asking for more winners than entrants yields everyone, no duplicates.
	all := drawWinners(entrants, 10)
	if len(all) != len(entrants) {
		t.Fatalf("over-draw len = %d, want %d", len(all), len(entrants))
	}

	if got := drawWinners(nil, 3); got != nil {
		t.Fatalf("empty pool must return nil, got %v", got)
	}
	if got := drawWinners(entrants, 0); got != nil {
		t.Fatalf("n=0 must return nil, got %v", got)
	}
}

func TestJoinAndWinnerList(t *testing.T) {
	joined := joinIDs([]int64{10, 20, 30})
	if joined != "10,20,30" {
		t.Fatalf("joinIDs = %q, want \"10,20,30\"", joined)
	}
	g := &Giveaway{WinnerIDs: joined}
	wl := g.winnerList()
	if len(wl) != 3 || wl[0] != "10" || wl[2] != "30" {
		t.Fatalf("winnerList = %v", wl)
	}
	empty := &Giveaway{}
	if empty.winnerList() != nil {
		t.Fatal("empty WinnerIDs must yield a nil list")
	}
}

func TestMentions(t *testing.T) {
	if got := mentions([]string{"1", "2"}); got != "<@1>, <@2>" {
		t.Fatalf("mentions = %q", got)
	}
	if got := mentions(nil); got != "" {
		t.Fatalf("mentions(nil) = %q, want empty", got)
	}
}

func TestIDRoundTrip(t *testing.T) {
	const raw = "1234567890123456789"
	if got := sid(pid(raw)); got != raw {
		t.Fatalf("sid(pid(%q)) = %q, want %q", raw, got, raw)
	}
}
