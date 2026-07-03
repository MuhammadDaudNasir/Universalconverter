package ui

import (
	"fmt"
	"os"
	"strings"

	"universal-converter/pkg/encrypt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type EncryptScreen struct {
	focused      bool
	focusIndex   int
	isEncrypt    bool // true = encrypt, false = decrypt

	// Form fields
	srcInput     textinput.Model
	dstInput     textinput.Model
	passInput    textinput.Model
	algorithms   []encrypt.Algorithm
	algIndex     int

	// Background worker state
	running      bool
	progress     progress.Model
	spinner      spinner.Model
	currentProg  float64
	statusMsg    string
	statusErr    bool

	// Channels for background execution
	progressChan chan float64
	resultChan   chan taskFinishedMsg
}

func NewEncryptScreen() *EncryptScreen {
	src := textinput.New()
	src.Placeholder = "/path/to/source/file"
	src.Width = 50

	dst := textinput.New()
	dst.Placeholder = "/path/to/output/file.enc"
	dst.Width = 50

	pass := textinput.New()
	pass.Placeholder = "Enter secure password passphrase"
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'
	pass.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SecondaryColor)

	p := progress.New(progress.WithDefaultGradient())

	return &EncryptScreen{
		isEncrypt:  true,
		srcInput:   src,
		dstInput:   dst,
		passInput:  pass,
		algorithms: []encrypt.Algorithm{encrypt.AlgAES256GCM, encrypt.AlgChaCha20ID},
		algIndex:   0,
		progress:   p,
		spinner:    s,
	}
}

func (s *EncryptScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *EncryptScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.srcInput.Blur()
		s.dstInput.Blur()
		s.passInput.Blur()
	} else {
		s.refocus()
	}
}

func (s *EncryptScreen) refocus() {
	s.srcInput.Blur()
	s.dstInput.Blur()
	s.passInput.Blur()

	switch s.focusIndex {
	case 1:
		s.srcInput.Focus()
	case 2:
		s.dstInput.Focus()
	case 3:
		s.passInput.Focus()
	}
}

func (s *EncryptScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.running {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			s.spinner, cmd = s.spinner.Update(msg)
			return s, cmd

		case ProgressMsg:
			s.currentProg = float64(msg)
			return s, pollProgress(s.progressChan)

		case taskFinishedMsg:
			s.running = false
			s.passInput.SetValue("") // Clear password field for security
			if msg.err != nil {
				s.statusMsg = fmt.Sprintf("Error: %v", msg.err)
				s.statusErr = true
			} else {
				s.statusMsg = "Operation completed successfully!"
				s.statusErr = false
			}
			return s, nil
		}
		return s, nil
	}

	if s.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				limit := 6
				if !s.isEncrypt {
					limit = 5 // Skip algorithm select during decryption
				}
				s.focusIndex = (s.focusIndex + 1) % limit
				s.refocus()
				return s, nil
			case "shift+tab":
				limit := 6
				if !s.isEncrypt {
					limit = 5
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
				limit := 5
				if s.isEncrypt {
					limit = 6
				}
				if s.focusIndex < limit-1 {
					s.focusIndex++
					s.refocus()
				}
				return s, nil

			case "left", "right", " ":
				if s.focusIndex == 0 {
					s.isEncrypt = !s.isEncrypt
					s.statusMsg = ""
					s.updateSuggestedDst()
					return s, nil
				}
				if s.focusIndex == 4 && s.isEncrypt {
					if msg.String() == "left" {
						s.algIndex = (s.algIndex - 1 + len(s.algorithms)) % len(s.algorithms)
					} else {
						s.algIndex = (s.algIndex + 1) % len(s.algorithms)
					}
					return s, nil
				}

			case "enter":
				// If on action button
				actionIdx := 5
				if !s.isEncrypt {
					actionIdx = 4
				}
				if s.focusIndex == actionIdx {
					return s, s.startOperation()
				}

				// Advance fields
				if s.focusIndex == 1 {
					s.focusIndex = 2
					s.refocus()
					s.updateSuggestedDst()
					return s, nil
				} else if s.focusIndex == 2 {
					s.focusIndex = 3
					s.refocus()
					return s, nil
				} else if s.focusIndex == 3 {
					if s.isEncrypt {
						s.focusIndex = 4
					} else {
						s.focusIndex = 4 // Submitt button during decrypt
					}
					s.refocus()
					return s, nil
				}
			}
		}
	}

	// Update text inputs
	switch s.focusIndex {
	case 1:
		var cmd tea.Cmd
		s.srcInput, cmd = s.srcInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		var cmd tea.Cmd
		s.dstInput, cmd = s.dstInput.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		var cmd tea.Cmd
		s.passInput, cmd = s.passInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

func (s *EncryptScreen) updateSuggestedDst() {
	src := CleanPath(s.srcInput.Value())
	s.srcInput.SetValue(src)
	if src == "" {
		return
	}

	if s.isEncrypt {
		s.dstInput.SetValue(src + ".enc")
	} else {
		// Strip .enc if possible
		s.dstInput.SetValue(strings.TrimSuffix(src, ".enc"))
	}
}

func (s *EncryptScreen) startOperation() tea.Cmd {
	src := CleanPath(s.srcInput.Value())
	dst := CleanPath(s.dstInput.Value())
	pass := s.passInput.Value()
	s.srcInput.SetValue(src)
	s.dstInput.SetValue(dst)

	if src == "" || dst == "" || pass == "" {
		s.statusMsg = "Error: All fields (source, destination, password) are required"
		s.statusErr = true
		return nil
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		s.statusMsg = "Error: Source file does not exist"
		s.statusErr = true
		return nil
	}

	s.running = true
	s.statusMsg = "Processing cryptography..."
	s.statusErr = false
	s.currentProg = 0.0
	s.progressChan = make(chan float64, 100)
	s.resultChan = make(chan taskFinishedMsg, 1)

	go func() {
		var err error
		if s.isEncrypt {
			choiceAlg := s.algorithms[s.algIndex]
			err = encrypt.EncryptFile(src, dst, pass, choiceAlg, func(p float64) {
				s.progressChan <- p
			})
		} else {
			err = encrypt.DecryptFile(src, dst, pass, func(p float64) {
				s.progressChan <- p
			})
		}
		close(s.progressChan)
		s.resultChan <- taskFinishedMsg{err: err}
	}()

	return tea.Batch(
		s.spinner.Tick,
		pollProgress(s.progressChan),
		pollResult(s.resultChan),
	)
}

func (s *EncryptScreen) View(width int, height int) string {
	var sections []string

	sections = append(sections, TitleStyle.Render("Cryptographic Shield"))
	sections = append(sections, SubtitleStyle.Render("Secure files with military-grade AES-256-GCM or ChaCha20-Poly1305 using Argon2id key derivation."))

	if s.running {
		progressView := s.progress.ViewAs(s.currentProg)
		spinnerView := s.spinner.View()

		runContent := fmt.Sprintf(
			"\n  %s  %s  %.1f%%\n\n  %s\n",
			spinnerView,
			StatusPending.Render("Processing cryptographic streams..."),
			s.currentProg*100,
			progressView,
		)
		sections = append(sections, BoxBorder.Render(runContent))
	} else {
		var fields []string

		// 1. Mode Select
		modeLabel := FieldLabel.Render("Crypto Mode: ")
		modeVal := " [Encrypt File] "
		if !s.isEncrypt {
			modeVal = " [Decrypt File] "
		}
		if s.focused && s.focusIndex == 0 {
			modeVal = HighlightActive.Render(modeVal)
		} else {
			modeVal = TextPrimary.Render(modeVal)
		}
		fields = append(fields, fmt.Sprintf("%s%s\n", modeLabel, modeVal))

		// 2. Source path input
		srcLabel := FieldLabel.Render("Source File Path:")
		srcBox := s.srcInput.View()
		if s.focused && s.focusIndex == 1 {
			srcBox = InputFocused.Render(srcBox)
		} else {
			srcBox = InputUnfocused.Render(srcBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", srcLabel, srcBox))

		// 3. Destination path input
		dstLabel := FieldLabel.Render("Destination File Path:")
		dstBox := s.dstInput.View()
		if s.focused && s.focusIndex == 2 {
			dstBox = InputFocused.Render(dstBox)
		} else {
			dstBox = InputUnfocused.Render(dstBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", dstLabel, dstBox))

		// 4. Password phrase input
		passLabel := FieldLabel.Render("Secret Passphrase:")
		passBox := s.passInput.View()
		if s.focused && s.focusIndex == 3 {
			passBox = InputFocused.Render(passBox)
		} else {
			passBox = InputUnfocused.Render(passBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", passLabel, passBox))

		// 5. Algorithm select (Encrypt only)
		if s.isEncrypt {
			algLabel := FieldLabel.Render("Cipher Algorithm:")
			var algOptions string
			for i, algOpt := range s.algorithms {
				var optName string
				switch algOpt {
				case encrypt.AlgAES256GCM:
					optName = "AES-256-GCM"
				case encrypt.AlgChaCha20ID:
					optName = "ChaCha20-Poly1305"
				}
				optStr := fmt.Sprintf(" %s ", optName)
				if i == s.algIndex {
					if s.focused && s.focusIndex == 4 {
						algOptions += HighlightActive.Render(optStr) + " "
					} else {
						algOptions += TextPrimary.Render(optStr) + " "
					}
				} else {
					algOptions += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
				}
			}
			fields = append(fields, fmt.Sprintf("%s\n%s\n", algLabel, algOptions))
		}

		// 6. Action Button
		btnText := "  Secure Encrypt File  "
		if !s.isEncrypt {
			btnText = "  Decrypt File  "
		}
		var btn string
		actionIdx := 5
		if !s.isEncrypt {
			actionIdx = 4
		}
		if s.focused && s.focusIndex == actionIdx {
			btn = BtnFocused.Render(btnText)
		} else {
			btn = BtnUnfocused.Render(btnText)
		}
		fields = append(fields, btn)

		sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))
	}

	// Status output
	if s.statusMsg != "" {
		if s.statusErr {
			sections = append(sections, StatusError.Render("✖ "+s.statusMsg))
		} else {
			sections = append(sections, StatusSuccess.Render("✔ "+s.statusMsg))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
