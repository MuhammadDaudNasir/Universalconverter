package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"universal-converter/pkg/hash"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HashInputMode int

const (
	HashInputText HashInputMode = 0
	HashInputFile HashInputMode = 1
)

type hashResultRow struct {
	label string
	value string
}

type HashScreen struct {
	focused     bool
	focusIndex  int // 0 = mode select, 1 = input box, 2 = output grid
	inputMode   HashInputMode

	// Form fields
	textIn      textinput.Model
	fileIn      textinput.Model

	// Results grid
	rows        []hashResultRow
	selectedRow int

	// Status messages
	statusMsg   string
	statusErr   bool
}

func NewHashScreen() *HashScreen {
	txt := textinput.New()
	txt.Placeholder = "Type plain text to hash / encode..."
	txt.Width = 50

	fil := textinput.New()
	fil.Placeholder = "/path/to/file/to/hash"
	fil.Width = 50

	return &HashScreen{
		inputMode:   HashInputText,
		textIn:      txt,
		fileIn:      fil,
		selectedRow: 0,
	}
}

func (s *HashScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *HashScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.textIn.Blur()
		s.fileIn.Blur()
	} else {
		s.refocus()
	}
}

func (s *HashScreen) refocus() {
	s.textIn.Blur()
	s.fileIn.Blur()

	if s.focusIndex == 1 {
		if s.inputMode == HashInputText {
			s.textIn.Focus()
		} else {
			s.fileIn.Focus()
		}
	}
}

func (s *HashScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Grid Navigation (FocusIndex = 2)
			if s.focusIndex == 2 && len(s.rows) > 0 {
				switch msg.String() {
				case "up":
					s.selectedRow = (s.selectedRow - 1 + len(s.rows)) % len(s.rows)
					return s, nil
				case "down":
					s.selectedRow = (s.selectedRow + 1) % len(s.rows)
					return s, nil
				case "enter", "y", "c":
					s.copySelectedToClipboard()
					return s, nil
				}
			}

			switch msg.String() {
			case "tab":
				s.focusIndex = (s.focusIndex + 1) % 3
				s.refocus()
				return s, nil
			case "shift+tab":
				s.focusIndex = (s.focusIndex - 1 + 3) % 3
				s.refocus()
				return s, nil
			case "up":
				if s.focusIndex > 0 {
					s.focusIndex--
					s.refocus()
				}
				return s, nil
			case "down":
				if s.focusIndex < 2 {
					s.focusIndex++
					s.refocus()
				}
				return s, nil

			case "left", "right", " ":
				if s.focusIndex == 0 {
					if s.inputMode == HashInputText {
						s.inputMode = HashInputFile
					} else {
						s.inputMode = HashInputText
					}
					s.statusMsg = ""
					s.rows = nil
					s.refocus()
					return s, nil
				}

			case "enter":
				if s.focusIndex == 1 {
					s.computeOutputs()
					s.focusIndex = 2
					s.selectedRow = 0
					s.refocus()
					return s, nil
				}
			}
		}
	}

	// Live update outputs for Text Mode
	var prevVal string
	if s.inputMode == HashInputText {
		prevVal = s.textIn.Value()
		var cmd tea.Cmd
		s.textIn, cmd = s.textIn.Update(msg)
		cmds = append(cmds, cmd)
		if s.textIn.Value() != prevVal {
			s.computeOutputs()
		}
	} else {
		prevVal = s.fileIn.Value()
		var cmd tea.Cmd
		s.fileIn, cmd = s.fileIn.Update(msg)
		cmds = append(cmds, cmd)
		// For file input, we only compute on Enter key to avoid reading disk on every keystroke
	}

	return s, tea.Batch(cmds...)
}

func (s *HashScreen) computeOutputs() {
	s.statusMsg = ""
	s.statusErr = false

	if s.inputMode == HashInputText {
		val := s.textIn.Value()
		if val == "" {
			s.rows = nil
			return
		}

		s.rows = []hashResultRow{
			{"MD5", hash.GenerateHash(val, hash.HashMD5)},
			{"SHA-1", hash.GenerateHash(val, hash.HashSHA1)},
			{"SHA-256", hash.GenerateHash(val, hash.HashSHA256)},
			{"SHA-512", hash.GenerateHash(val, hash.HashSHA512)},
			{"SHA3-256", hash.GenerateHash(val, hash.HashSHA3256)},
			{"Base64", hash.EncodeString(val, hash.EncBase64)},
			{"Hex Representation", hash.EncodeString(val, hash.EncHex)},
			{"URL Encoded", hash.EncodeString(val, hash.EncURL)},
		}
	} else {
		filePath := CleanPath(s.fileIn.Value())
		s.fileIn.SetValue(filePath)
		if filePath == "" {
			s.rows = nil
			return
		}

		// Read file contents safely
		f, err := os.Open(filePath)
		if err != nil {
			s.statusMsg = fmt.Sprintf("Error: Cannot open file: %v", err)
			s.statusErr = true
			s.rows = nil
			return
		}
		defer f.Close()

		// Read file completely to compute hashes
		data, err := io.ReadAll(f)
		if err != nil {
			s.statusMsg = fmt.Sprintf("Error: Failed to read file: %v", err)
			s.statusErr = true
			s.rows = nil
			return
		}

		s.rows = []hashResultRow{
			{"File MD5", hash.GenerateHash(string(data), hash.HashMD5)},
			{"File SHA-1", hash.GenerateHash(string(data), hash.HashSHA1)},
			{"File SHA-256", hash.GenerateHash(string(data), hash.HashSHA256)},
			{"File SHA-512", hash.GenerateHash(string(data), hash.HashSHA512)},
			{"File SHA3-256", hash.GenerateHash(string(data), hash.HashSHA3256)},
			{"File Base64 size", fmt.Sprintf("%d bytes", len(hash.EncodeString(string(data), hash.EncBase64)))},
		}
	}
}

func (s *HashScreen) copySelectedToClipboard() {
	if s.selectedRow < 0 || s.selectedRow >= len(s.rows) {
		return
	}
	val := s.rows[s.selectedRow].value

	// Natively write to macOS clipboard using pbcopy
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.statusMsg = "Failed to copy to clipboard"
		s.statusErr = true
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, val)
	}()

	err = cmd.Run()
	if err != nil {
		s.statusMsg = "Failed to copy to clipboard"
		s.statusErr = true
	} else {
		s.statusMsg = fmt.Sprintf("Copied %s value to system clipboard!", s.rows[s.selectedRow].label)
		s.statusErr = false
	}
}

func (s *HashScreen) View(width int, height int) string {
	var sections []string

	sections = append(sections, TitleStyle.Render("Text & Hash Utility"))
	sections = append(sections, SubtitleStyle.Render("Compute cryptographic hashes and text representations of strings or local files."))

	var fields []string

	// 1. Input Mode Selection
	modeLabel := FieldLabel.Render("Input Type: ")
	modeVal := " [Plain Text] "
	if s.inputMode == HashInputFile {
		modeVal = " [Local File] "
	}
	if s.focused && s.focusIndex == 0 {
		modeVal = HighlightActive.Render(modeVal)
	} else {
		modeVal = TextPrimary.Render(modeVal)
	}
	fields = append(fields, fmt.Sprintf("%s%s\n", modeLabel, modeVal))

	// 2. Input Box
	var inputLabel string
	var inputBox string
	if s.inputMode == HashInputText {
		inputLabel = FieldLabel.Render("Plain Text String:")
		inputBox = s.textIn.View()
	} else {
		inputLabel = FieldLabel.Render("File Path to Hash:")
		inputBox = s.fileIn.View()
	}

	if s.focused && s.focusIndex == 1 {
		inputBox = InputFocused.Render(inputBox)
	} else {
		inputBox = InputUnfocused.Render(inputBox)
	}
	fields = append(fields, fmt.Sprintf("%s\n%s\n", inputLabel, inputBox))

	sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))

	// 3. Grid representation of results
	if len(s.rows) > 0 {
		var gridRows []string
		headerText := "Calculated Results (Press [Enter] or [y] to copy highlighted row):"
		if s.focused && s.focusIndex == 2 {
			gridRows = append(gridRows, TextSecondary.Bold(true).Render(headerText))
		} else {
			gridRows = append(gridRows, SectionHeader.Render(headerText))
		}

		for i, r := range s.rows {
			// Limit printed character length for terminal neatness
			valDisplay := r.value
			if len(valDisplay) > 50 {
				valDisplay = valDisplay[:47] + "..."
			}

			rowContent := fmt.Sprintf("  %-18s %s", r.label+":", valDisplay)

			if i == s.selectedRow && s.focused && s.focusIndex == 2 {
				// Highlight selected row in grid
				gridRows = append(gridRows, HighlightActive.Render(" > "+rowContent))
			} else {
				gridRows = append(gridRows, "   "+rowContent)
			}
		}

		sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, gridRows...)))
	}

	// Status messages
	if s.statusMsg != "" {
		if s.statusErr {
			sections = append(sections, StatusError.Render("✖ "+s.statusMsg))
		} else {
			sections = append(sections, StatusSuccess.Render("✔ "+s.statusMsg))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
