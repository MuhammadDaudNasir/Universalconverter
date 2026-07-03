package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainModel struct {
	screens      []SubModel
	activeTab    int
	focusSidebar bool
	width        int
	height       int
	sidebarItems []string
}

func NewMainModel(initialPaths []string) *MainModel {
	cs := NewCompressScreen()
	// If file paths were passed via CLI args (drag-and-drop or command line),
	// pre-populate the compression source field and auto-suggest destination.
	if len(initialPaths) > 0 {
		cs.srcInput.SetValue(initialPaths[0])
		cs.updateSuggestedDst()
	}

	m := &MainModel{
		screens: []SubModel{
			cs,
			NewEncryptScreen(),
			NewDataScreen(),
			NewMediaScreen(),
			NewHashScreen(),
			NewUnitScreen(),
		},
		activeTab:    0,
		focusSidebar: true,
		sidebarItems: []string{
			"📦 Compression",
			"🔒 Encryption",
			"📊 Data Formats",
			"🎬 Media Convert",
			"🔑 Text & Hash",
			"📐 Unit Converter",
		},
	}
	m.updateActiveScreenFocus()
	return m
}

func (m *MainModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, scr := range m.screens {
		cmds = append(cmds, scr.Init())
	}
	return tea.Batch(cmds...)
}

func (m *MainModel) updateActiveScreenFocus() {
	for i, scr := range m.screens {
		scr.SetFocus(i == m.activeTab && !m.focusSidebar)
	}
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate size to all sub-screens if needed, but since View takes width/height, we handle it there.
		return m, nil

	case tea.KeyMsg:
		if m.focusSidebar {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "up", "k":
				m.activeTab = (m.activeTab - 1 + len(m.screens)) % len(m.screens)
				m.updateActiveScreenFocus()
				return m, nil
			case "down", "j":
				m.activeTab = (m.activeTab + 1) % len(m.screens)
				m.updateActiveScreenFocus()
				return m, nil
			case "right", "tab", "enter":
				m.focusSidebar = false
				m.updateActiveScreenFocus()
				return m, nil
			}
		} else {
			// Inside a workspace content pane
			if msg.String() == "esc" {
				m.focusSidebar = true
				m.updateActiveScreenFocus()
				return m, nil
			}
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

			// Delegate key msg to the active screen
			var subCmd tea.Cmd
			m.screens[m.activeTab], subCmd = m.screens[m.activeTab].Update(msg)
			return m, subCmd
		}
	}

	// Propagate background events (ProgressMsg, LogMsg, taskFinishedMsg) to ALL sub-screens
	var cmds []tea.Cmd
	for i := range m.screens {
		var subCmd tea.Cmd
		m.screens[i], subCmd = m.screens[i].Update(msg)
		if subCmd != nil {
			cmds = append(cmds, subCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *MainModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing dashboard layout..."
	}

	// Calculate dimensions
	sidebarWidth := 25
	contentWidth := m.width - sidebarWidth - 4
	if contentWidth < 40 {
		contentWidth = 40 // Minimum fallback
	}

	contentHeight := m.height - 6 // Reservation for borders and footer

	// Render Sidebar
	var sbLines []string
	sbLines = append(sbLines, SidebarTitle.Render("✨ UNICONVERT"))
	sbLines = append(sbLines, lipgloss.NewStyle().Foreground(TextMutedColor).Render("────────────────────"))

	for i, item := range m.sidebarItems {
		var line string
		if i == m.activeTab {
			if m.focusSidebar {
				// Focused on sidebar tab
				line = SidebarItemActive.Render(item)
			} else {
				// Focused on content, tab is selected but muted
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(lipgloss.Color("#2C2D30")).
					PaddingLeft(1).
					Render(item)
			}
		} else {
			line = SidebarItem.Render(item)
		}
		sbLines = append(sbLines, "\n"+line)
	}

	// Render Content
	activeScreenView := m.screens[m.activeTab].View(contentWidth, contentHeight)

	// Sidebar container styling
	sidebarRender := Sidebar.
		Height(contentHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, sbLines...))

	// Content container styling
	contentBorderCol := BorderMutedCol
	if !m.focusSidebar {
		contentBorderCol = BorderActiveCol
	}

	contentContainer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(contentBorderCol).
		Width(contentWidth).
		Height(contentHeight).
		Padding(1, 2)

	contentRender := contentContainer.Render(activeScreenView)

	// Combine horizontally
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, sidebarRender, contentRender)

	// Render Footer Help Guide
	footerHelp := m.renderFooterHelp()

	// Combine all vertically
	fullLayout := lipgloss.JoinVertical(
		lipgloss.Left,
		mainBody,
		FooterStyle.Width(m.width-2).Render(footerHelp),
	)

	return MainContainer.
		Width(m.width).
		Height(m.height).
		Render(fullLayout)
}

func (m *MainModel) renderFooterHelp() string {
	var elements []string

	if m.focusSidebar {
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("↑/↓ / j/k"), "Navigate Tabs"))
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("Tab/Enter/→"), "Enter Workspace"))
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("q"), "Quit App"))
	} else {
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("Esc"), "Return to Sidebar"))
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("Tab / Shift+Tab"), "Next/Prev Field"))
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("↑/↓ / ←/→"), "Modify Form Elements"))
		elements = append(elements, fmt.Sprintf("%s %s", FooterHelpKey.Render("Enter"), "Submit / Run Task"))
	}

	return strings.Join(elements, "   •   ")
}
