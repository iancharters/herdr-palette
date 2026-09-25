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
	output     string // result view text (plugin action stdout); "" = list mode
	outputTitle string
	outputScroll int
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
			if m.output != "" {
				m.output, m.outputTitle, m.outputScroll = "", "", 0
				return m, nil
			}
		if m.output != "" {
			// Result view: scroll output, esc backs out (handled above).
			lines := strings.Count(m.output, "\n") + 1
			span := max(1, m.height-4)
			switch {
			case key == "up" || key == "ctrl+p":
				if m.outputScroll > 0 {
					m.outputScroll--
				}
				return m, nil
			case key == "down" || key == "ctrl+n":
				if m.outputScroll < max(0, lines-span) {
					m.outputScroll++
				}
				return m, nil
			}
			return m, nil
		}
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
		return m.finishRun(res)
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
	return m.finishRun(res)
}

// finishRun routes a result: output opens the scrollable result view,
// clean success closes the palette, anything else is a status line.
func (m Model) finishRun(res model.CommandResult) (tea.Model, tea.Cmd) {
	if res.Output != "" {
		m.output, m.outputTitle, m.outputScroll = res.Output, res.Title, 0
		m.status = res.Message
		m.promptItem = nil
		m.promptVal = ""
		m.input.SetValue(m.query)
		m.input.Placeholder = "Search commands"
		return m, nil
	}
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

	if m.output != "" {
		all := strings.Split(m.output, "\n")
		span := max(1, m.height-5-(boolToInt(m.status != "")))
		start := min(m.outputScroll, max(0, len(all)-span))
		m.outputScroll = start
		for _, ln := range all[start:min(len(all), start+span)] {
			lines = append(lines, m.styles.Text.Render(ln))
		}
		if m.status != "" {
			lines = append(lines, m.styles.Accent.Render(oneLine(m.status, max(20, m.width-4))))
		}
		footer := m.styles.Footer.Render(
			m.styles.Accent.Bold(true).Render("esc") + m.styles.FooterText.Render(" back   ") +
				m.styles.Accent.Bold(true).Render("↑/↓") + m.styles.FooterText.Render(" scroll"))
		lines = append(lines, strings.Repeat("\n", max(0, m.height-len(lines)-1)), footer)
		return strings.Join(lines, "\n")
	}

	if m.promptItem != nil {
		lines = append(lines, m.styles.Muted.Render(m.promptItem.Description))
	} else {
		vis := m.visible()
		if len(vis) == 0 {
			lines = append(lines, m.styles.Muted.Render("No commands match your search."))
		} else {
			body := m.renderList(vis)
			// Shrink-to-fit dialog: center the content block instead of
			// stretching rows across the frame.
			lines = append(lines, lipgloss.PlaceHorizontal(m.width, lipgloss.Center, strings.Join(body, "\n")))
		}
	}
	if m.status != "" {
		lines = append(lines, m.styles.Accent.Render(oneLine(m.status, max(20, m.width-4))))
	}
	// Pin the footer to the bottom of the frame. The centered body is one
	// string element, so count its logical lines explicitly.
	bodyCount := 0
	for _, ln := range lines[3:] {
		bodyCount += strings.Count(ln, "\n") + 1
	}
	footer := m.styles.Footer.Render(
		m.styles.Accent.Bold(true).Render("enter") + m.styles.FooterText.Render(" select   ") +
			m.styles.Accent.Bold(true).Render("↑/↓") + m.styles.FooterText.Render(" move   ") +
			m.styles.FooterText.Render(itoa(len(m.visible()))+" commands"))
	lines = append(lines, strings.Repeat("\n", max(0, m.height-3-bodyCount-1)), footer)
	return strings.Join(lines, "\n")
}

// renderList builds the dialog body: grouped rows with wrapped titles and a
// fixed keybind column aligned with each row's first line.
func (m Model) renderList(vis []model.PaletteItem) []string {
	frameInner := max(20, m.width-4)
	maxKeys, maxTitle := 0, 0
	keysOf := make([]string, len(vis))
	for i, it := range vis {
		keysOf[i] = strings.Join(it.Shortcuts, " / ")
		if w := runewidth.StringWidth(keysOf[i]); w > maxKeys {
			maxKeys = w
		}
		if w := runewidth.StringWidth(it.Title); w > maxTitle {
			maxTitle = w
		}
	}
	keysGap := 0
	if maxKeys > 0 {
		keysGap = 2
	}
	// icon cell (3) + space (1) + title + gap + keybinds must fit frameInner.
	titleW := min(maxTitle, max(10, frameInner-4-1-keysGap-maxKeys))
	tlines := make([][]string, len(vis))
	for i, it := range vis {
		tlines[i] = wrapText(it.Title, titleW)
	}
	capacity := max(1, m.height-4)
	if m.status != "" {
		capacity = max(1, capacity-1)
	}
	win := viewport.GroupedAdv(len(vis), m.selected, capacity,
		func(i int) string { return string(vis[i].Category) },
		func(i int) string { return vis[i].Group },
		func(i int) int { return len(tlines[i]) })
	body := []string{}
	cat, grp := "", ""
	afterHeader := false // last emitted line was a header: skip the gap
	started := false     // any list line emitted yet (no gap before the first)
	for idx := win.Start; idx < win.End; idx++ {
		it := vis[idx]
		if string(it.Category) != cat {
			if started && !afterHeader {
				body = append(body, "")
			}
			cat, grp = string(it.Category), ""
			body = append(body, m.styles.Accent.Bold(true).Render(cat))
			afterHeader, started = true, true
		}
		if it.Group != "" && it.Group != grp {
			if !afterHeader {
				body = append(body, "")
			}
			grp = it.Group
			body = append(body, m.styles.Muted.Render("  "+grp))
			afterHeader = true
		} else if it.Group == "" {
			grp = ""
		}
		icon := lipgloss.NewStyle().Width(3).Align(lipgloss.Center).Render(it.Icon)
		sel := idx == m.selected
		for li, tl := range tlines[idx] {
			var row string
			if li == 0 {
				row = icon + " " + padRight(tl, titleW)
				if keysOf[idx] != "" {
					row += "  " + keysOf[idx]
				}
			} else {
				row = "    " + padRight(tl, titleW)
			}
			if sel {
				body = append(body, m.styles.Panel.Render(m.styles.Text.Render("┃ "+row)))
			} else {
				body = append(body, m.styles.Muted.Render("  "+row))
			}
		}
		afterHeader = false
	}
	return body
}

// wrapText greedly wraps s to display width w, hard-splitting overlong words.
// It is rune-width aware so CJK/wide glyphs wrap correctly.
func wrapText(s string, w int) []string {
	if w < 1 {
		w = 1
	}
	if runewidth.StringWidth(s) <= w {
		return []string{s}
	}
	splitWord := func(word string) []string {
		var parts []string
		var cur strings.Builder
		curW := 0
		for _, r := range word {
			rw := runewidth.RuneWidth(r)
			if rw > w {
				rw = w
			}
			if curW+rw > w && curW > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
				curW = 0
			}
			cur.WriteRune(r)
			curW += rw
		}
		if curW > 0 {
			parts = append(parts, cur.String())
		}
		return parts
	}
	var lines []string
	var cur strings.Builder
	curW := 0
	flush := func() {
		if curW > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
	}
	for _, word := range strings.Fields(s) {
		if runewidth.StringWidth(word) > w {
			flush()
			lines = append(lines, splitWord(word)...)
			continue
		}
		ww := runewidth.StringWidth(word)
		add := ww
		if curW > 0 {
			add++
		}
		if curW+add > w {
			flush()
		}
		if curW > 0 {
			cur.WriteByte(' ')
			curW++
		}
		cur.WriteString(word)
		curW += ww
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
