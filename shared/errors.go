package shared

import (
	"errors"
	"fmt"
)

// UserError is an error whose message is safe and intended to be shown to the
// end user (e.g. "You don't have permission to do that"). The router surfaces
// its message in an ephemeral reply instead of a generic failure.
type UserError struct {
	Msg string
}

func (e *UserError) Error() string { return e.Msg }

// UserErr builds a UserError with printf-style formatting.
func UserErr(format string, args ...any) error {
	return &UserError{Msg: fmt.Sprintf(format, args...)}
}

// AsUserError reports whether err is (or wraps) a *UserError.
func AsUserError(err error) (*UserError, bool) {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue, true
	}
	return nil, false
}
