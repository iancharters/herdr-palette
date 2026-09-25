package plugins

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/iancharters/herdr-palette/internal/herdr"
	"github.com/iancharters/herdr-palette/internal/model"
)

type DiscoveredAction struct {
	PluginID string `json:"plugin_id"`
	ActionID string `json:"action_id"`
	Title    string `json:"title"`
}

func selfPlugin() string {
	if v := os.Getenv("HERDR_PLUGIN_ID"); v != "" {
		return v
	}
	return "iancharters.herdr-palette"
}

// QualifiedID is what Herdr accepts for invoke (action ids may contain dots).
func QualifiedID(pluginID, actionID string) string { return pluginID + "." + actionID }

// ToPaletteItems maps server action list entries to runnable palette items.
func ToPaletteItems(actions []DiscoveredAction, shortcuts map[string]string) []model.PaletteItem {
	self := selfPlugin()
	out := []model.PaletteItem{}
	for _, a := range actions {
		if a.PluginID == "" || a.ActionID == "" || a.PluginID == self {
			continue
		}
		qid := QualifiedID(a.PluginID, a.ActionID)
		title := a.Title
		if title == "" {
			title = qid
		}
		var sc []string
		if b, ok := shortcuts[qid]; ok {
			if b != "" {
				sc = []string{b}
			} else {
				sc = []string{}
			}
		} else {
			sc = []string{}
		}
		out = append(out, model.PaletteItem{
			ID: "plugin:" + qid, Title: title, Category: model.CategoryCustom, Group: a.PluginID,
			Description: qid, Icon: "◆",
			Aliases:     []string{qid, a.PluginID, a.ActionID},
			Shortcuts:   sc,
			Invocation:  model.Invocation{Kind: model.InvocationHerdr, Argv: []string{"plugin", "action", "invoke", qid}},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Title < out[j].Title
	})
	return out
}

// ParseActionList parses `herdr plugin action list` stdout, fail-open to nil.
func ParseActionList(stdout string) []DiscoveredAction {
	var env struct {
		Result *struct {
			Actions []DiscoveredAction `json:"actions"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(stdout), &env) != nil || env.Result == nil {
		return nil
	}
	return env.Result.Actions
}

var tomlEscapes = map[string]string{"\\": "\\", `"`: `"`, "t": "\t", "n": "\n", "r": "\r", "b": "\b", "f": "\f"}

func unescapeToml(v string) string {
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

var kvRe = regexp.MustCompile(`(?m)^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"([^"]*)"`)

// ParsePluginActionShortcuts scans [[keys.command]] blocks for plugin_action bindings.
func ParsePluginActionShortcuts(source string) map[string]string {
	out := map[string]string{}
	blocks := regexp.MustCompile(`(?m)^\[\[keys\.command\]\]`).Split(source, -1)
	for _, block := range blocks[1:] {
		if idx := regexp.MustCompile(`(?m)^\[`).FindStringIndex(block); idx != nil {
			block = block[:idx[0]]
		}
		get := func(k string) (string, bool) {
			m := regexp.MustCompile(`(?m)^` + k + `\s*=\s*"([^"]*)"`).FindStringSubmatch(block)
			if m == nil {
				return "", false
			}
			return m[1], true
		}
		typ, ok1 := get("type")
		cmd, ok2 := get("command")
		key, ok3 := get("key")
		_ = kvRe
		if ok1 && ok2 && ok3 && typ == "plugin_action" {
			out[cmd] = unescapeToml(key)
		}
	}
	return out
}

// LoadPluginActions queries the live server; fail-open to nil.
func LoadPluginActions(shortcuts map[string]string) []model.PaletteItem {
	res := herdr.RunHerdr("plugin", "action", "list")
	if res.Code != 0 {
		return nil
	}
	return ToPaletteItems(ParseActionList(res.Stdout), shortcuts)
}
