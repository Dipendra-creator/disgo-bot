package shared_test

import (
	"testing"

	"github.com/dipu-sharma/disgo-bot/shared"
	"github.com/stretchr/testify/assert"
)

func TestBuildAndParseID(t *testing.T) {
	tests := []struct {
		name    string
		module  string
		action  string
		args    []string
		wantID  string
		wantArg []string
	}{
		{"no args", "utility", "refresh", nil, "utility:refresh", nil},
		{"one arg", "utility", "avatar", []string{"42"}, "utility:avatar:42", []string{"42"}},
		{"many args", "tickets", "open", []string{"a", "b", "c"}, "tickets:open:a:b:c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := shared.BuildID(tt.module, tt.action, tt.args...)
			assert.Equal(t, tt.wantID, id)

			mod, act, args := shared.ParseID(id)
			assert.Equal(t, tt.module, mod)
			assert.Equal(t, tt.action, act)
			assert.Equal(t, tt.wantArg, args)
		})
	}
}

func TestParseIDEdgeCases(t *testing.T) {
	mod, act, args := shared.ParseID("solo")
	assert.Equal(t, "solo", mod)
	assert.Equal(t, "", act)
	assert.Nil(t, args)
}

func TestAsUserError(t *testing.T) {
	ue, ok := shared.AsUserError(shared.UserErr("nope %d", 1))
	assert.True(t, ok)
	assert.Equal(t, "nope 1", ue.Error())

	_, ok = shared.AsUserError(assert.AnError)
	assert.False(t, ok)
}
