package automod

import (
	"context"
	"testing"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// The action fields accept only the delete/timeout enum; an invalid value is
// rejected before any persistence happens (so no DB/service is needed here).
func TestSetConfigRejectsBadAction(t *testing.T) {
	m := &Module{}
	for _, key := range []string{"words_action", "invites_action", "mentions_action", "spam_action"} {
		err := m.SetConfig(context.Background(), 123, map[string]any{key: "nuke"})
		if err == nil {
			t.Fatalf("%s: expected error for invalid action", key)
		}
		if _, ok := shared.AsUserError(err); !ok {
			t.Fatalf("%s: want UserError, got %T", key, err)
		}
	}
}

func TestConfigSchemaShape(t *testing.T) {
	s := (&Module{}).ConfigSchema()
	if s.Module != "automod" {
		t.Fatalf("module = %q", s.Module)
	}
	if len(s.Fields) == 0 {
		t.Fatal("schema has no fields")
	}
	// Every field must declare a known type and a non-empty key.
	known := map[shared.FieldType]bool{
		shared.FieldBool: true, shared.FieldInt: true, shared.FieldString: true,
		shared.FieldChannel: true, shared.FieldRole: true,
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		if f.Key == "" || !known[f.Type] {
			t.Fatalf("bad field %+v", f)
		}
		if seen[f.Key] {
			t.Fatalf("duplicate key %q", f.Key)
		}
		seen[f.Key] = true
	}
}
