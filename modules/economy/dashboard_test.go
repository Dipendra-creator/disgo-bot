package economy

import (
	"testing"

	"github.com/dipu-sharma/disgo-bot/shared"
)

func TestToShopView(t *testing.T) {
	it := &ShopItem{ID: 7, GuildID: 1, Name: "VIP", Description: "shiny", Price: 500, RoleID: 42, Stock: -1}
	v := toShopView(it)
	if v.ID != 7 || v.Name != "VIP" || v.Price != 500 || v.Stock != -1 {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.RoleID != "42" {
		t.Errorf("RoleID = %q, want \"42\"", v.RoleID)
	}

	noRole := toShopView(&ShopItem{ID: 8, RoleID: 0})
	if noRole.RoleID != "" {
		t.Errorf("zero role should map to empty string, got %q", noRole.RoleID)
	}
}

func TestValidateItem(t *testing.T) {
	long := make([]byte, itemNameMax+1)
	for i := range long {
		long[i] = 'a'
	}
	cases := []struct {
		name    string
		in      shared.ShopItemInput
		wantErr bool
	}{
		{"ok", shared.ShopItemInput{Name: "Sword", Price: 100, Stock: 5}, false},
		{"ok unlimited + role", shared.ShopItemInput{Name: "Role", Price: 0, Stock: -1, RoleID: "123"}, false},
		{"trims name", shared.ShopItemInput{Name: "  x  ", Price: 1, Stock: 0}, false},
		{"empty name", shared.ShopItemInput{Name: "   ", Price: 1, Stock: 0}, true},
		{"name too long", shared.ShopItemInput{Name: string(long), Price: 1, Stock: 0}, true},
		{"negative price", shared.ShopItemInput{Name: "x", Price: -1, Stock: 0}, true},
		{"stock below -1", shared.ShopItemInput{Name: "x", Price: 1, Stock: -2}, true},
		{"bad role", shared.ShopItemInput{Name: "x", Price: 1, Stock: 0, RoleID: "abc"}, true},
		{"zero role rejected", shared.ShopItemInput{Name: "x", Price: 1, Stock: 0, RoleID: "0"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := validateItem(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (out=%+v)", out)
				}
				if _, ok := shared.AsUserError(err); !ok {
					t.Errorf("expected UserError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Name != trimmed(c.in.Name) {
				t.Errorf("name not trimmed: got %q", out.Name)
			}
		})
	}
}

func trimmed(s string) string {
	// mirror strings.TrimSpace for the single space-padded case under test
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

func TestIsSnowflake(t *testing.T) {
	for _, s := range []string{"123", "962587206604685342"} {
		if !isSnowflake(s) {
			t.Errorf("isSnowflake(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "0", "-1", "12x", "abc"} {
		if isSnowflake(s) {
			t.Errorf("isSnowflake(%q) = true, want false", s)
		}
	}
}

func TestClampPage(t *testing.T) {
	if l, o := clampPage(shared.PageQuery{Limit: 0, Offset: -5}, 50); l != 50 || o != 0 {
		t.Errorf("clampPage zero/neg = (%d,%d), want (50,0)", l, o)
	}
	if l, o := clampPage(shared.PageQuery{Limit: 999, Offset: 10}, 50); l != 50 || o != 10 {
		t.Errorf("clampPage over-limit = (%d,%d), want (50,10)", l, o)
	}
	if l, o := clampPage(shared.PageQuery{Limit: 20, Offset: 5}, 50); l != 20 || o != 5 {
		t.Errorf("clampPage in-range = (%d,%d), want (20,5)", l, o)
	}
}
