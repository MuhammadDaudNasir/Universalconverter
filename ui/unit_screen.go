package ui

import (
	"fmt"
	"strconv"
	"universal-converter/pkg/unit"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type UnitScreen struct {
	focused      bool
	focusIndex   int // 0 = category, 1 = input value, 2 = from unit, 3 = to unit

	// Form fields
	valInput     textinput.Model
	categories   []unit.Category
	catIndex     int
	fromUnitIdx  int
	toUnitIdx    int

	// Calculated Result
	resultStr    string
	hasError     bool
}

func NewUnitScreen() *UnitScreen {
	val := textinput.New()
	val.Placeholder = "Enter numeric value (e.g. 100)"
	val.Width = 50
	val.SetValue("1.0") // Default value

	cats := []unit.Category{
		unit.CatLength,
		unit.CatWeight,
		unit.CatTemperature,
		unit.CatDataSize,
		unit.CatArea,
		unit.CatSpeed,
	}

	us := &UnitScreen{
		valInput:    val,
		categories:  cats,
		catIndex:    0,
		fromUnitIdx: 2, // Meter (default in Length: mm, cm, m...)
		toUnitIdx:   4, // Inch
	}
	us.calculateConversion()
	return us
}

func (s *UnitScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *UnitScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.valInput.Blur()
	} else {
		s.refocus()
	}
}

func (s *UnitScreen) refocus() {
	s.valInput.Blur()
	if s.focusIndex == 1 {
		s.valInput.Focus()
	}
}

func (s *UnitScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				s.focusIndex = (s.focusIndex + 1) % 4
				s.refocus()
				return s, nil
			case "shift+tab":
				s.focusIndex = (s.focusIndex - 1 + 4) % 4
				s.refocus()
				return s, nil
			case "up":
				if s.focusIndex > 0 {
					s.focusIndex--
					s.refocus()
				}
				return s, nil
			case "down":
				if s.focusIndex < 3 {
					s.focusIndex++
					s.refocus()
				}
				return s, nil

			case "left", "right":
				// Category Select
				if s.focusIndex == 0 {
					if msg.String() == "left" {
						s.catIndex = (s.catIndex - 1 + len(s.categories)) % len(s.categories)
					} else {
						s.catIndex = (s.catIndex + 1) % len(s.categories)
					}
					// Reset from/to indexes when changing category to avoid out of bounds
					s.fromUnitIdx = 0
					s.toUnitIdx = 1
					if s.toUnitIdx >= len(unit.Categories[s.categories[s.catIndex]]) {
						s.toUnitIdx = 0
					}
					s.calculateConversion()
					return s, nil
				}

				// From Unit Select
				if s.focusIndex == 2 {
					units := unit.Categories[s.categories[s.catIndex]]
					if msg.String() == "left" {
						s.fromUnitIdx = (s.fromUnitIdx - 1 + len(units)) % len(units)
					} else {
						s.fromUnitIdx = (s.fromUnitIdx + 1) % len(units)
					}
					s.calculateConversion()
					return s, nil
				}

				// To Unit Select
				if s.focusIndex == 3 {
					units := unit.Categories[s.categories[s.catIndex]]
					if msg.String() == "left" {
						s.toUnitIdx = (s.toUnitIdx - 1 + len(units)) % len(units)
					} else {
						s.toUnitIdx = (s.toUnitIdx + 1) % len(units)
					}
					s.calculateConversion()
					return s, nil
				}
			}
		}
	}

	// Capture text updates in the input box
	prevVal := s.valInput.Value()
	var cmd tea.Cmd
	s.valInput, cmd = s.valInput.Update(msg)
	cmds = append(cmds, cmd)

	if s.valInput.Value() != prevVal {
		s.calculateConversion()
	}

	return s, tea.Batch(cmds...)
}

func (s *UnitScreen) calculateConversion() {
	rawVal := s.valInput.Value()
	if rawVal == "" {
		s.resultStr = "Please enter a value"
		s.hasError = true
		return
	}

	val, err := strconv.ParseFloat(rawVal, 64)
	if err != nil {
		s.resultStr = "Invalid number format"
		s.hasError = true
		return
	}

	cat := s.categories[s.catIndex]
	units := unit.Categories[cat]

	if s.fromUnitIdx >= len(units) || s.toUnitIdx >= len(units) {
		s.resultStr = "Invalid unit configuration"
		s.hasError = true
		return
	}

	fromUnit := units[s.fromUnitIdx]
	toUnit := units[s.toUnitIdx]

	res, err := unit.Convert(val, cat, fromUnit.Name, toUnit.Name)
	if err != nil {
		s.resultStr = fmt.Sprintf("Error: %v", err)
		s.hasError = true
		return
	}

	s.hasError = false
	s.resultStr = fmt.Sprintf(
		"%s %s   =   %s %s",
		unit.FormatResult(val),
		fromUnit.Symbol,
		TextSecondary.Bold(true).Render(unit.FormatResult(res)),
		TextPrimary.Bold(true).Render(toUnit.Symbol),
	)
}

func (s *UnitScreen) View(width int, height int) string {
	var sections []string

	sections = append(sections, TitleStyle.Render("Unit Conversion Grid"))
	sections = append(sections, SubtitleStyle.Render("Fast linear and physical unit mapping across physical quantities and scales."))

	var fields []string

	// 1. Category Selection
	catLabel := FieldLabel.Render("Quantity Category:")
	var catOptions string
	for i, catOpt := range s.categories {
		optStr := fmt.Sprintf(" %s ", catOpt)
		if i == s.catIndex {
			if s.focused && s.focusIndex == 0 {
				catOptions += HighlightActive.Render(optStr) + " "
			} else {
				catOptions += TextPrimary.Render(optStr) + " "
			}
		} else {
			catOptions += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
		}
	}
	fields = append(fields, fmt.Sprintf("%s\n%s\n", catLabel, catOptions))

	// 2. Input Value
	valLabel := FieldLabel.Render("Source Value:")
	valBox := s.valInput.View()
	if s.focused && s.focusIndex == 1 {
		valBox = InputFocused.Render(valBox)
	} else {
		valBox = InputUnfocused.Render(valBox)
	}
	fields = append(fields, fmt.Sprintf("%s\n%s\n", valLabel, valBox))

	// 3. Units lists
	units := unit.Categories[s.categories[s.catIndex]]

	// From unit select
	fromLabel := FieldLabel.Render("From Unit:")
	var fromOptions string
	for i, u := range units {
		optStr := fmt.Sprintf(" %s (%s) ", u.Name, u.Symbol)
		if i == s.fromUnitIdx {
			if s.focused && s.focusIndex == 2 {
				fromOptions += HighlightActive.Render(optStr) + " "
			} else {
				fromOptions += TextPrimary.Render(optStr) + " "
			}
		} else {
			fromOptions += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
		}
	}
	fields = append(fields, fmt.Sprintf("%s\n%s\n", fromLabel, fromOptions))

	// To unit select
	toLabel := FieldLabel.Render("To Unit:")
	var toOptions string
	for i, u := range units {
		optStr := fmt.Sprintf(" %s (%s) ", u.Name, u.Symbol)
		if i == s.toUnitIdx {
			if s.focused && s.focusIndex == 3 {
				toOptions += HighlightActive.Render(optStr) + " "
			} else {
				toOptions += TextPrimary.Render(optStr) + " "
			}
		} else {
			toOptions += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
		}
	}
	fields = append(fields, fmt.Sprintf("%s\n%s\n", toLabel, toOptions))

	sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))

	// Result Panel
	var resultBox string
	if s.hasError {
		resultBox = StatusError.Render("✖ " + s.resultStr)
	} else {
		resultBox = fmt.Sprintf("\n  %s\n", s.resultStr)
	}

	sections = append(sections, lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(BorderActiveCol).
		Padding(1, 4).
		MarginBottom(1).
		Render(resultBox))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
