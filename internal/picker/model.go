package picker

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Model is the picker's Bubble Tea model.
type Model struct {
	title     string
	input     textinput.Model
	items     []Item
	filtered  []int // indexes into items
	cursor    int
	chosen    *Item
	cancelled bool
	cursorSty lipgloss.Style
	titleSty  lipgloss.Style
	dimSty    lipgloss.Style
}

// NewModel builds a picker over items with an empty filter.
func NewModel(title string, items []Item) Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Focus()

	m := Model{title: title, input: ti, items: items}
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		m.cursorSty = lipgloss.NewStyle()
		m.titleSty = lipgloss.NewStyle()
		m.dimSty = lipgloss.NewStyle()
	} else {
		m.cursorSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
		m.titleSty = lipgloss.NewStyle().Bold(true)
		m.dimSty = lipgloss.NewStyle().Faint(true)
	}
	m.refilter()
	return m
}

func (m *Model) refilter() {
	m.cursor = 0
	q := m.input.Value()
	m.filtered = nil // fresh slice: Model is copied by value
	if q == "" {
		for i := range m.items {
			m.filtered = append(m.filtered, i)
		}
		return
	}
	hay := make([]string, len(m.items))
	for i, it := range m.items {
		hay[i] = it.Label + " " + it.Detail
	}
	for _, r := range fuzzy.Find(q, hay) {
		m.filtered = append(m.filtered, r.Index)
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch key.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		if len(m.filtered) == 0 {
			return m, nil
		}
		it := m.items[m.filtered[m.cursor]]
		m.chosen = &it
		return m, tea.Quit
	case tea.KeyUp, tea.KeyCtrlP:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown, tea.KeyCtrlN:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil
	}
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.refilter()
	}
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.titleSty.Render(m.title))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	for row, idx := range m.filtered {
		it := m.items[idx]
		line := it.Label
		if it.Detail != "" {
			line += "  " + it.Detail
		}
		if row == m.cursor {
			b.WriteString(m.cursorSty.Render("> " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString(m.dimSty.Render("  no matches"))
		b.WriteString("\n")
	}
	return b.String()
}

// Filtered returns the visible items in ranked order.
func (m Model) Filtered() []Item {
	out := make([]Item, len(m.filtered))
	for i, idx := range m.filtered {
		out[i] = m.items[idx]
	}
	return out
}

// Cursor returns the index of the highlighted row within Filtered.
func (m Model) Cursor() int { return m.cursor }

// Result returns the chosen item, or ErrCancelled if none was chosen.
func (m Model) Result() (Item, error) {
	if m.chosen == nil {
		return Item{}, ErrCancelled
	}
	return *m.chosen, nil
}
