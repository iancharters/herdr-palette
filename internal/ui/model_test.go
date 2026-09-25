package ui

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestPadRightAlignsDisplayWidth(t *testing.T) {
	a, b := "←  Focus", "▯  Split pane right"
	w := max(runewidth.StringWidth(a), runewidth.StringWidth(b))
	if runewidth.StringWidth(padRight(a, w)) != w || runewidth.StringWidth(padRight(b, w)) != w {
		t.Fatalf("widths %d %d, want %d", runewidth.StringWidth(padRight(a, w)), runewidth.StringWidth(padRight(b, w)), w)
	}
	if got := padRight("abc", 2); got != "abc" {
		t.Fatalf("should not truncate, got %q", got)
	}
}
