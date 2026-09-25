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
	items := config.LoadPaletteItems("")
	overrides := theme.LoadTheme("", nil)
	styles := theme.StylesFor(overrides)
	m := ui.New(items, styles, func(it model.PaletteItem, input string) model.CommandResult {
		return execute.Execute(it, input)
	})
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-palette: "+err.Error())
		os.Exit(1)
	}
}
