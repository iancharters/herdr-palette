package herdr

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/iancharters/herdr-palette/internal/model"
)

func binary() string {
	if b := os.Getenv("HERDR_BIN_PATH"); b != "" {
		return b
	}
	return "herdr"
}

type runResult struct {
	Code   int
	Stdout string
	Stderr string
}

// RunHerdr spawns the herdr CLI and captures output.
func RunHerdr(argv ...string) runResult {
	cmd := exec.Command(binary(), argv...)
	var out, err bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err
	code := 0
	if e := cmd.Run(); e != nil {
		if ee, ok := e.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			return runResult{Code: 127, Stdout: "", Stderr: e.Error()}
		}
	}
	return runResult{Code: code, Stdout: out.String(), Stderr: err.String()}
}

// Explain mirrors the TS helper: Herdr reports server errors as JSON on stderr.
func Explain(stderr string, code int) string {
	trimmed := strings.TrimSpace(stderr)
	if trimmed != "" {
		var env struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(trimmed), &env) == nil && env.Error != nil && env.Error.Message != "" {
			return env.Error.Message
		}
		return trimmed
	}
	if code != 0 {
		return "Herdr exited with status " + itoa(code) + "."
	}
	return ""
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

// ParseLaunchContext reads HERDR_PLUGIN_CONTEXT_JSON for the summoning pane.
func ParseLaunchContext(jsonText string) *model.SessionTarget {
	if jsonText == "" {
		return nil
	}
	var ctx struct {
		FocusedPaneID string `json:"focused_pane_id"`
		TabID         string `json:"tab_id"`
		WorkspaceID   string `json:"workspace_id"`
	}
	if json.Unmarshal([]byte(jsonText), &ctx) != nil {
		return nil
	}
	if ctx.FocusedPaneID == "" || ctx.TabID == "" || ctx.WorkspaceID == "" {
		return nil
	}
	return &model.SessionTarget{PaneID: ctx.FocusedPaneID, TabID: ctx.TabID, WorkspaceID: ctx.WorkspaceID}
}

// SessionTarget resolves the pane the palette acts on.
func SessionTarget() *model.SessionTarget {
	if t := ParseLaunchContext(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON")); t != nil {
		return t
	}
	res := RunHerdr("pane", "current", "--current")
	if res.Code != 0 {
		return nil
	}
	var env struct {
		Result *struct {
			Pane *struct {
				PaneID      string `json:"pane_id"`
				TabID       string `json:"tab_id"`
				WorkspaceID string `json:"workspace_id"`
			} `json:"pane"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(res.Stdout), &env) != nil || env.Result == nil || env.Result.Pane == nil || env.Result.Pane.PaneID == "" {
		return nil
	}
	return &model.SessionTarget{PaneID: env.Result.Pane.PaneID, TabID: env.Result.Pane.TabID, WorkspaceID: env.Result.Pane.WorkspaceID}
}

// StepRing steps through an ordered ring of IDs, wrapping at the ends.
func StepRing(ids []string, current string, step int, alone, missing string) (string, string) {
	if len(ids) < 2 {
		return "", alone
	}
	idx := -1
	for i, id := range ids {
		if id == current {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", missing
	}
	return ids[(idx+step+len(ids))%len(ids)], ""
}

func NeighborTab(target model.SessionTarget, step int) (string, string) {
	res := RunHerdr("tab", "list", "--workspace", target.WorkspaceID)
	if res.Code != 0 {
		return "", Explain(res.Stderr, res.Code)
	}
	var env struct {
		Result *struct {
			Tabs []struct {
				TabID string `json:"tab_id"`
			} `json:"tabs"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(res.Stdout), &env) != nil || env.Result == nil {
		return "", "Herdr returned an unreadable tab list."
	}
	ids := make([]string, 0, len(env.Result.Tabs))
	for _, t := range env.Result.Tabs {
		ids = append(ids, t.TabID)
	}
	id, msg := StepRing(ids, target.TabID, step, "This workspace only has one tab.", "Herdr did not report the tab that opened the palette.")
	return id, msg
}

func NeighborWorkspace(target model.SessionTarget, step int) (string, string) {
	res := RunHerdr("workspace", "list")
	if res.Code != 0 {
		return "", Explain(res.Stderr, res.Code)
	}
	var env struct {
		Result *struct {
			Workspaces []struct {
				WorkspaceID string `json:"workspace_id"`
			} `json:"workspaces"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(res.Stdout), &env) != nil || env.Result == nil {
		return "", "Herdr returned an unreadable workspace list."
	}
	ids := make([]string, 0, len(env.Result.Workspaces))
	for _, w := range env.Result.Workspaces {
		ids = append(ids, w.WorkspaceID)
	}
	return StepRing(ids, target.WorkspaceID, step, "Only one workspace is open.", "Herdr did not report the workspace that opened the palette.")
}

func NeighborAgent(target model.SessionTarget, step int) (string, string) {
	res := RunHerdr("agent", "list")
	if res.Code != 0 {
		return "", Explain(res.Stderr, res.Code)
	}
	var env struct {
		Result *struct {
			Agents []struct {
				PaneID  string `json:"pane_id"`
				Focused bool   `json:"focused"`
			} `json:"agents"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(res.Stdout), &env) != nil || env.Result == nil {
		return "", "Herdr returned an unreadable agent list."
	}
	agents := env.Result.Agents
	if len(agents) == 0 {
		return "", "No agents are running."
	}
	if len(agents) < 2 {
		return "", "Only one agent is running."
	}
	idx := -1
	for i, a := range agents {
		if a.PaneID == target.PaneID {
			idx = i
			break
		}
	}
	if idx < 0 {
		for i, a := range agents {
			if a.Focused {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		idx = 0
	}
	return agents[(idx+step+len(agents))%len(agents)].PaneID, ""
}

// WorktreeOpenArgv prefers --path for filesystem-looking input, else --branch.
func WorktreeOpenArgv(workspaceID, input string) []string {
	v := strings.TrimSpace(input)
	flag := "--branch"
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "~") || strings.HasPrefix(v, ".") {
		flag = "--path"
	}
	return []string{"worktree", "open", "--workspace", workspaceID, flag, v, "--focus"}
}

// ForegroundProc is one entry of pane process-info.
type ForegroundProc struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
}

// ProcessInfo queries a pane's process state. Returns nil on any failure
// (callers must treat nil as "unknown", i.e. busy — never type into it).
func ProcessInfo(paneID string) []ForegroundProc {
	res := RunHerdr("pane", "process-info", "--pane", paneID)
	if res.Code != 0 {
		return nil
	}
	return ParseProcessInfo(res.Stdout)
}

// ParseProcessInfo extracts foreground processes from process-info JSON.
func ParseProcessInfo(stdout string) []ForegroundProc {
	var env struct {
		Result *struct {
			Info *struct {
				Procs []ForegroundProc `json:"foreground_processes"`
			} `json:"process_info"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(stdout), &env) != nil || env.Result == nil || env.Result.Info == nil {
		return nil
	}
	return env.Result.Info.Procs
}

var shells = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "fish": true,
	"nu": true, "nushell": true, "elvish": true, "xonsh": true, "xonsh.exe": true,
	"powershell": true, "powershell.exe": true, "pwsh": true, "pwsh.exe": true,
	"cmd": true, "cmd.exe": true,
}

// IsShell reports whether a pane is sitting at an interactive shell prompt.
// procs==nil means unknown (query failed) → false: callers split instead of
// typing into a possibly-running TUI. Empty means a fresh idle shell → true.
func IsShell(procs []ForegroundProc) bool {
	if procs == nil {
		return false
	}
	for _, p := range procs {
		if !shells[strings.ToLower(strings.TrimSpace(p.Name))] {
			return false
		}
	}
	return true
}

// SplitPane splits paneID vertically (right) and returns the new pane's id.
func SplitPane(paneID string, focus bool) (string, string) {
	args := []string{"pane", "split", paneID, "--direction", "right"}
	if focus {
		args = append(args, "--focus")
	}
	res := RunHerdr(args...)
	if res.Code != 0 {
		return "", Explain(res.Stderr, res.Code)
	}
	var env struct {
		Result *struct {
			Pane *struct {
				PaneID string `json:"pane_id"`
			} `json:"pane"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(res.Stdout), &env) != nil || env.Result == nil || env.Result.Pane == nil || env.Result.Pane.PaneID == "" {
		return "", "Herdr split the pane but did not report the new one."
	}
	return env.Result.Pane.PaneID, ""
}

// RunInPane runs a command in the given pane via its shell.
func RunInPane(paneID string, argv ...string) string {
	res := RunHerdr(append([]string{"pane", "run", paneID}, argv...)...)
	if res.Code != 0 {
		return Explain(res.Stderr, res.Code)
	}
	return ""
}
