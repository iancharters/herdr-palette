// Package ui implements the palette as an extensible Bubble Tea model.
// Composition points: Matcher filters items, Executor runs them, KeyMap
// rebinds keys — embed Model or swap these to extend without forking.
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/iancharters/herdr-palette/internal/model"
	"github.com/iancharters/herdr-palette/internal/theme"
	"github.com/iancharters/herdr-palette/internal/viewport"
)

// Matcher decides visibility; the default is token-substring like the TS palette.
type Matcher func(item model.PaletteItem, query string) bool

// Executor runs an item; ok=true closes the palette.
type Executor func(item model.PaletteItem, input string) model.CommandResult

// KeyMap rebinds navigation without touching the model.
type KeyMap struct {
	Up, Down, Confirm, Back []string
}

// DefaultMatcher matches every whitespace-separated token against
// title+description+aliases+shortcuts, case-insensitive.
func DefaultMatcher(item model.PaletteItem, query string) bool {
	hay := strings.ToLower(strings.Join(append([]string{item.Title, item.Description}, append(append([]string{}, item.Aliases...), item.Shortcuts...)...), " "))
	for _, tok := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(hay, tok) {
			return false
		}
	}
	return true
}

type closeMsg struct{ ok bool }

// Model is the palette. Use New with items, styles, and an executor.
type Model struct {
	items      []model.PaletteItem
	styles     theme.Styles
	exec       Executor
	match      Matcher
	keys       KeyMap
	input      textinput.Model
	query      string
	selected   int
	status     string
	running    bool
	promptItem *model.PaletteItem
	promptVal  string
	width      int
	height     int
	closed     bool
	shouldQuit bool
}

func New(items []model.PaletteItem, styles theme.Styles, exec Executor) Model {
	ti := textinput.New()
	ti.Placeholder = "Search commands"
	ti.Focus()
	m := Model{
		items: items, styles: styles, exec: exec, match: DefaultMatcher,
		keys: KeyMap{Up: []string{"up", "ctrl+p"}, Down: []string{"down", "ctrl+n"}, Confirm: []string{"enter"}, Back: []string{"esc"}},
		input: ti, width: 80, height: 20,
	}
	return m
}

func (m Model) visible() []model.PaletteItem {
	q := m.query
	if m.promptItem != nil {
		return m.items
	}
	out := make([]model.PaletteItem, 0, len(m.items))
	for _, it := range m.items {
		if m.match(it, q) {
			out = append(out, it)
		}
	}
	return out
}

func groupKey(it model.PaletteItem) string { return string(it.Category) + "\x00" + it.Group }

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case closeMsg:
		m.shouldQuit = true
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if contains(m.keys.Back, key) {
			if m.promptItem != nil {
				m.promptItem = nil
				m.promptVal = ""
				m.status = ""
				m.input.SetValue("")
				m.input.Placeholder = "Search commands"
				return m, nil
			}
			return m, tea.Quit
		}
		if m.promptItem != nil {
			if contains(m.keys.Confirm, key) && !m.running {
				return m.runSelected()
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.promptVal = m.input.Value()
			return m, cmd
		}
		switch {
		case key == "up" || key == "ctrl+p":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case key == "down" || key == "ctrl+n":
			if m.selected < len(m.visible())-1 {
				m.selected++
			}
			return m, nil
		case contains(m.keys.Confirm, key):
			if !m.running {
				return m.runSelected()
			}
			return m, nil
		default:
			prev := m.query
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.query = m.input.Value()
			if m.query != prev {
				m.selected = 0
				m.status = ""
			}
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func contains(list []string, key string) bool {
	for _, k := range list {
		if k == key {
			return true
		}
	}
	return false
}

func (m Model) runSelected() (tea.Model, tea.Cmd) {
	vis := m.visible()
	if m.promptItem != nil {
		item := *m.promptItem
		m.running = true
		res := m.exec(item, m.promptVal)
		m.running = false
		if res.OK {
			return m, func() tea.Msg { return closeMsg{true} }
		}
		m.status = res.Message
		return m, nil
	}
	if len(vis) == 0 {
		m.status = "No commands match your search."
		return m, nil
	}
	if m.selected < 0 || m.selected >= len(vis) {
		m.selected = 0
	}
	item := vis[m.selected]
	if item.Prompt != nil {
		cp := item
		m.promptItem = &cp
		m.promptVal = ""
		m.input.SetValue("")
		m.input.Placeholder = item.Prompt.Placeholder
		m.status = ""
		return m, nil
	}
	m.running = true
	res := m.exec(item, "")
	m.running = false
	if res.OK {
		return m, func() tea.Msg { return closeMsg{true} }
	}
	m.status = res.Message
	return m, nil
}

// View renders heading, input, grouped list, status, and a bottom-pinned footer.
func (m Model) View() string {
	lines := []string{}
	title := "Commands"
	if m.promptItem != nil {
		title = m.promptItem.Title
	}
	lines = append(lines,
		m.styles.Accent.Bold(true).Render(title)+"  "+m.styles.Muted.Render("esc"),
		m.input.View(),
		"",
	)

	if m.promptItem != nil {
		lines = append(lines, m.styles.Muted.Render(m.promptItem.Description))
	} else {
		vis := m.visible()
		if len(vis) == 0 {
			lines = append(lines, m.styles.Muted.Render("No commands match your search."))
		} else {
			capacity := max(1, m.height-4)
			if m.status != "" {
				capacity = max(1, capacity-1)
			}
			win := viewport.GroupedGaps(len(vis), m.selected, capacity, func(i int) string { return groupKey(vis[i]) })
			// Fixed shortcut column so keybinds align: pad labels to the
			// widest visible label (display width, not byte length).
			labelOf := func(it model.PaletteItem) string { return it.Icon + "  " + it.Title }
			labelW := 0
			for _, it := range vis {
				if w := runewidth.StringWidth(labelOf(it)); w > labelW {
					labelW = w
				}
			}
			cat, grp := "", ""
			afterHeader := false // last emitted line was a header: skip the gap
			started := false     // any list line emitted yet (no gap before the first)
			for idx := win.Start; idx < win.End; idx++ {
				it := vis[idx]
				if string(it.Category) != cat {
					if started && !afterHeader {
						lines = append(lines, "")
					}
					cat, grp = string(it.Category), ""
					lines = append(lines, m.styles.Accent.Bold(true).Render(cat))
					afterHeader, started = true, true
				}
				if it.Group != "" && it.Group != grp {
					if !afterHeader {
						lines = append(lines, "")
					}
					grp = it.Group
					lines = append(lines, m.styles.Muted.Render("  "+grp))
					afterHeader = true
				} else if it.Group == "" {
					grp = ""
				}
				icon := lipgloss.NewStyle().Width(3).Align(lipgloss.Center).Render(it.Icon)
				label := labelOf(it)
				row := icon + padRight(label, labelW)
				if keys := strings.Join(it.Shortcuts, " / "); keys != "" {
					row += "  " + keys
				}
				if idx == m.selected {
					lines = append(lines, m.styles.Panel.Render(m.styles.Text.Render("┃ "+row)))
				} else {
					lines = append(lines, m.styles.Muted.Render("  "+row))
				}
				afterHeader = false
			}
		}
	}
	if m.status != "" {
		lines = append(lines, m.styles.Accent.Render(oneLine(m.status, max(20, m.width-4))))
	}
	// Pin the footer to the bottom of the frame.
	footer := m.styles.Footer.Render(
		m.styles.Accent.Bold(true).Render("enter") + m.styles.FooterText.Render(" select   ") +
			m.styles.Accent.Bold(true).Render("↑/↓") + m.styles.FooterText.Render(" move   ") +
			m.styles.FooterText.Render(itoa(len(m.visible()))+" commands"))
	lines = append(lines, strings.Repeat("\n", max(0, m.height-len(lines)-1)), footer)
	return strings.Join(lines, "\n")
}

// padRight pads s with spaces to display width w (rune-width aware).
func padRight(s string, w int) string {
	if pad := w - runewidth.StringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
