package tickets

import "testing"

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]string{
		"active":      "active",
		StatusClosed:  StatusClosed,
		"open":        "", // not a filter vocabulary value
		"":            "",
		"garbage":     "",
		StatusClaimed: "", // claimed folds into "active", not a direct filter
	}
	for in, want := range cases {
		if got := normalizeStatus(in); got != want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClampPage(t *testing.T) {
	cases := []struct {
		limit, offset, max    int
		wantLimit, wantOffset int
	}{
		{0, 0, 100, 100, 0},
		{25, 50, 100, 25, 50},
		{500, 0, 100, 100, 0},
		{-5, -5, 100, 100, 0},
	}
	for _, c := range cases {
		gl, go_ := clampPage(c.limit, c.offset, c.max)
		if gl != c.wantLimit || go_ != c.wantOffset {
			t.Errorf("clampPage(%d,%d,%d) = (%d,%d), want (%d,%d)",
				c.limit, c.offset, c.max, gl, go_, c.wantLimit, c.wantOffset)
		}
	}
}

func TestToTicketView(t *testing.T) {
	m := &Module{} // nil deps -> memberName returns "" without touching the session
	tk := &Ticket{
		ID:        7,
		Number:    42,
		ChannelID: 555,
		OpenerID:  111,
		ClaimerID: 0,
		Subject:   "help",
		Status:    StatusOpen,
	}
	v := m.toTicketView("123", tk)
	if v.ID != 7 || v.Number != 42 || v.ChannelID != "555" || v.OpenerID != "111" {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.ClaimerID != "" {
		t.Errorf("unclaimed ticket should have empty ClaimerID, got %q", v.ClaimerID)
	}

	tk.ClaimerID = 222
	v = m.toTicketView("123", tk)
	if v.ClaimerID != "222" {
		t.Errorf("claimed ticket ClaimerID = %q, want 222", v.ClaimerID)
	}
}
