package catalog

import "github.com/iancharters/herdr-palette/internal/model"

func entry(id, title string, cat model.Category, desc, icon, key string, inv model.Invocation, prompt *model.PromptSpec) model.PaletteItem {
	shortcuts := []string{}
	if key != "" {
		shortcuts = []string{key}
	}
	return model.PaletteItem{
		ID: id, Title: title, Category: cat, Description: desc, Icon: icon,
		Aliases: []string{}, Shortcuts: shortcuts, Invocation: inv, Prompt: prompt,
	}
}

func action(id, title string, cat model.Category, desc, icon, shortcut string, argv []string) model.PaletteItem {
	return entry(id, title, cat, desc, icon, shortcut, model.Invocation{Kind: model.InvocationHerdr, Argv: argv}, nil)
}

func resolve(id, title string, cat model.Category, desc, icon, shortcut string, act model.ResolveAction, step int, hasStep bool, prompt *model.PromptSpec) model.PaletteItem {
	return entry(id, title, cat, desc, icon, shortcut, model.Invocation{Kind: model.InvocationResolve, Action: act, Step: step, HasStep: hasStep}, prompt)
}

func shortcut(id, title string, cat model.Category, desc, icon, key string) model.PaletteItem {
	return entry(id, title, cat, desc, icon, key, model.Invocation{Kind: model.InvocationShortcut}, nil)
}

func step(s int) (int, bool) { return s, true }

// DefaultItems mirrors the TS catalog one-to-one (ids, titles, argv).
func DefaultItems() []model.PaletteItem {
	name := &model.PromptSpec{Placeholder: "New name"}
	branchOrPath := &model.PromptSpec{Placeholder: "Branch or path"}
	confirmRemove := &model.PromptSpec{Placeholder: `Type "yes" to confirm`}
	s1, h1 := step(-1)
	s2, h2 := step(1)

	return []model.PaletteItem{
		action("new_workspace", "New workspace", model.CategoryWorkspace, "Create and focus a workspace", "+", "prefix+shift+n", []string{"workspace", "create", "--focus"}),
		resolve("rename_workspace", "Rename workspace", model.CategoryWorkspace, "Rename the current workspace", "+", "prefix+shift+w", model.ResolveRenameWorkspace, 0, false, name),
		resolve("close_workspace", "Close workspace", model.CategoryWorkspace, "Close the current workspace", "×", "prefix+shift+d", model.ResolveCloseWorkspace, 0, false, nil),
		resolve("previous_workspace", "Previous workspace", model.CategoryWorkspace, "Focus the previous workspace", "←", "", model.ResolveFocusWorkspace, s1, h1, nil),
		resolve("next_workspace", "Next workspace", model.CategoryWorkspace, "Focus the next workspace", "→", "", model.ResolveFocusWorkspace, s2, h2, nil),

		action("new_tab", "New tab", model.CategoryTabs, "Create and focus a tab", "#", "prefix+c", []string{"tab", "create", "--focus"}),
		resolve("rename_tab", "Rename tab", model.CategoryTabs, "Rename the current tab", "#", "prefix+shift+t", model.ResolveRenameTab, 0, false, name),
		resolve("previous_tab", "Previous tab", model.CategoryTabs, "Focus the previous tab", "←", "prefix+p", model.ResolveFocusTab, s1, h1, nil),
		resolve("next_tab", "Next tab", model.CategoryTabs, "Focus the next tab", "→", "prefix+n", model.ResolveFocusTab, s2, h2, nil),
		resolve("close_tab", "Close tab", model.CategoryTabs, "Close the current tab", "×", "prefix+shift+x", model.ResolveCloseTab, 0, false, nil),

		action("focus_pane_left", "Focus pane left", model.CategoryPanes, "Focus the pane to the left", "←", "prefix+h", []string{"pane", "focus", "--direction", "left", "--current"}),
		action("focus_pane_down", "Focus pane down", model.CategoryPanes, "Focus the pane below", "↓", "prefix+j", []string{"pane", "focus", "--direction", "down", "--current"}),
		action("focus_pane_up", "Focus pane up", model.CategoryPanes, "Focus the pane above", "↑", "prefix+k", []string{"pane", "focus", "--direction", "up", "--current"}),
		action("focus_pane_right", "Focus pane right", model.CategoryPanes, "Focus the pane to the right", "→", "prefix+l", []string{"pane", "focus", "--direction", "right", "--current"}),
		shortcut("cycle_pane_next", "Cycle pane next", model.CategoryPanes, "Focus the next pane in the tab", "→", "prefix+tab"),
		shortcut("cycle_pane_previous", "Cycle pane previous", model.CategoryPanes, "Focus the previous pane in the tab", "←", "prefix+shift+tab"),
		shortcut("last_pane", "Last pane", model.CategoryPanes, "Focus the previously focused pane", "<", ""),
		action("split_vertical", "Split pane right", model.CategoryPanes, "Split the current pane side by side", "|", "prefix+v", []string{"pane", "split", "--current", "--direction", "right", "--focus"}),
		action("split_horizontal", "Split pane down", model.CategoryPanes, "Split the current pane stacked", "-", "prefix+minus", []string{"pane", "split", "--current", "--direction", "down", "--focus"}),
		action("zoom", "Zoom pane", model.CategoryPanes, "Toggle focused pane zoom", "+", "prefix+z", []string{"pane", "zoom", "--current"}),
		resolve("rename_pane", "Rename pane", model.CategoryPanes, "Rename the current pane", "~", "prefix+shift+p", model.ResolveRenamePane, 0, false, name),
		resolve("clear_pane_name", "Clear pane name", model.CategoryPanes, "Remove the current pane's custom name", "~", "", model.ResolveRenamePaneClear, 0, false, nil),
		action("resize_pane_left", "Resize pane left", model.CategoryPanes, "Grow the pane toward the left", "←", "", []string{"pane", "resize", "--direction", "left", "--current"}),
		action("resize_pane_down", "Resize pane down", model.CategoryPanes, "Grow the pane toward the bottom", "↓", "", []string{"pane", "resize", "--direction", "down", "--current"}),
		action("resize_pane_up", "Resize pane up", model.CategoryPanes, "Grow the pane toward the top", "↑", "", []string{"pane", "resize", "--direction", "up", "--current"}),
		action("resize_pane_right", "Resize pane right", model.CategoryPanes, "Grow the pane toward the right", "→", "", []string{"pane", "resize", "--direction", "right", "--current"}),
		action("swap_pane_left", "Swap pane left", model.CategoryPanes, "Swap with the pane to the left", "←", "", []string{"pane", "swap", "--direction", "left", "--current"}),
		action("swap_pane_down", "Swap pane down", model.CategoryPanes, "Swap with the pane below", "↓", "", []string{"pane", "swap", "--direction", "down", "--current"}),
		action("swap_pane_up", "Swap pane up", model.CategoryPanes, "Swap with the pane above", "↑", "", []string{"pane", "swap", "--direction", "up", "--current"}),
		action("swap_pane_right", "Swap pane right", model.CategoryPanes, "Swap with the pane to the right", "→", "", []string{"pane", "swap", "--direction", "right", "--current"}),
		resolve("move_pane_new_tab", "Move pane to new tab", model.CategoryPanes, "Move the current pane into a new tab", "#", "", model.ResolveMovePaneNewTab, 0, false, nil),
		resolve("move_pane_new_workspace", "Move pane to new workspace", model.CategoryPanes, "Move the current pane into a new workspace", "+", "", model.ResolveMovePaneNewWksp, 0, false, nil),
		resolve("close_pane", "Close pane", model.CategoryPanes, "Close the focused pane", "×", "prefix+x", model.ResolveClosePane, 0, false, nil),

		resolve("new_worktree", "New worktree", model.CategoryWorktrees, "Create and open a Git worktree", "*", "prefix+shift+g", model.ResolveWorktreeCreate, 0, false, nil),
		resolve("open_worktree", "Open worktree", model.CategoryWorktrees, "Open an existing Git worktree by branch or path", "*", "", model.ResolveWorktreeOpen, 0, false, branchOrPath),
		resolve("remove_worktree", "Remove worktree", model.CategoryWorktrees, "Remove the current workspace worktree checkout", "×", "", model.ResolveWorktreeRemove, 0, false, confirmRemove),

		resolve("previous_agent", "Previous agent", model.CategoryAgents, "Focus the previous agent", "←", "", model.ResolveFocusAgent, s1, h1, nil),
		resolve("next_agent", "Next agent", model.CategoryAgents, "Focus the next agent", "→", "", model.ResolveFocusAgent, s2, h2, nil),

		shortcut("help", "Keyboard shortcuts", model.CategoryHerdr, "Show Herdr's shortcut guide", "?", "prefix+?"),
		shortcut("settings", "Settings", model.CategoryHerdr, "Open Herdr settings", "=", "prefix+s"),
		shortcut("copy_mode", "Copy mode", model.CategoryHerdr, "Enter copy mode", "%", "prefix+["),
		shortcut("detach", "Detach", model.CategoryHerdr, "Leave the current Herdr session", ">", "prefix+q"),
	}
}
