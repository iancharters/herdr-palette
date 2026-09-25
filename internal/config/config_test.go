package config

import "testing"

func TestParseKeyRemaps(t *testing.T) {
	m := ParseKeyRemaps("prefix = \"ctrl+a\"\nrename_workspace = \"prefix+shift+r\"\nopen_worktree = \"\"\n")
	if m["rename_workspace"] != "prefix+shift+r" || m["open_worktree"] != "" {
		t.Fatalf("%v", m)
	}
	if UnescapeToml("prefix+\\\\") != "prefix+\\" {
		t.Fatal("unescape")
	}
}

func TestLoadPaletteItemsLive(t *testing.T) {
	items := LoadPaletteItems("")
	if len(items) == 0 {
		t.Fatal("expected built-ins at least")
	}
	found := false
	for _, it := range items {
		if it.ID == "zoom" {
			found = true
		}
	}
	if !found {
		t.Fatal("built-in zoom missing")
	}
}
