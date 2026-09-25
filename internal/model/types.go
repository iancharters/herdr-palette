package model

type Category string

const (
	CategoryWorkspace Category = "Workspace"
	CategoryTabs      Category = "Tabs"
	CategoryPanes     Category = "Panes"
	CategoryWorktrees Category = "Worktrees"
	CategoryAgents    Category = "Agents"
	CategoryHerdr     Category = "Herdr"
	CategoryCustom    Category = "Custom"
)

type ResolveAction string

const (
	ResolveClosePane         ResolveAction = "close-pane"
	ResolveCloseTab          ResolveAction = "close-tab"
	ResolveCloseWorkspace    ResolveAction = "close-workspace"
	ResolveFocusTab          ResolveAction = "focus-tab"
	ResolveFocusWorkspace    ResolveAction = "focus-workspace"
	ResolveFocusAgent        ResolveAction = "focus-agent"
	ResolveRenamePane        ResolveAction = "rename-pane"
	ResolveRenamePaneClear   ResolveAction = "rename-pane-clear"
	ResolveRenameTab         ResolveAction = "rename-tab"
	ResolveRenameWorkspace   ResolveAction = "rename-workspace"
	ResolveMovePaneNewTab    ResolveAction = "move-pane-new-tab"
	ResolveMovePaneNewWksp   ResolveAction = "move-pane-new-workspace"
	ResolveWorktreeCreate    ResolveAction = "worktree-create"
	ResolveWorktreeOpen      ResolveAction = "worktree-open"
	ResolveWorktreeRemove    ResolveAction = "worktree-remove"
)

type InvocationKind string

const (
	InvocationHerdr    InvocationKind = "herdr"
	InvocationResolve  InvocationKind = "resolve"
	InvocationShortcut InvocationKind = "shortcut"
)

// Invocation mirrors the TS union: exactly one variant is populated.
type Invocation struct {
	Kind   InvocationKind
	Argv   []string      // herdr kind
	Action ResolveAction // resolve kind
	Step   int           // resolve kind, -1 | 0 | 1 (0 = unset)
	HasStep bool
}

type PromptSpec struct {
	Placeholder string
	AllowEmpty  bool
}

type SessionTarget struct {
	PaneID      string
	TabID       string
	WorkspaceID string
}

type PaletteItem struct {
	ID         string
	Title      string
	Category   Category
	Group      string // second-level heading inside a category (plugin_id for discovered actions)
	Description string
	Icon       string
	Aliases    []string
	Shortcuts  []string
	Invocation Invocation
	Prompt     *PromptSpec
}

type CommandResult struct {
	OK      bool
	Message string
	// Output is captured stdout worth showing (e.g. plugin action results).
	// Non-empty Output with OK means "keep the palette open on a result view".
	Output string
	Title  string
}
