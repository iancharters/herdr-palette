package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iancharters/herdr-palette/internal/catalog"
	"github.com/iancharters/herdr-palette/internal/model"
	"github.com/iancharters/herdr-palette/internal/plugins"
)

var tomlEscapes = map[string]string{"\\": "\\", `"`: `"`, "t": "\t", "n": "\n", "r": "\r", "b": "\b", "f": "\f"}

// UnescapeToml decodes the TOML basic-string escapes we care about.
func UnescapeToml(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' && i+1 < len(v) {
			if r, ok := tomlEscapes[string(v[i+1])]; ok {
				b.WriteString(r)
			} else {
				b.WriteByte('\\')
				b.WriteByte(v[i+1])
			}
			i++
			continue
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

var remapRe = regexp.MustCompile(`(?m)^([a-z_]+)\s*=\s*"([^"]*)"`)

// ParseKeyRemaps reads bare `name = "binding"` assignments.
func ParseKeyRemaps(source string) map[string]string {
	out := map[string]string{}
	for _, m := range remapRe.FindAllStringSubmatch(source, -1) {
		out[m[1]] = UnescapeToml(m[2])
	}
	return out
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

// LoadPaletteItems merges hardcoded built-ins (with remaps) and live plugin discovery (fail-open).
func LoadPaletteItems(path string) []model.PaletteItem {
	if path == "" {
		path = defaultConfigPath()
	}
	items := catalog.DefaultItems()
	var source string
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			source = string(b)
		}
	}
	remaps := ParseKeyRemaps(source)
	withRemaps := make([]model.PaletteItem, 0, len(items))
	for _, it := range items {
		if r, ok := remaps[it.ID]; ok {
			if r != "" {
				it.Shortcuts = []string{r}
			} else {
				it.Shortcuts = []string{}
			}
		}
		withRemaps = append(withRemaps, it)
	}
	discovered := plugins.LoadPluginActions(plugins.ParsePluginActionShortcuts(source))
	return append(withRemaps, discovered...)
}
