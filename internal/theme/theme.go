// Package theme resolves popup colors from Herdr's config while defaulting to
// the host terminal's own theme: detection is dynamic (termenv dark/light
// background probe) and all styles use adaptive colors, so the popup follows
// terminal light/dark switches instead of fighting them with hardcoded hex.
package theme

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

//go:embed palettes.json
var palettesRaw []byte

type palettesFile struct {
	Tokens  []string          `json:"tokens"`
	Aliases map[string]string `json:"aliases"`
	Themes  map[string]map[string]string `json:"themes"`
}

var palettes palettesFile

func init() {
	_ = json.Unmarshal(palettesRaw, &palettes)
}

// PaletteTheme mirrors the TS slot names (hex or "" for terminal default).
type PaletteTheme struct {
	Background string
	Panel      string
	Text       string
	Muted      string
	Accent     string
	Shortcut   string
	Footer     string
	FooterText string
}

var FallbackTheme = PaletteTheme{
	Background: "#181825", Panel: "#313244", Text: "#cdd6f4", Muted: "#a6adc8",
	Accent: "#89b4fa", Shortcut: "#94e2d5", Footer: "#1e1e2e", FooterText: "#6c7086",
}

// HasDarkBackground reports the terminal's current background luminance.
// termenv probes COLORFGBG / OSC 11; default to dark (most terminal users).
func HasDarkBackground() bool { return termenv.HasDarkBackground() }

// Styles are lipgloss styles built on adaptive colors so light/dark flips
// follow the terminal. Explicit Herdr hex overrides pin a slot; otherwise the
// adaptive pair (terminal-aware) wins.
type Styles struct {
	Background lipgloss.Style
	Panel      lipgloss.Style // selected row
	Text       lipgloss.Style
	Muted      lipgloss.Style
	Accent     lipgloss.Style
	Shortcut   lipgloss.Style
	Footer     lipgloss.Style
	FooterText lipgloss.Style
}

func adaptive(dark, light string, explicit string) lipgloss.TerminalColor {
	if explicit != "" {
		return lipgloss.Color(explicit)
	}
	return lipgloss.AdaptiveColor{Dark: dark, Light: light}
}

// StylesFor builds rendering styles from a resolved PaletteTheme.
// Empty slots mean "terminal default" (no background/foreground set).
func StylesFor(t PaletteTheme) Styles {
	fg := func(explicit, dark, light string) lipgloss.Style {
		if explicit != "" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(explicit))
		}
		return lipgloss.NewStyle().Foreground(adaptive(dark, light, ""))
	}
	bg := func(explicit, dark, light string) lipgloss.Style {
		if explicit != "" {
			return lipgloss.NewStyle().Background(lipgloss.Color(explicit))
		}
		return lipgloss.NewStyle().Background(adaptive(dark, light, ""))
	}
	_ = bg
	return Styles{
		Background: lipgloss.NewStyle(),
		Panel:      lipgloss.NewStyle().Background(adaptive("#313244", "#e4e4e4", t.Panel)),
		Text:       fg(t.Text, "#cdd6f4", "#1a1a1a"),
		Muted:      fg(t.Muted, "#a6adc8", "#6f6f6f"),
		Accent:     fg(t.Accent, "#89b4fa", "#0000ee"),
		Shortcut:   fg(t.Shortcut, "#94e2d5", "#008080"),
		Footer:     lipgloss.NewStyle().Background(adaptive("#1e1e2e", "#f2f2f2", t.Footer)),
		FooterText: fg(t.FooterText, "#6c7086", "#6f6f6f"),
	}
}

// --- Herdr config resolution (port of theme.ts) ---

func NormalizeThemeName(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = regexp.MustCompile(`[ _]+`).ReplaceAllString(key, "-")
	if a, ok := palettes.Aliases[key]; ok {
		return a
	}
	return key
}

var themePairs = [][2]string{
	{"catppuccin", "catppuccin-latte"},
	{"tokyo-night", "tokyo-night-day"},
	{"gruvbox", "gruvbox-light"},
	{"one-dark", "one-light"},
	{"solarized", "solarized-light"},
	{"kanagawa", "kanagawa-lotus"},
	{"rose-pine", "rose-pine-dawn"},
}

func themePair(name string) (dark, light string) {
	c := NormalizeThemeName(name)
	for _, p := range themePairs {
		if p[0] == c || p[1] == c {
			return p[0], p[1]
		}
	}
	return c, c
}

func isLightName(name string) bool {
	c := NormalizeThemeName(name)
	for _, p := range themePairs {
		if p[1] == c {
			return true
		}
	}
	return false
}

var namedColors = map[string]string{
	"black": "#000000", "red": "#cd0000", "green": "#00cd00", "yellow": "#cdcd00", "blue": "#0000ee",
	"magenta": "#cd00cd", "purple": "#cd00cd", "cyan": "#00cdcd", "white": "#e5e5e5",
	"gray": "#e5e5e5", "grey": "#e5e5e5", "darkgray": "#7f7f7f", "darkgrey": "#7f7f7f",
	"lightred": "#ff0000", "lightgreen": "#00ff00", "lightyellow": "#ffff00", "lightblue": "#5c5cff",
	"lightmagenta": "#ff00ff", "lightcyan": "#00ffff",
}

var hexRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)
var shortRe = regexp.MustCompile(`^#([0-9a-f])([0-9a-f])([0-9a-f])$`)
var rgbRe = regexp.MustCompile(`^rgb\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*\)$`)
var resetAliases = map[string]bool{"reset": true, "default": true, "none": true, "transparent": true}

func toHex2(n int) string {
	const h = "0123456789abcdef"
	return string([]byte{h[n/16], h[n%16]})
}

func atoi3(s string) (int, bool) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, n < 256
}

// ResolveColor maps a Herdr color value to hex; "" means terminal default.
func ResolveColor(value string, namedAsReset bool) string {
	t := strings.ToLower(strings.TrimSpace(value))
	if hexRe.MatchString(t) {
		return t
	}
	if m := shortRe.FindStringSubmatch(t); m != nil {
		return "#" + m[1] + m[1] + m[2] + m[2] + m[3] + m[3]
	}
	if m := rgbRe.FindStringSubmatch(t); m != nil {
		r, okr := atoi3(m[1])
		g, okg := atoi3(m[2])
		b, okb := atoi3(m[3])
		if okr && okg && okb {
			return "#" + toHex2(r) + toHex2(g) + toHex2(b)
		}
		return ""
	}
	if resetAliases[t] {
		return ""
	}
	if namedAsReset {
		return ""
	}
	return namedColors[t]
}

type ThemeSelection struct {
	Name       string
	AutoSwitch bool
	DarkName   string
	LightName  string
}

type ThemeConfig struct {
	Selection   *ThemeSelection
	Custom      map[string]string
	CustomLight map[string]string
	CustomDark  map[string]string
}

var sectionRe = regexp.MustCompile(`^\s*\[([^\]]+)\]\s*(?:#.*)?$`)
var entryRe = regexp.MustCompile(`^\s*([a-z_][a-z0-9_]*)\s*=\s*(.*)$`)

func tomlValue(raw string) string {
	t := strings.TrimSpace(raw)
	if len(t) >= 2 && (t[0] == '"' || t[0] == '\'') {
		q := t[0]
		for i := 1; i < len(t); i++ {
			if t[i] == q {
				return strings.TrimSpace(t[1:i])
			}
		}
	}
	if idx := strings.Index(t, " #"); idx >= 0 {
		t = t[:idx]
	}
	return strings.TrimSpace(t)
}

func inTokens(k string) bool {
	for _, t := range palettes.Tokens {
		if t == k {
			return true
		}
	}
	return false
}

// ParseThemeConfig reads [theme] and [theme.custom*] sections.
func ParseThemeConfig(source string) ThemeConfig {
	cfg := ThemeConfig{Custom: map[string]string{}, CustomLight: map[string]string{}, CustomDark: map[string]string{}}
	section := ""
	var name, darkName, lightName string
	auto := false
	hasName := false
	for _, line := range strings.Split(source, "\n") {
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			section = strings.TrimSpace(m[1])
			continue
		}
		if section != "theme" && !strings.HasPrefix(section, "theme.custom") {
			continue
		}
		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], tomlValue(m[2])
		if val == "" {
			continue
		}
		switch section {
		case "theme.custom":
			cfg.Custom[key] = val
		case "theme.custom.light":
			if inTokens(key) {
				cfg.CustomLight[key] = val
			}
		case "theme.custom.dark":
			if inTokens(key) {
				cfg.CustomDark[key] = val
			}
		case "theme":
			switch key {
			case "name":
				name, hasName = val, true
			case "auto_switch":
				auto = val == "true"
			case "dark_name":
				darkName = val
			case "light_name":
				lightName = val
			}
		}
	}
	if hasName {
		cfg.Selection = &ThemeSelection{Name: name, AutoSwitch: auto, DarkName: darkName, LightName: lightName}
	}
	return cfg
}

func effectiveThemeName(sel ThemeSelection, hostLight bool) string {
	if !sel.AutoSwitch {
		return NormalizeThemeName(sel.Name)
	}
	dark, light := themePair(sel.Name)
	if hostLight {
		if sel.LightName != "" {
			return NormalizeThemeName(sel.LightName)
		}
		return NormalizeThemeName(light)
	}
	if sel.DarkName != "" {
		return NormalizeThemeName(sel.DarkName)
	}
	return NormalizeThemeName(dark)
}

func builtinTheme(name string) map[string]string { return palettes.Themes[NormalizeThemeName(name)] }

func composeTheme(tokens map[string]string) PaletteTheme {
	get := func(tok string) string {
		if v, ok := tokens[tok]; ok && hexRe.MatchString(strings.ToLower(strings.TrimSpace(v))) {
			return strings.ToLower(strings.TrimSpace(v))
		}
		return ""
	}
	return PaletteTheme{
		Background: get("panel_bg"),
		Panel:      get("surface0"),
		Text:       get("text"),
		Muted:      get("subtext0"),
		Accent:     get("accent"),
		Shortcut:   get("teal"),
		Footer:     get("sidebar_bg"),
		FooterText: get("overlay0"),
	}
}

// ThemeFor layers builtin + custom + mode overrides; "" slots = terminal default.
func ThemeFor(name string, custom, modeCustom map[string]string) PaletteTheme {
	tokens := map[string]string{}
	merge := func(src map[string]string, namedAsReset bool) {
		for tok, val := range src {
			if !inTokens(tok) {
				continue
			}
			if r := ResolveColor(val, namedAsReset); r != "" {
				tokens[tok] = r
			} else {
				delete(tokens, tok)
			}
		}
	}
	base := builtinTheme(name)
	if base == nil {
		base = palettes.Themes["catppuccin"]
	}
	merge(base, NormalizeThemeName(name) == "terminal")
	merge(custom, false)
	merge(modeCustom, false)
	t := composeTheme(tokens)
	// Sidebar reset falls back to dim surface (popup floats over a pane).
	if t.Footer == "" {
		if v, ok := tokens["surface_dim"]; ok {
			t.Footer = v
		}
	}
	return t
}

func defaultConfigPath() string {
	if p := os.Getenv("HERDR_CONFIG_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "herdr", "config.toml")
}

// LoadTheme resolves the Herdr theme to explicit overrides; unset slots stay
// terminal-default so the popup dynamically follows the terminal theme.
// hostLight may be nil to auto-detect via termenv.
func LoadTheme(path string, hostLight *bool) PaletteTheme {
	if path == "" {
		path = defaultConfigPath()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return PaletteTheme{}
	}
	cfg := ParseThemeConfig(string(b))
	if cfg.Selection == nil {
		return PaletteTheme{}
	}
	light := false
	if hostLight != nil {
		light = *hostLight
	} else {
		light = !HasDarkBackground()
	}
	name := cfg.Selection.Name
	if cfg.Selection.AutoSwitch {
		name = effectiveThemeName(*cfg.Selection, light)
	} else {
		name = NormalizeThemeName(name)
	}
	mode := map[string]string{}
	if cfg.Selection.AutoSwitch {
		if light {
			mode = cfg.CustomLight
		} else {
			mode = cfg.CustomDark
		}
	}
	return ThemeFor(name, cfg.Custom, mode)
}
