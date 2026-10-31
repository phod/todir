package cli

import (
	"errors"
	"fmt"

	"github.com/phod/todir/internal/picker"
)

// UsageError marks an error caused by invalid command-line usage (exit 2).
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(format string, a ...any) error {
	return &UsageError{Msg: fmt.Sprintf(format, a...)}
}

func exitCode(err error) int {
	var ue *UsageError
	switch {
	case err == nil:
		return 0
	case errors.Is(err, picker.ErrCancelled):
		return 130
	case errors.As(err, &ue):
		return 2
	default:
		return 1
	}
}
