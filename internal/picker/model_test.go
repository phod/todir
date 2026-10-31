package picker

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sample() []Item {
	return []Item{
		{Label: "acct", Detail: "/src/acct", Value: "acct"},
		{Label: "notes", Detail: "/n", Value: "notes"},
		{Label: "account-service", Detail: "/s", Value: "account-service"},
	}
}

func send(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func typeText(m Model, s string) Model {
	for _, r := range s {
		m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func labels(items []Item) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		out[it.Label] = true
	}
	return out
}

func TestTypingFiltersFuzzy(t *testing.T) {
	m := typeText(NewModel("T", sample()), "acs")
	got := labels(m.Filtered())
	if len(got) != 2 || !got["acct"] || !got["account-service"] || got["notes"] {
		t.Fatalf("filtered = %v", m.Filtered())
	}
}

func TestEmptyFilterKeepsOrder(t *testing.T) {
	m := NewModel("T", sample())
	f := m.Filtered()
	if len(f) != 3 || f[0].Label != "acct" || f[1].Label != "notes" || f[2].Label != "account-service" {
		t.Fatalf("filtered = %v", f)
	}
}

func TestArrowKeysMoveCursorWithinBounds(t *testing.T) {
	m := NewModel("T", sample())
	m = send(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor() != 0 {
		t.Fatalf("cursor = %d", m.Cursor())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d", m.Cursor())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor() != 2 {
		t.Fatalf("cursor past end = %d", m.Cursor())
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlP})
	if m.Cursor() != 1 {
		t.Fatalf("cursor = %d", m.Cursor())
	}
}

func TestEnterChoosesCursorItem(t *testing.T) {
	m := send(NewModel("T", sample()), tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyEnter})
	it, err := m.Result()
	if err != nil || it.Value != "notes" {
		t.Fatalf("result = %v, %v", it, err)
	}
}

func TestEscapeCancels(t *testing.T) {
	m := send(NewModel("T", sample()), tea.KeyMsg{Type: tea.KeyEsc})
	if _, err := m.Result(); !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCtrlCCancels(t *testing.T) {
	m := send(NewModel("T", sample()), tea.KeyMsg{Type: tea.KeyCtrlC})
	if _, err := m.Result(); !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnterOnEmptyFilterIsNoop(t *testing.T) {
	m := typeText(NewModel("T", sample()), "zzz")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("enter on empty list must not quit")
	}
	if m.chosen != nil {
		t.Fatal("chosen set")
	}
}

func TestFilterResetsCursor(t *testing.T) {
	m := send(NewModel("T", sample()), tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d", m.Cursor())
	}
	m = typeText(m, "a")
	if m.Cursor() != 0 {
		t.Fatalf("cursor after typing = %d", m.Cursor())
	}
}

func TestNoColorNeedsNonEmptyValue(t *testing.T) {
	env := func(v string) func(string) string {
		return func(k string) string {
			if k == "NO_COLOR" {
				return v
			}
			return ""
		}
	}
	if noColor(env("")) {
		t.Fatal("empty NO_COLOR must not disable colour")
	}
	if !noColor(env("1")) {
		t.Fatal("NO_COLOR=1 must disable colour")
	}
	t.Setenv("NO_COLOR", "")
	if !NewModel("T", nil).cursorSty.GetBold() {
		t.Fatal("empty NO_COLOR: cursor style should be bold")
	}
	t.Setenv("NO_COLOR", "1")
	if NewModel("T", nil).cursorSty.GetBold() {
		t.Fatal("NO_COLOR=1: cursor style should be plain")
	}
}

func TestInterruptMapsToCancelled(t *testing.T) {
	if err := runErr(tea.ErrInterrupted); !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v", err)
	}
	if err := runErr(fmt.Errorf("wrap: %w", tea.ErrInterrupted)); !errors.Is(err, ErrCancelled) {
		t.Fatalf("wrapped err = %v", err)
	}
	other := errors.New("boom")
	if err := runErr(other); err != other {
		t.Fatalf("other err = %v", err)
	}
}

func many(n int) []Item {
	out := make([]Item, n)
	for i := range out {
		v := fmt.Sprintf("item%02d", i)
		out[i] = Item{Label: v, Detail: "/p/" + v, Value: v}
	}
	return out
}

// itemRows returns the rendered item lines (those naming an item).
func itemRows(view string) []string {
	var rows []string
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, "item") {
			rows = append(rows, l)
		}
	}
	return rows
}

func hasRow(rows []string, want string) bool {
	for _, r := range rows {
		if strings.Contains(r, want) {
			return true
		}
	}
	return false
}

func TestViewFitsWindowAndFollowsCursor(t *testing.T) {
	m := send(NewModel("T", many(20)), tea.WindowSizeMsg{Width: 80, Height: 6})
	view := m.View()
	if n := len(strings.Split(view, "\n")); n > 6 {
		t.Fatalf("view has %d lines, want <= 6:\n%s", n, view)
	}
	if rows := itemRows(view); len(rows) != 3 || !strings.Contains(rows[0], "> item00") {
		t.Fatalf("rows = %q", rows)
	}
	if !strings.HasPrefix(view, "T\n") || !strings.Contains(view, "… 17 more") {
		t.Fatalf("view:\n%s", view)
	}
	for i := 1; i <= 10; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyDown})
		view = m.View()
		rows := itemRows(view)
		if len(rows) > 3 {
			t.Fatalf("after %d downs: %d rows", i, len(rows))
		}
		if want := fmt.Sprintf("> item%02d", i); !hasRow(rows, want) {
			t.Fatalf("after %d downs cursor row %q not rendered: %q", i, want, rows)
		}
		if n := len(strings.Split(view, "\n")); n > 6 || !strings.HasPrefix(view, "T\n") {
			t.Fatalf("after %d downs view:\n%s", i, view)
		}
	}
	for i := 0; i < 10; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyUp})
	}
	if rows := itemRows(m.View()); !strings.Contains(rows[0], "> item00") {
		t.Fatalf("rows after moving back up = %q", rows)
	}
}

func TestViewShowsAllRowsBeforeWindowSize(t *testing.T) {
	view := NewModel("T", many(20)).View()
	if rows := itemRows(view); len(rows) != 20 || strings.Contains(view, "more") {
		t.Fatalf("rows = %d\n%s", len(rows), view)
	}
}
