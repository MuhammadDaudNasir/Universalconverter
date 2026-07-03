package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// ── Blue Theme Color Palette ──────────────────────────────────────────
	BgColor        = lipgloss.Color("#0F172A") // Slate 900
	SidebarBgColor = lipgloss.Color("#1E293B") // Slate 800
	PrimaryColor   = lipgloss.Color("#3B82F6") // Blue 500
	SecondaryColor = lipgloss.Color("#60A5FA") // Blue 400
	AccentColor    = lipgloss.Color("#2563EB") // Blue 600
	AccentSuccess  = lipgloss.Color("#22C55E") // Green 500
	AccentError    = lipgloss.Color("#EF4444") // Red 500
	AccentWarning  = lipgloss.Color("#F59E0B") // Amber 500
	TextBaseColor  = lipgloss.Color("#E2E8F0") // Slate 200
	TextMutedColor = lipgloss.Color("#94A3B8") // Slate 400
	TextDimColor   = lipgloss.Color("#64748B") // Slate 500
	BorderActiveCol = lipgloss.Color("#3B82F6")
	BorderMutedCol  = lipgloss.Color("#334155") // Slate 700

	// ── Base Layout ──────────────────────────────────────────────────────
	MainContainer = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(TextBaseColor)

	Sidebar = lipgloss.NewStyle().
		Background(SidebarBgColor).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(BorderMutedCol).
		Padding(1, 1)

	ContentArea = lipgloss.NewStyle().
			Padding(1, 2)

	// ── Sidebar Items ────────────────────────────────────────────────────
	SidebarTitle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	SidebarItem = lipgloss.NewStyle().
			Foreground(TextDimColor).
			PaddingLeft(2)

	SidebarItemActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(PrimaryColor).
				Bold(true).
				PaddingLeft(2)

	SidebarItemSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#1E3A5F")).
				PaddingLeft(2)

	// ── Content UI Elements ──────────────────────────────────────────────
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(TextMutedColor).
			Italic(true).
			MarginBottom(1)

	SectionHeader = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true).
			MarginBottom(1)

	// ── Fields and Inputs ────────────────────────────────────────────────
	FieldLabel = lipgloss.NewStyle().
			Foreground(TextBaseColor).
			Bold(true)

	InputFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 1)

	InputUnfocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderMutedCol).
			Padding(0, 1)

	// ── Buttons ──────────────────────────────────────────────────────────
	BtnFocused = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(PrimaryColor).
			Bold(true).
			Padding(0, 3).
			MarginTop(1)

	BtnUnfocused = lipgloss.NewStyle().
			Foreground(TextBaseColor).
			Background(lipgloss.Color("#334155")).
			Padding(0, 3).
			MarginTop(1)

	// ── Status Messages ──────────────────────────────────────────────────
	StatusSuccess = lipgloss.NewStyle().
			Foreground(AccentSuccess).
			Bold(true).
			MarginTop(1)

	StatusError = lipgloss.NewStyle().
			Foreground(AccentError).
			Bold(true).
			MarginTop(1)

	StatusPending = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true).
			MarginTop(1)

	StatusWarning = lipgloss.NewStyle().
			Foreground(AccentWarning).
			Bold(true).
			MarginTop(1)

	// ── Box / Border Wrappers ────────────────────────────────────────────
	BoxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderMutedCol).
			Padding(1, 2).
			MarginBottom(1)

	// ── Footer ───────────────────────────────────────────────────────────
	FooterStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder(), true, false, false, false).
			BorderForeground(BorderMutedCol).
			Foreground(TextMutedColor).
			PaddingTop(1).
			MarginTop(1)

	FooterHelpKey = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	// ── Color Style Helpers ──────────────────────────────────────────────
	TextPrimary   = lipgloss.NewStyle().Foreground(PrimaryColor)
	TextSecondary = lipgloss.NewStyle().Foreground(SecondaryColor)
	TextSuccess   = lipgloss.NewStyle().Foreground(AccentSuccess)
	TextError     = lipgloss.NewStyle().Foreground(AccentError)
	TextMuted     = lipgloss.NewStyle().Foreground(TextMutedColor)

	HighlightActive  = lipgloss.NewStyle().Background(PrimaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	HighlightPrimary = lipgloss.NewStyle().Background(AccentColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	// ── Post-Action Menu ─────────────────────────────────────────────────
	ActionItem = lipgloss.NewStyle().
			Foreground(TextBaseColor).
			PaddingLeft(2)

	ActionItemActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(PrimaryColor).
				Bold(true).
				PaddingLeft(2)
)

// CleanPath normalizes paths pasted or dragged-and-dropped into TUI fields.
func CleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") && len(p) >= 2 {
		p = p[1 : len(p)-1]
	}
	if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") && len(p) >= 2 {
		p = p[1 : len(p)-1]
	}
	p = strings.ReplaceAll(p, "\\ ", " ")
	return strings.TrimSpace(p)
}

// Responsive helpers

// CompactLabel truncates a label to fit within maxWidth.
func CompactLabel(label string, maxWidth int) string {
	if len(label) <= maxWidth {
		return label
	}
	if maxWidth <= 3 {
		return label[:maxWidth]
	}
	return label[:maxWidth-2] + ".."
}

// IsCompact returns true when the terminal is too narrow for the full layout.
func IsCompact(width int) bool {
	return width < 100
}

// IsUltraCompact returns true when the terminal is very small.
func IsUltraCompact(width int) bool {
	return width < 70
}
