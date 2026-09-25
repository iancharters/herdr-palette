package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/iancharters/herdr-palette/internal/model"
	"github.com/iancharters/herdr-palette/internal/theme"
)

func TestPadRightAlignsDisplayWidth(t *testing.T) {
	a, b := "←  Focus", "|  Split pane right"
	w := max(runewidth.StringWidth(a), runewidth.StringWidth(b))
	if runewidth.StringWidth(padRight(a, w)) != w || runewidth.StringWidth(padRight(b, w)) != w {
		t.Fatalf("widths %d %d, want %d", runewidth.StringWidth(padRight(a, w)), runewidth.StringWidth(padRight(b, w)), w)
	}
	if got := padRight("abc", 2); got != "abc" {
		t.Fatalf("should not truncate, got %q", got)
	}
}

func TestWrapTextFitsAndWraps(t *testing.T) {
	if got := wrapText("short", 10); len(got) != 1 || got[0] != "short" {
		t.Fatalf("%q", got)
	}
	got := wrapText("Resurrect: preview restore (dry run)", 16)
	if len(got) < 2 {
		t.Fatalf("expected wrap, got %q", got)
	}
	for _, ln := range got {
		if runewidth.StringWidth(ln) > 16 {
			t.Fatalf("line %q exceeds width", ln)
		}
	}
	// overlong words hard-split
	if got := wrapText("supercalifragilistic", 8); len(got) < 2 {
		t.Fatalf("expected hard split, got %q", got)
	}
}

func TestViewPinsFooterAndKeepsKeysOnFirstLine(t *testing.T) {
	items := []model.PaletteItem{
		{ID: "a", Title: "A very long command title that must wrap somewhere", Category: "Custom", Group: "g", Icon: "*", Shortcuts: []string{"prefix+x"}, Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"x"}}},
		{ID: "b", Title: "Short", Category: "Custom", Group: "g", Icon: "*", Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"y"}}},
	}
	m := New(items, theme.StylesFor(theme.PaletteTheme{}), func(it model.PaletteItem, _ string) model.CommandResult {
		return model.CommandResult{OK: true}
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m = updated.(Model)
	view := m.View()
	rows := strings.Split(view, "\n")
	// footer pinned: last row mentions the count
	if !strings.Contains(rows[len(rows)-1], "commands") {
		t.Fatalf("footer not last: %q", rows[len(rows)-1])
	}
	// keys stay on the wrapped row's first line
	found := false
	for _, r := range rows {
		if strings.Contains(r, "prefix+x") && strings.Contains(r, "A very long") {
			found = true
		}
	}
	if !found {
		t.Fatalf("keys not top-aligned:\n%s", view)
	}
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func TestRowsShareColumnOffsets(t *testing.T) {
	items := []model.PaletteItem{
		{ID: "a", Title: "New workspace", Category: "Workspace", Icon: "+", Shortcuts: []string{"prefix+shift+n"}, Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"x"}}},
		{ID: "b", Title: "A much longer workspace title here", Category: "Workspace", Icon: "+", Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"y"}}},
		{ID: "c", Title: "Close", Category: "Workspace", Icon: "x", Shortcuts: []string{"prefix+x"}, Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"z"}}},
	}
	m := New(items, theme.StylesFor(theme.PaletteTheme{}), func(it model.PaletteItem, _ string) model.CommandResult {
		return model.CommandResult{OK: true}
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = updated.(Model)
	type row struct {
		line string
		lead int // leading spaces
	}
	var rows []row
	for _, r := range strings.Split(stripANSI(m.View()), "\n") {
		if !strings.Contains(r, "workspace") && !strings.Contains(r, "Close") {
			continue
		}
		// skip wrapped continuation lines (indented, no icon, no keys)
		if !strings.Contains(r, "+") && !strings.Contains(r, "x") && !strings.Contains(r, "prefix") {
			continue
		}
		lead := 0
		for _, c := range r {
			if c != ' ' {
				break
			}
			lead++
		}
		rows = append(rows, row{r, lead})
	}
	if len(rows) < 3 {
		t.Fatalf("want 3 first-line rows, got %d", len(rows))
	}
	// shared block offset = smallest indent (a selected row's ┃ marker
	// occupies its first two cells, so its indent is the bare offset)
	offset := rows[0].lead
	for _, r := range rows[1:] {
		if r.lead < offset {
			offset = r.lead
		}
	}
	keysX := -1
	for _, r := range rows {
		cells := []rune(r.line)
		// icon sits after the 2-cell marker + 1 centering space, for all rows
		if offset+3 >= len(cells) || cells[offset+3] == ' ' {
			t.Fatalf("no icon at shared column in %q (offset %d)", r.line, offset)
		}
		if k := strings.Index(r.line, "prefix"); k >= 0 {
			// byte index → display cells (┃ is 3 bytes in 1 cell)
			kx := runewidth.StringWidth(r.line[:k])
			if keysX < 0 {
				keysX = kx
			} else if kx != keysX {
				t.Fatalf("keys x %d != %d in %q", kx, keysX, r.line)
			}
		}
	}
	if keysX < 0 {
		t.Fatal("found no keybinds to compare")
	}
}

func TestViewIsExactlyFrameHeight(t *testing.T) {
	mkItems := func() []model.PaletteItem {
		return []model.PaletteItem{
			{ID: "a", Title: "New workspace", Category: "Workspace", Icon: "+", Shortcuts: []string{"prefix+shift+n"}, Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"x"}}},
			{ID: "b", Title: "A much longer workspace title here that wraps", Category: "Workspace", Icon: "+", Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"y"}}},
			{ID: "c", Title: "Close", Category: "Tabs", Icon: "x", Shortcuts: []string{"prefix+x"}, Invocation: model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"z"}}},
		}
	}
	newM := func() Model {
		return New(mkItems(), theme.StylesFor(theme.PaletteTheme{}), func(it model.PaletteItem, _ string) model.CommandResult {
			return model.CommandResult{OK: true}
		})
	}
	for _, h := range []int{14, 24, 40} {
		m := newM()
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: h})
		m = updated.(Model)
		if n := len(strings.Split(m.View(), "\n")); n != h {
			t.Fatalf("height %d: view has %d lines", h, n)
		}
		// empty-filter state too
		m.query = "zzz-no-match"
		m.input.SetValue("zzz-no-match")
		if n := len(strings.Split(stripANSI(m.View()), "\n")); n != h {
			t.Fatalf("height %d empty: view has %d lines", h, n)
		}
	}
}
