package execute

import (
	"strings"

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

// Execute runs a palette item; ok=true means close the palette.
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
	res := herdr.RunHerdr(argv...)
	if res.Code == 0 {
		return model.CommandResult{OK: true}
	}
	return model.CommandResult{Message: herdr.Explain(res.Stderr, res.Code)}
}
