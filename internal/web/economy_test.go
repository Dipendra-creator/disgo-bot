package web

import "testing"

func TestShopItemRequestInput(t *testing.T) {
	b := shopItemRequest{
		Name:        "VIP",
		Description: "shiny role",
		Price:       500,
		RoleID:      "42",
		Stock:       -1,
	}
	in := b.input()
	if in.Name != "VIP" || in.Description != "shiny role" || in.Price != 500 || in.RoleID != "42" || in.Stock != -1 {
		t.Fatalf("unexpected mapping: %+v", in)
	}
}
