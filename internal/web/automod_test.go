package web

import (
	"encoding/json"
	"testing"
)

func TestWordRequestDecode(t *testing.T) {
	var b wordRequest
	if err := json.Unmarshal([]byte(`{"word":"BadWord"}`), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.Word != "BadWord" {
		t.Fatalf("word = %q, want BadWord", b.Word)
	}
}

func TestWordsResponseEncode(t *testing.T) {
	out, err := json.Marshal(wordsResponse{Words: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); got != `{"words":["a","b"]}` {
		t.Fatalf("encoded = %s", got)
	}
}
