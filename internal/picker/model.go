package picker

import (
	"fmt"
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
	height    int // terminal rows; 0 until a WindowSizeMsg arrives
	offset    int // index into filtered of the first rendered row
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
	if noColor(os.Getenv) {
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

// noColor reports whether colour is disabled: NO_COLOR set to a non-empty
// value (https://no-color.org).
func noColor(getenv func(string) string) bool { return getenv("NO_COLOR") != "" }

// window returns how many item rows fit: the terminal height minus the title,
// prompt and "more" lines. Zero means no limit (height not yet known).
func (m Model) window() int {
	if m.height <= 0 {
		return 0
	}
	if w := m.height - 3; w > 0 {
		return w
	}
	return 1
}

// follow scrolls offset so the cursor row stays inside the rendered window.
func (m *Model) follow() {
	w := m.window()
	if w == 0 {
		m.offset = 0
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+w {
		m.offset = m.cursor - w + 1
	}
	if maxOff := len(m.filtered) - w; m.offset > maxOff {
		m.offset = max(maxOff, 0)
	}
}

func (m *Model) refilter() {
	m.cursor = 0
	m.offset = 0
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
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = ws.Height
		m.follow()
		return m, nil
	}
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
		m.follow()
		return m, nil
	case tea.KeyDown, tea.KeyCtrlN:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		m.follow()
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

// View renders the title, the prompt and a window of rows that follows the
// cursor. It has no trailing newline, so at most height lines are drawn and
// the renderer never pushes the title off the top of the terminal.
func (m Model) View() string {
	lines := []string{m.titleSty.Render(m.title), m.input.View()}
	start, end := 0, len(m.filtered)
	if w := m.window(); w > 0 && end > w {
		start = m.offset
		end = min(start+w, len(m.filtered))
	}
	for row := start; row < end; row++ {
		it := m.items[m.filtered[row]]
		line := it.Label
		if it.Detail != "" {
			line += "  " + it.Detail
		}
		if row == m.cursor {
			lines = append(lines, m.cursorSty.Render("> "+line))
		} else {
			lines = append(lines, "  "+line)
		}
	}
	if hidden := len(m.filtered) - (end - start); hidden > 0 {
		lines = append(lines, m.dimSty.Render(fmt.Sprintf("  … %d more", hidden)))
	}
	if len(m.filtered) == 0 {
		lines = append(lines, m.dimSty.Render("  no matches"))
	}
	return strings.Join(lines, "\n")
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
