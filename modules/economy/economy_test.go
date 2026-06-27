package economy

import "testing"

func TestRandAmount(t *testing.T) {
	for i := 0; i < 1000; i++ {
		v := randAmount(50, 250)
		if v < 50 || v > 250 {
			t.Fatalf("randAmount(50,250) out of range: %d", v)
		}
	}
	if randAmount(100, 100) != 100 {
		t.Error("equal bounds should return the bound")
	}
	// Inverted range tolerated.
	v := randAmount(250, 50)
	if v < 50 || v > 250 {
		t.Fatalf("inverted range out of bounds: %d", v)
	}
}

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings(42)
	if s.GuildID != 42 {
		t.Errorf("GuildID = %d, want 42", s.GuildID)
	}
	if s.CurrencyName != "coins" || s.CurrencySymbol == "" {
		t.Errorf("unexpected currency: %q %q", s.CurrencyName, s.CurrencySymbol)
	}
	if s.DailyAmount != 250 || s.WorkMin != 50 || s.WorkMax != 250 || s.WorkCooldownSec != 3600 {
		t.Errorf("unexpected reward defaults: %+v", s)
	}
	if s.StartingBalance != 0 {
		t.Errorf("StartingBalance = %d, want 0", s.StartingBalance)
	}
}

func TestAccountNet(t *testing.T) {
	a := &Account{Wallet: 120, Bank: 80}
	if a.Net() != 200 {
		t.Errorf("Net() = %d, want 200", a.Net())
	}
}

func TestShopItemUnlimited(t *testing.T) {
	if !(&ShopItem{Stock: -1}).Unlimited() {
		t.Error("stock -1 should be unlimited")
	}
	if (&ShopItem{Stock: 0}).Unlimited() || (&ShopItem{Stock: 5}).Unlimited() {
		t.Error("non-negative stock should be limited")
	}
}

func TestMoney(t *testing.T) {
	s := defaultSettings(1)
	s.CurrencySymbol = "$"
	if got := money(s, 1234567); got != "$ 1,234,567" {
		t.Errorf("money = %q, want %q", got, "$ 1,234,567")
	}
}

func TestPageCount(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, itemsPerPage: 1, itemsPerPage + 1: 2, itemsPerPage * 3: 3}
	for total, want := range cases {
		if got := pageCount(total); got != want {
			t.Errorf("pageCount(%d) = %d, want %d", total, got, want)
		}
	}
}

func TestIDs(t *testing.T) {
	if pid("175928847299117063") != 175928847299117063 {
		t.Error("pid round-trip failed")
	}
	if sid(175928847299117063) != "175928847299117063" {
		t.Error("sid round-trip failed")
	}
}

func TestValidAmount(t *testing.T) {
	if validAmount(0) == nil || validAmount(-5) == nil {
		t.Error("non-positive amounts should be rejected")
	}
	if validAmount(maxAmount+1) == nil {
		t.Error("oversized amount should be rejected")
	}
	if validAmount(100) != nil {
		t.Error("valid amount should pass")
	}
}
