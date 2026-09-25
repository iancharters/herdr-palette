package theme

import "testing"

func TestResolveColor(t *testing.T) {
	if got := ResolveColor("#ABCDEF", false); got != "#abcdef" {
		t.Fatalf("hex %q", got)
	}
	if got := ResolveColor("#abc", false); got != "#aabbcc" {
		t.Fatalf("short %q", got)
	}
	if got := ResolveColor("rgb(255, 0, 16)", false); got != "#ff0010" {
		t.Fatalf("rgb %q", got)
	}
	if got := ResolveColor("reset", false); got != "" {
		t.Fatalf("reset %q", got)
	}
	if got := ResolveColor("red", false); got != "#cd0000" {
		t.Fatalf("named %q", got)
	}
	if got := ResolveColor("red", true); got != "" {
		t.Fatalf("namedAsReset %q", got)
	}
}

func TestNormalizeThemeName(t *testing.T) {
	if NormalizeThemeName("Catppuccin Mocha") != "catppuccin" {
		t.Fatal("alias not applied")
	}
	if NormalizeThemeName("tokyo_night") != "tokyo-night" {
		t.Fatal("separator not normalized")
	}
}

func TestThemeForCustomOverride(t *testing.T) {
	th := ThemeFor("catppuccin", map[string]string{"accent": "#ff0000"}, map[string]string{})
	if th.Accent != "#ff0000" {
		t.Fatalf("custom accent %q", th.Accent)
	}
	// reset erases the base so the slot falls back to terminal default
	th = ThemeFor("catppuccin", map[string]string{"accent": "reset"}, map[string]string{})
	if th.Accent != "" {
		t.Fatalf("reset accent %q", th.Accent)
	}
}
