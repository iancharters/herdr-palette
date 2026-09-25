package catalog

import (
	"testing"

	"github.com/iancharters/herdr-palette/internal/model"
)

func TestRunnableAndShortcutSplit(t *testing.T) {
	var runnable, shortcuts []string
	for _, it := range DefaultItems() {
		if it.Invocation.Kind == model.InvocationShortcut {
			shortcuts = append(shortcuts, it.ID)
		} else {
			runnable = append(runnable, it.ID)
		}
	}
	need := []string{"rename_pane", "close_tab", "previous_workspace", "resize_pane_left", "move_pane_new_tab", "new_worktree"}
	for _, id := range need {
		found := false
		for _, r := range runnable {
			if r == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("runnable %s missing", id)
		}
	}
	if len(shortcuts) != 7 {
		t.Fatalf("shortcuts %v", shortcuts)
	}
}

func TestPrompted(t *testing.T) {
	var prompted []string
	for _, it := range DefaultItems() {
		if it.Prompt != nil {
			prompted = append(prompted, it.ID)
		}
	}
	want := []string{"rename_workspace", "rename_tab", "rename_pane", "open_worktree", "remove_worktree"}
	if len(prompted) != len(want) {
		t.Fatalf("prompted %v", prompted)
	}
	for i := range want {
		if prompted[i] != want[i] {
			t.Fatalf("prompted %v", prompted)
		}
	}
}

func TestIconsAreSingleCellSafe(t *testing.T) {
	// Geometric block glyphs (U+25xx) render double-width in some terminals
	// while arrows render single-width, which breaks icon column alignment.
	// Icons must stay in the safe set: ASCII plus arrows/latin-1/×.
	for _, it := range DefaultItems() {
		for _, r := range it.Icon {
			if r > 0x2500 && r < 0x2600 {
				t.Fatalf("item %s icon %q is a geometric block glyph", it.ID, it.Icon)
			}
		}
		if got := len([]rune(it.Icon)); got != 1 {
			t.Fatalf("item %s icon %q is not one rune", it.ID, it.Icon)
		}
	}
}
