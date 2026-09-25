package ui

import (
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
