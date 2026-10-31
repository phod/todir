// Package picker is a small fuzzy-filter selection UI built on Bubble Tea.
package picker

import (
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// Item is one selectable row.
type Item struct {
	Label  string // alias name or branch
	Detail string // path, optionally with a worktree note
	Value  string // what the caller acts on
}

var (
	ErrCancelled  = errors.New("cancelled")
	ErrNoTerminal = errors.New("interactive mode needs a terminal")
)

// Run shows the picker on out, reading keys from in, and returns the chosen
// item. It returns ErrNoTerminal when isTTY is false and ErrCancelled when the
// user quits without choosing.
func Run(title string, items []Item, in io.Reader, out io.Writer, isTTY bool) (Item, error) {
	if !isTTY {
		return Item{}, ErrNoTerminal
	}
	p := tea.NewProgram(NewModel(title, items), tea.WithInput(in), tea.WithOutput(out))
	final, err := p.Run()
	if err != nil {
		return Item{}, runErr(err)
	}
	m, ok := final.(Model)
	if !ok {
		return Item{}, ErrCancelled
	}
	return m.Result()
}

// runErr maps an external SIGINT (tea.ErrInterrupted) to ErrCancelled so it
// exits 130 like Escape or Ctrl-C.
func runErr(err error) error {
	if errors.Is(err, tea.ErrInterrupted) {
		return ErrCancelled
	}
	return err
}
