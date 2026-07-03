package main

import (
	"fmt"
	"os"
	"universal-converter/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Pass any CLI arguments (e.g. dragged file paths) as initial paths
	initialPaths := os.Args[1:]

	// Clean all incoming paths (handles macOS drag-and-drop quoting/escaping)
	for i, p := range initialPaths {
		initialPaths[i] = ui.CleanPath(p)
	}

	// Initialize the main model coordinator with pre-populated paths
	m := ui.NewMainModel(initialPaths)

	// Launch in alternate screen buffer for full-screen dashboard experience
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting TUI: %v\n", err)
		os.Exit(1)
	}
}
