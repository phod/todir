package cli

import "fmt"

// UsageError marks an error caused by invalid command-line usage (exit 2).
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(format string, a ...any) error {
	return &UsageError{Msg: fmt.Sprintf(format, a...)}
}

func exitCode(err error) int {
	// The 130 (cancelled) case is added in Task 9.
	switch err.(type) {
	case nil:
		return 0
	case *UsageError:
		return 2
	default:
		return 1
	}
}
