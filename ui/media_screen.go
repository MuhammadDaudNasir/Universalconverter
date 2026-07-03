package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"universal-converter/pkg/media"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MediaScreen struct {
	focused      bool
	focusIndex   int
	isExtract    bool // true = extract audio, false = transcode format

	// Form fields
	srcInput     textinput.Model
	dstInput     textinput.Model
	formats      []string
	formatIndex  int

	// Background execution
	running      bool
	spinner      spinner.Model
	logView      viewport.Model
	logLines     []string
	statusMsg    string
	statusErr    bool

	// Channels
	logChan      chan string
	resultChan   chan error
}

func NewMediaScreen() *MediaScreen {
	src := textinput.New()
	src.Placeholder = "/path/to/video.mp4 or photo.png"
	src.Width = 50

	dst := textinput.New()
	dst.Placeholder = "/path/to/output.mkv or photo.webp"
	dst.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(SecondaryColor)

	vp := viewport.New(60, 8)
	vp.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderMutedCol)
	vp.SetContent("FFmpeg logs will stream here during operation...")

	return &MediaScreen{
		isExtract:   false,
		srcInput:    src,
		dstInput:    dst,
		formats:     []string{"mp4", "mkv", "webm", "mp3", "wav", "flac", "aac", "png", "jpg", "webp"},
		formatIndex: 0,
		spinner:     s,
		logView:     vp,
	}
}

func (s *MediaScreen) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, spinner.Tick)
}

func (s *MediaScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.srcInput.Blur()
		s.dstInput.Blur()
	} else {
		s.refocus()
	}
}

func (s *MediaScreen) refocus() {
	s.srcInput.Blur()
	s.dstInput.Blur()

	if s.focusIndex == 1 {
		s.srcInput.Focus()
	} else if s.focusIndex == 2 {
		s.dstInput.Focus()
	}
}

func (s *MediaScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.running {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			s.spinner, cmd = s.spinner.Update(msg)
			return s, cmd

		case LogMsg:
			s.logLines = append(s.logLines, string(msg))
			// Keep only last 100 lines to avoid memory leak
			if len(s.logLines) > 100 {
				s.logLines = s.logLines[len(s.logLines)-100:]
			}
			s.logView.SetContent(strings.Join(s.logLines, "\n"))
			s.logView.GotoBottom()
			return s, pollMediaLogs(s.logChan)

		case error:
			s.running = false
			if msg != nil {
				s.statusMsg = fmt.Sprintf("FFmpeg failed: %v", msg)
				s.statusErr = true
			} else {
				s.statusMsg = "Media operation completed successfully!"
				s.statusErr = false
			}
			return s, nil
		}

		// Allow users to scroll the logs viewport during running
		if s.focused && s.focusIndex == 5 {
			var cmd tea.Cmd
			s.logView, cmd = s.logView.Update(msg)
			cmds = append(cmds, cmd)
		}
		return s, tea.Batch(cmds...)
	}

	if s.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				s.focusIndex = (s.focusIndex + 1) % 5
				s.refocus()
				return s, nil
			case "shift+tab":
				s.focusIndex = (s.focusIndex - 1 + 5) % 5
				s.refocus()
				return s, nil
			case "up":
				if s.focusIndex > 0 {
					s.focusIndex--
					s.refocus()
				}
				return s, nil
			case "down":
				if s.focusIndex < 4 {
					s.focusIndex++
					s.refocus()
				}
				return s, nil

			case "left", "right", " ":
				if s.focusIndex == 0 {
					s.isExtract = !s.isExtract
					s.statusMsg = ""
					// Adjust formats list depending on mode
					if s.isExtract {
						s.formats = []string{"mp3", "wav", "flac", "aac"}
						if s.formatIndex >= len(s.formats) {
							s.formatIndex = 0
						}
					} else {
						s.formats = []string{"mp4", "mkv", "webm", "mp3", "wav", "flac", "aac", "png", "jpg", "webp"}
					}
					s.updateSuggestedDst()
					return s, nil
				}
				if s.focusIndex == 3 {
					if msg.String() == "left" {
						s.formatIndex = (s.formatIndex - 1 + len(s.formats)) % len(s.formats)
					} else {
						s.formatIndex = (s.formatIndex + 1) % len(s.formats)
					}
					s.updateSuggestedDst()
					return s, nil
				}

			case "enter":
				if s.focusIndex == 4 {
					return s, s.startConversion()
				}
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

	if s.focusIndex == 1 {
		var cmd tea.Cmd
		s.srcInput, cmd = s.srcInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if s.focusIndex == 2 {
		var cmd tea.Cmd
		s.dstInput, cmd = s.dstInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

func (s *MediaScreen) updateSuggestedDst() {
	src := CleanPath(s.srcInput.Value())
	s.srcInput.SetValue(src)
	if src == "" {
		return
	}
	ext := "." + s.formats[s.formatIndex]
	s.dstInput.SetValue(strings.TrimSuffix(src, filepath.Ext(src)) + ext)
}

func (s *MediaScreen) startConversion() tea.Cmd {
	src := CleanPath(s.srcInput.Value())
	dst := CleanPath(s.dstInput.Value())
	s.srcInput.SetValue(src)
	s.dstInput.SetValue(dst)

	if src == "" || dst == "" {
		s.statusMsg = "Error: Input and Output file paths are required"
		s.statusErr = true
		return nil
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		s.statusMsg = "Error: Input file does not exist"
		s.statusErr = true
		return nil
	}

	s.running = true
	s.statusMsg = "Processing media streams..."
	s.statusErr = false
	s.logLines = []string{}
	s.logView.SetContent("Spawning FFmpeg process...")
	s.logChan = make(chan string, 100)
	s.resultChan = make(chan error, 1)

	go func() {
		var err error
		// Verify if it's standard images and we can do it natively, or use ffmpeg
		extSrc := strings.ToLower(filepath.Ext(src))
		extDst := strings.ToLower(filepath.Ext(dst))
		isNativeImage := (extSrc == ".jpg" || extSrc == ".jpeg" || extSrc == ".png" || extSrc == ".gif") &&
			(extDst == ".jpg" || extDst == ".jpeg" || extDst == ".png" || extDst == ".gif")

		if !s.isExtract && isNativeImage {
			s.logChan <- "Converting image natively..."
			err = media.ConvertImageNatively(src, dst)
		} else {
			if s.isExtract {
				err = media.ExtractAudio(src, dst, s.formats[s.formatIndex], s.logChan)
			} else {
				err = media.ConvertMedia(src, dst, s.logChan)
			}
		}
		close(s.logChan)
		s.resultChan <- err
	}()

	return tea.Batch(
		s.spinner.Tick,
		pollMediaLogs(s.logChan),
		pollMediaResult(s.resultChan),
	)
}

func pollMediaLogs(ch chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return LogMsg(line)
	}
}

func pollMediaResult(ch chan error) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (s *MediaScreen) View(width int, height int) string {
	var sections []string

	sections = append(sections, TitleStyle.Render("FFmpeg Media Converter"))
	sections = append(sections, SubtitleStyle.Render("Perform format swaps for videos, audio tracks, and images, or extract audio tracks."))

	if s.running {
		spinnerView := s.spinner.View()
		logBox := s.logView.View()

		runContent := fmt.Sprintf(
			"\n  %s  %s\n\n%s\n",
			spinnerView,
			StatusPending.Render("Encoding media streams..."),
			logBox,
		)
		sections = append(sections, BoxBorder.Render(runContent))
	} else {
		var fields []string

		// 1. Mode Select
		modeLabel := FieldLabel.Render("Media Task: ")
		modeVal := " [Convert Format] "
		if s.isExtract {
			modeVal = " [Extract Audio Track] "
		}
		if s.focused && s.focusIndex == 0 {
			modeVal = HighlightActive.Render(modeVal)
		} else {
			modeVal = TextPrimary.Render(modeVal)
		}
		fields = append(fields, fmt.Sprintf("%s%s\n", modeLabel, modeVal))

		// 2. Source path
		srcLabel := FieldLabel.Render("Source Media Path:")
		srcBox := s.srcInput.View()
		if s.focused && s.focusIndex == 1 {
			srcBox = InputFocused.Render(srcBox)
		} else {
			srcBox = InputUnfocused.Render(srcBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", srcLabel, srcBox))

		// 3. Destination path
		dstLabel := FieldLabel.Render("Destination Media Path:")
		dstBox := s.dstInput.View()
		if s.focused && s.focusIndex == 2 {
			dstBox = InputFocused.Render(dstBox)
		} else {
			dstBox = InputUnfocused.Render(dstBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", dstLabel, dstBox))

		// 4. Format Selector
		fmtLabel := FieldLabel.Render("Target Extension:")
		var formatOptions string
		for i, fmtOpt := range s.formats {
			optStr := fmt.Sprintf(" %s ", strings.ToUpper(fmtOpt))
				if i == s.formatIndex {
					if s.focused && s.focusIndex == 3 {
						formatOptions += HighlightActive.Render(optStr) + " "
					} else {
						formatOptions += TextPrimary.Render(optStr) + " "
					}
				} else {
				formatOptions += lipgloss.NewStyle().Foreground(TextMutedColor).Render(optStr) + " "
			}
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", fmtLabel, formatOptions))

		// 5. Submit Button
		btnText := "  Start Media Operation  "
		var btn string
		if s.focused && s.focusIndex == 4 {
			btn = BtnFocused.Render(btnText)
		} else {
			btn = BtnUnfocused.Render(btnText)
		}
		fields = append(fields, btn)

		sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))
	}

	// Status Message
	if s.statusMsg != "" {
		if s.statusErr {
			sections = append(sections, StatusError.Render("✖ "+s.statusMsg))
		} else {
			sections = append(sections, StatusSuccess.Render("✔ "+s.statusMsg))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
