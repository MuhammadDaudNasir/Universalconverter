package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// SubModel represents a distinct workspace screen in the TUI application.
type SubModel interface {
	Init() tea.Cmd
	Update(tea.Msg) (SubModel, tea.Cmd)
	View(width int, height int) string
	SetFocus(focused bool)
}

// Common messages used for asynchronous updates
type ProgressMsg float64

type LogMsg string

type StatusMsg struct {
	IsError bool
	Message string
}

type CompletedMsg struct {
	Stats map[string]string
}
