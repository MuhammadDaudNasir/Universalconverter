package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"universal-converter/pkg/data"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DataScreenMode int

const (
	DataModeInteractive DataScreenMode = 0
	DataModeFile        DataScreenMode = 1
)

type DataScreen struct {
	focused      bool
	focusIndex   int
	mode         DataScreenMode

	// Common selectors
	formats      []data.Format
	fromFormat   int
	toFormat     int

	// Interactive Mode fields
	inputArea    textarea.Model
	outputView   viewport.Model
	outputStr    string

	// File Mode fields
	srcInput     textinput.Model
	dstInput     textinput.Model

	// Status info
	statusMsg    string
	statusErr    bool
}

func NewDataScreen() *DataScreen {
	ta := textarea.New()
	ta.Placeholder = "Paste your data format here (e.g. JSON, YAML...)"
	ta.SetWidth(50)
	ta.SetHeight(8)
	ta.FocusedStyle.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(PrimaryColor)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderMutedCol)

	vp := viewport.New(60, 10)
	vp.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderMutedCol)
	vp.SetContent("Converted output will appear here...")

	src := textinput.New()
	src.Placeholder = "/path/to/input.json"
	src.Width = 50

	dst := textinput.New()
	dst.Placeholder = "/path/to/output.yaml"
	dst.Width = 50

	return &DataScreen{
		mode:        DataModeInteractive,
		formats:     []data.Format{data.FormatJSON, data.FormatYAML, data.FormatTOML, data.FormatXML, data.FormatCSV},
		fromFormat:  0,
		toFormat:    1,
		inputArea:   ta,
		outputView:  vp,
		srcInput:    src,
		dstInput:    dst,
	}
}

func (s *DataScreen) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, textinput.Blink)
}

func (s *DataScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.inputArea.Blur()
		s.srcInput.Blur()
		s.dstInput.Blur()
	} else {
		s.refocus()
	}
}

func (s *DataScreen) refocus() {
	s.inputArea.Blur()
	s.srcInput.Blur()
	s.dstInput.Blur()

	if s.mode == DataModeInteractive {
		if s.focusIndex == 1 {
			s.inputArea.Focus()
		}
	} else {
		if s.focusIndex == 1 {
			s.srcInput.Focus()
		} else if s.focusIndex == 2 {
			s.dstInput.Focus()
		}
	}
}

func (s *DataScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Viewport scrolling
			if s.mode == DataModeInteractive && s.focusIndex == 4 {
				var cmd tea.Cmd
				s.outputView, cmd = s.outputView.Update(msg)
				return s, cmd
			}

			switch msg.String() {
			case "tab":
				limit := 6 // Interactive: Mode(0), InputArea(1), From(2), To(3), Output(4), Button(5)
				if s.mode == DataModeFile {
					limit = 6 // File: Mode(0), Src(1), Dst(2), From(3), To(4), Button(5)
				}
				s.focusIndex = (s.focusIndex + 1) % limit
				s.refocus()
				return s, nil
			case "shift+tab":
				limit := 6
				if s.mode == DataModeFile {
					limit = 6
				}
				s.focusIndex = (s.focusIndex - 1 + limit) % limit
				s.refocus()
				return s, nil
			case "up":
				if s.focusIndex > 0 {
					s.focusIndex--
					s.refocus()
				}
				return s, nil
			case "down":
				limit := 6
				if s.focusIndex < limit-1 {
					s.focusIndex++
					s.refocus()
				}
				return s, nil

			case "left", "right":
				// Mode Toggle
				if s.focusIndex == 0 {
					if s.mode == DataModeInteractive {
						s.mode = DataModeFile
					} else {
						s.mode = DataModeInteractive
					}
					s.statusMsg = ""
					s.focusIndex = 0
					s.refocus()
					return s, nil
				}
				// Format From Selection
				if (s.mode == DataModeInteractive && s.focusIndex == 2) || (s.mode == DataModeFile && s.focusIndex == 3) {
					idx := s.fromFormat
					if msg.String() == "left" {
						idx = (idx - 1 + len(s.formats)) % len(s.formats)
					} else {
						idx = (idx + 1) % len(s.formats)
					}
					s.fromFormat = idx
					s.updateSuggestedDst()
					return s, nil
				}
				// Format To Selection
				if (s.mode == DataModeInteractive && s.focusIndex == 3) || (s.mode == DataModeFile && s.focusIndex == 4) {
					idx := s.toFormat
					if msg.String() == "left" {
						idx = (idx - 1 + len(s.formats)) % len(s.formats)
					} else {
						idx = (idx + 1) % len(s.formats)
					}
					s.toFormat = idx
					s.updateSuggestedDst()
					return s, nil
				}

			case "enter":
				// Check if action button is clicked
				btnIdx := 5
				if s.focusIndex == btnIdx {
					s.runConversion()
					return s, nil
				}

				// Advance fields
				if s.mode == DataModeInteractive {
					if s.focusIndex == 1 {
						s.focusIndex = 2
						s.refocus()
						return s, nil
					}
				} else {
					if s.focusIndex == 1 {
						s.focusIndex = 2
						s.refocus()
						s.updateSuggestedDst()
						return s, nil
					} else if s.focusIndex == 2 {
						s.focusIndex = 3
						s.refocus()
						return s, nil
					}
				}
			}
		}
	}

	// Update text inputs or textareas depending on focus
	if s.mode == DataModeInteractive {
		if s.focusIndex == 1 {
			var cmd tea.Cmd
			s.inputArea, cmd = s.inputArea.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		if s.focusIndex == 1 {
			var cmd tea.Cmd
			s.srcInput, cmd = s.srcInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if s.focusIndex == 2 {
			var cmd tea.Cmd
			s.dstInput, cmd = s.dstInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *DataScreen) updateSuggestedDst() {
	src := CleanPath(s.srcInput.Value())
	s.srcInput.SetValue(src)
	if src == "" {
		return
	}
	ext := "." + string(s.formats[s.toFormat])
	s.dstInput.SetValue(strings.TrimSuffix(src, filepath.Ext(src)) + ext)
}

func (s *DataScreen) runConversion() {
	s.statusMsg = ""
	s.statusErr = false

	if s.mode == DataModeInteractive {
		text := s.inputArea.Value()
		if strings.TrimSpace(text) == "" {
			s.statusMsg = "Error: Input data area is empty"
			s.statusErr = true
			return
		}

		out, err := data.Convert(text, s.formats[s.fromFormat], s.formats[s.toFormat])
		if err != nil {
			s.statusMsg = fmt.Sprintf("Conversion failed: %v", err)
			s.statusErr = true
			s.outputView.SetContent("Error rendering conversion output.")
		} else {
			s.outputStr = out
			s.outputView.SetContent(out)
			s.statusMsg = "Conversion successful! Use keyboard focus on viewport to scroll."
		}
	} else {
		src := CleanPath(s.srcInput.Value())
		dst := CleanPath(s.dstInput.Value())
		s.srcInput.SetValue(src)
		s.dstInput.SetValue(dst)

		if src == "" || dst == "" {
			s.statusMsg = "Error: File paths cannot be empty"
			s.statusErr = true
			return
		}

		// Read file
		content, err := os.ReadFile(src)
		if err != nil {
			s.statusMsg = fmt.Sprintf("Failed to read source file: %v", err)
			s.statusErr = true
			return
		}

		out, err := data.Convert(string(content), s.formats[s.fromFormat], s.formats[s.toFormat])
		if err != nil {
			s.statusMsg = fmt.Sprintf("Conversion failed: %v", err)
			s.statusErr = true
			return
		}

		// Write file
		err = os.WriteFile(dst, []byte(out), 0644)
		if err != nil {
			s.statusMsg = fmt.Sprintf("Failed to write output file: %v", err)
			s.statusErr = true
			return
		}

		s.statusMsg = fmt.Sprintf("Successfully converted and saved to %s", filepath.Base(dst))
	}
}

func (s *DataScreen) View(width int, height int) string {
	var sections []string

	sections = append(sections, TitleStyle.Render("Data Format Converter"))
	sections = append(sections, SubtitleStyle.Render("Convert hierarchical formats JSON, YAML, TOML, XML, and tabular CSV files."))

	var fields []string

	// 1. Mode Select
	modeLabel := FieldLabel.Render("Conversion Mode: ")
	modeVal := " [Interactive Text] "
	if s.mode == DataModeFile {
		modeVal = " [File to File] "
	}
	if s.focused && s.focusIndex == 0 {
		modeVal = HighlightActive.Render(modeVal)
	} else {
		modeVal = TextPrimary.Render(modeVal)
	}
	fields = append(fields, fmt.Sprintf("%s%s\n", modeLabel, modeVal))

	if s.mode == DataModeInteractive {
		// Interactive Input Area
		taLabel := FieldLabel.Render("Source Text:")
		taBox := s.inputArea.View()
		fields = append(fields, fmt.Sprintf("%s\n%s\n", taLabel, taBox))

		// Formats
		fmtRow := s.renderFormatSelectors(2, 3)
		fields = append(fields, fmtRow+"\n")

		// Viewport Output
		vpLabel := FieldLabel.Render("Converted Output:")
		vpBox := s.outputView.View()
		if s.focused && s.focusIndex == 4 {
			s.outputView.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(PrimaryColor)
			vpBox = s.outputView.View()
		} else {
			s.outputView.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderMutedCol)
			vpBox = s.outputView.View()
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", vpLabel, vpBox))

	} else {
		// File Mode Inputs
		srcLabel := FieldLabel.Render("Source Data File:")
		srcBox := s.srcInput.View()
		if s.focused && s.focusIndex == 1 {
			srcBox = InputFocused.Render(srcBox)
		} else {
			srcBox = InputUnfocused.Render(srcBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", srcLabel, srcBox))

		dstLabel := FieldLabel.Render("Destination Data File:")
		dstBox := s.dstInput.View()
		if s.focused && s.focusIndex == 2 {
			dstBox = InputFocused.Render(dstBox)
		} else {
			dstBox = InputUnfocused.Render(dstBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", dstLabel, dstBox))

		// Formats
		fmtRow := s.renderFormatSelectors(3, 4)
		fields = append(fields, fmtRow+"\n")
	}

	// Action Button
	btnText := "  Convert Structure  "
	var btn string
	if s.focused && s.focusIndex == 5 {
		btn = BtnFocused.Render(btnText)
	} else {
		btn = BtnUnfocused.Render(btnText)
	}
	fields = append(fields, btn)

	sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))

	// Status Info
	if s.statusMsg != "" {
		if s.statusErr {
			sections = append(sections, StatusError.Render("✖ "+s.statusMsg))
		} else {
			sections = append(sections, StatusSuccess.Render("✔ "+s.statusMsg))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (s *DataScreen) renderFormatSelectors(fromIdx, toIdx int) string {
	fromLabel := FieldLabel.Render("From: ")
	var fromStr string
	for i, f := range s.formats {
		optStr := fmt.Sprintf(" %s ", strings.ToUpper(string(f)))
		if i == s.fromFormat {
			if s.focused && s.focusIndex == fromIdx {
				fromStr += HighlightActive.Render(optStr) + " "
			} else {
				fromStr += TextPrimary.Render(optStr) + " "
			}
		} else {
			fromStr += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
		}
	}

	toLabel := FieldLabel.Render("  To: ")
	var toStr string
	for i, f := range s.formats {
		optStr := fmt.Sprintf(" %s ", strings.ToUpper(string(f)))
		if i == s.toFormat {
			if s.focused && s.focusIndex == toIdx {
				toStr += HighlightActive.Render(optStr) + " "
			} else {
				toStr += TextPrimary.Render(optStr) + " "
			}
		} else {
			toStr += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
		}
	}

	return fmt.Sprintf("%s%s%s%s", fromLabel, fromStr, toLabel, toStr)
}
