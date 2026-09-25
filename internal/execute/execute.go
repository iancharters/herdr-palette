package execute

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/iancharters/herdr-palette/internal/herdr"
	"github.com/iancharters/herdr-palette/internal/model"
)

// Resolve maps an invocation (+ prompt input) to herdr argv or a message.
func Resolve(inv model.Invocation, input string) ([]string, string) {
	if inv.Kind == model.InvocationHerdr {
		return inv.Argv, ""
	}
	if inv.Kind == model.InvocationShortcut {
		return nil, ""
	}
	target := herdr.SessionTarget()
	if target == nil {
		return nil, "Herdr did not report the pane that opened the palette."
	}
	return resolveAction(inv.Action, inv.Step, *target, input)
}

func needInput(input, label string) string {
	if strings.TrimSpace(input) == "" {
		return "Enter a " + label + "."
	}
	return ""
}

func resolveAction(act model.ResolveAction, step int, t model.SessionTarget, input string) ([]string, string) {
	switch act {
	case model.ResolveClosePane:
		return []string{"pane", "close", t.PaneID}, ""
	case model.ResolveCloseTab:
		return []string{"tab", "close", t.TabID}, ""
	case model.ResolveCloseWorkspace:
		return []string{"workspace", "close", t.WorkspaceID}, ""
	case model.ResolveFocusTab:
		s := step
		if s == 0 {
			s = 1
		}
		id, msg := herdr.NeighborTab(t, s)
		if msg != "" {
			return nil, msg
		}
		return []string{"tab", "focus", id}, ""
	case model.ResolveFocusWorkspace:
		s := step
		if s == 0 {
			s = 1
		}
		id, msg := herdr.NeighborWorkspace(t, s)
		if msg != "" {
			return nil, msg
		}
		return []string{"workspace", "focus", id}, ""
	case model.ResolveFocusAgent:
		s := step
		if s == 0 {
			s = 1
		}
		id, msg := herdr.NeighborAgent(t, s)
		if msg != "" {
			return nil, msg
		}
		return []string{"agent", "focus", id}, ""
	case model.ResolveRenamePane:
		if m := needInput(input, "pane name"); m != "" {
			return nil, m
		}
		return []string{"pane", "rename", t.PaneID, strings.TrimSpace(input)}, ""
	case model.ResolveRenamePaneClear:
		return []string{"pane", "rename", t.PaneID, "--clear"}, ""
	case model.ResolveRenameTab:
		if m := needInput(input, "tab name"); m != "" {
			return nil, m
		}
		return []string{"tab", "rename", t.TabID, strings.TrimSpace(input)}, ""
	case model.ResolveRenameWorkspace:
		if m := needInput(input, "workspace name"); m != "" {
			return nil, m
		}
		return []string{"workspace", "rename", t.WorkspaceID, strings.TrimSpace(input)}, ""
	case model.ResolveMovePaneNewTab:
		return []string{"pane", "move", t.PaneID, "--new-tab", "--focus"}, ""
	case model.ResolveMovePaneNewWksp:
		return []string{"pane", "move", t.PaneID, "--new-workspace", "--focus"}, ""
	case model.ResolveWorktreeCreate:
		return []string{"worktree", "create", "--workspace", t.WorkspaceID, "--focus"}, ""
	case model.ResolveWorktreeOpen:
		if m := needInput(input, "branch or path"); m != "" {
			return nil, m
		}
		return herdr.WorktreeOpenArgv(t.WorkspaceID, input), ""
	case model.ResolveWorktreeRemove:
		if strings.ToLower(strings.TrimSpace(input)) != "yes" {
			return nil, `Type "yes" to remove this worktree.`
		}
		return []string{"worktree", "remove", "--workspace", t.WorkspaceID}, ""
	}
	return nil, "Unknown action."
}

// Execute runs a palette item; ok=true with empty output means close the
// palette, ok=true with output means show the result view.
func Execute(item model.PaletteItem, input string) model.CommandResult {
	if item.Invocation.Kind == model.InvocationShortcut {
		keys := strings.Join(item.Shortcuts, " / ")
		if keys != "" {
			return model.CommandResult{Message: "Press " + keys + " — Herdr only runs this one from the keyboard."}
		}
		return model.CommandResult{Message: "Herdr only runs this one from the keyboard."}
	}
	argv, msg := Resolve(item.Invocation, input)
	if msg != "" {
		return model.CommandResult{Message: msg}
	}
	// Plugin action invoke is fire-and-forget: exit 0 only means "accepted".
	// Follow the run's plugin log so output (e.g. snapshot lists) and real
	// failures surface in the palette instead of vanishing with the overlay.
	if isPluginInvoke(argv) {
		return invokePluginAction(item.Title, argv)
	}
	res := herdr.RunHerdr(argv...)
	if res.Code == 0 {
		return model.CommandResult{OK: true}
	}
	return model.CommandResult{Message: herdr.Explain(res.Stderr, res.Code)}
}

func isPluginInvoke(argv []string) bool {
	return len(argv) == 4 && argv[0] == "plugin" && argv[1] == "action" && argv[2] == "invoke"
}

// ParseInvokeResponse extracts the dispatched run's log_id and plugin_id.
func ParseInvokeResponse(stdout string) (logID, pluginID string) {
	var env struct {
		Result *struct {
			Log *struct {
				LogID    string `json:"log_id"`
				PluginID string `json:"plugin_id"`
			} `json:"log"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(stdout), &env) != nil || env.Result == nil || env.Result.Log == nil {
		return "", ""
	}
	return env.Result.Log.LogID, env.Result.Log.PluginID
}

type logEntry struct {
	LogID    string `json:"log_id"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// FindLogEntry pulls one run out of `plugin log list` output.
func FindLogEntry(logListJSON, logID string) *logEntry {
	var env struct {
		Result *struct {
			Logs []logEntry `json:"logs"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(logListJSON), &env) != nil || env.Result == nil {
		return nil
	}
	for i := range env.Result.Logs {
		if env.Result.Logs[i].LogID == logID {
			return &env.Result.Logs[i]
		}
	}
	return nil
}

func invokePluginAction(title string, argv []string) model.CommandResult {
	res := herdr.RunHerdr(argv...)
	if res.Code != 0 {
		return model.CommandResult{Message: herdr.Explain(res.Stderr, res.Code)}
	}
	logID, pluginID := ParseInvokeResponse(res.Stdout)
	if logID == "" || pluginID == "" {
		return model.CommandResult{OK: true} // unexpected shape: don't make a working invoke look broken
	}
	// Poll until the run finishes; a still-running action at the deadline is
	// assumed to be a healthy long-running one — close and leave it alone.
	for range 25 {
		time.Sleep(200 * time.Millisecond)
		list := herdr.RunHerdr("plugin", "log", "list", "--plugin", pluginID, "--limit", "20")
		if list.Code != 0 {
			continue
		}
		entry := FindLogEntry(list.Stdout, logID)
		if entry == nil {
			continue
		}
		switch entry.Status {
		case "succeeded":
			out := strings.TrimRight(entry.Stdout, "\n")
			if out == "" {
				return model.CommandResult{OK: true}
			}
			return model.CommandResult{OK: true, Output: out, Title: title}
		case "failed":
			code := "?"
			if entry.ExitCode != nil {
				code = itoa(*entry.ExitCode)
			}
			msg := strings.TrimSpace(entry.Stderr)
			if msg == "" {
				msg = "action failed (exit " + code + ")"
			} else {
				msg = "action failed (exit " + code + "): " + msg
			}
			// A failed run may still have useful stdout (partial results).
			if out := strings.TrimRight(entry.Stdout, "\n"); out != "" {
				return model.CommandResult{Output: out, Title: title, Message: msg}
			}
			return model.CommandResult{Message: msg}
		}
	}
	return model.CommandResult{OK: true}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
