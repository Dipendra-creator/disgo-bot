package shared

import "testing"

func sampleSchema() ConfigSchema {
	return ConfigSchema{
		Module: "sample",
		Title:  "Sample",
		Fields: []Field{
			{Key: "on", Type: FieldBool},
			{Key: "count", Type: FieldInt, Min: 1, Max: 10},
			{Key: "free", Type: FieldInt}, // unbounded
			{Key: "note", Type: FieldString, MaxLen: 5},
			{Key: "chan", Type: FieldChannel},
		},
	}
}

func TestNormalizeHappyPath(t *testing.T) {
	s := sampleSchema()
	out, err := s.Normalize(map[string]any{
		"on":    true,
		"count": float64(7), // JSON numbers decode to float64
		"note":  "hi",
		"chan":  "123456789012345678",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["on"] != true {
		t.Fatalf("on = %v", out["on"])
	}
	if out["count"] != 7 {
		t.Fatalf("count = %v (%T), want int 7", out["count"], out["count"])
	}
	if out["chan"] != "123456789012345678" {
		t.Fatalf("chan = %v", out["chan"])
	}
	// Partial patch: only provided keys appear.
	if _, ok := out["free"]; ok {
		t.Fatal("free should not be present in a partial patch")
	}
}

func TestNormalizeRejections(t *testing.T) {
	s := sampleSchema()
	cases := map[string]map[string]any{
		"unknown key":        {"nope": true},
		"int over max":       {"count": float64(99)},
		"int under min":      {"count": float64(0)},
		"int non-whole":      {"count": float64(3.5)},
		"bool wrong type":    {"on": "yes"},
		"string too long":    {"note": "toolong"},
		"channel non-digit":  {"chan": "abc"},
		"channel wrong type": {"chan": float64(5)},
	}
	for name, patch := range cases {
		if _, err := s.Normalize(patch); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestNormalizeEmptyChannelClears(t *testing.T) {
	s := sampleSchema()
	out, err := s.Normalize(map[string]any{"chan": ""})
	if err != nil {
		t.Fatalf("empty channel must be allowed (clear): %v", err)
	}
	if out["chan"] != "" {
		t.Fatalf("chan = %q, want empty", out["chan"])
	}
}
