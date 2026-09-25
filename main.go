package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iancharters/herdr-palette/internal/config"
	"github.com/iancharters/herdr-palette/internal/execute"
	"github.com/iancharters/herdr-palette/internal/model"
	"github.com/iancharters/herdr-palette/internal/theme"
	"github.com/iancharters/herdr-palette/internal/ui"
)

func main() {
	// Blocking single-action runner for terminal-visible execution:
	// `herdr-palette invoke <plugin_id>.<action_id>` streams the action's
	// output to this pane. The palette itself launches it via `pane run`.
	if len(os.Args) == 3 && os.Args[1] == "invoke" {
		out, msg, code := execute.Invoke(os.Args[2])
		if out != "" {
			fmt.Println(out)
		}
		if msg != "" {
			fmt.Fprintln(os.Stderr, "herdr-palette: "+msg)
		}
		os.Exit(code)
	}
	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "usage: herdr-palette [invoke <plugin_id>.<action_id>]")
		os.Exit(2)
	}
	items := config.LoadPaletteItems("")
	overrides := theme.LoadTheme("", nil)
	styles := theme.StylesFor(overrides)
	m := ui.New(items, styles, func(it model.PaletteItem, input string) model.CommandResult {
		return execute.ExecuteInTerminal(it, input)
	})
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-palette: "+err.Error())
		os.Exit(1)
	}
}
