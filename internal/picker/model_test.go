package picker

import (
	"errors"
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
