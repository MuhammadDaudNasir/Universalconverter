package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"strings"

	"universal-converter/pkg/compress"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CompressScreen struct {
	focused      bool
	focusIndex   int
	isCompress   bool // true = compress, false = decompress
	
	// Form fields
	srcInput     textinput.Model
	dstInput     textinput.Model
	formats      []compress.Format
	formatIndex  int

	// Background worker state
	running      bool
	progress     progress.Model
	spinner      spinner.Model
	currentProg  float64
	statusMsg    string
	statusErr    bool
	stats        map[string]string
	
	// Channels for background execution
	progressChan chan float64
	resultChan   chan taskFinishedMsg
}

type taskFinishedMsg struct {
	err   error
	stats map[string]string
}

func NewCompressScreen() *CompressScreen {
	src := textinput.New()
	src.Placeholder = "/path/to/source/file/or/folder"
	src.Width = 50

	dst := textinput.New()
	dst.Placeholder = "/path/to/output/archive.zip"
	dst.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(SecondaryColor)

	p := progress.New(progress.WithDefaultGradient())

	return &CompressScreen{
		isCompress: true,
		srcInput:   src,
		dstInput:   dst,
		formats: []compress.Format{
			compress.FormatZip,
			compress.FormatTarGz,
			compress.FormatTarXz,
			compress.FormatTarZst,
			compress.FormatGzip,
			compress.FormatXz,
			compress.FormatZstd,
			compress.FormatBrotli,
		},
		formatIndex: 0,
		progress:    p,
		spinner:     s,
	}
}

func (s *CompressScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *CompressScreen) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.srcInput.Blur()
		s.dstInput.Blur()
	} else {
		s.refocus()
	}
}

func (s *CompressScreen) refocus() {
	s.srcInput.Blur()
	s.dstInput.Blur()

	if s.focusIndex == 1 {
		s.srcInput.Focus()
	} else if s.focusIndex == 2 {
		s.dstInput.Focus()
	}
}

func (s *CompressScreen) Update(msg tea.Msg) (SubModel, tea.Cmd) {
	var cmds []tea.Cmd

	if s.running {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			s.spinner, cmd = s.spinner.Update(msg)
			return s, cmd

		case ProgressMsg:
			s.currentProg = float64(msg)
			// Wait for the next progress tick
			return s, pollProgress(s.progressChan)

		case taskFinishedMsg:
			s.running = false
			if msg.err != nil {
				s.statusMsg = fmt.Sprintf("Error: %v", msg.err)
				s.statusErr = true
			} else {
				s.statusMsg = "Operation completed successfully!"
				s.statusErr = false
				s.stats = msg.stats
			}
			return s, nil
		}
		return s, nil
	}

	// Normal inputs updating
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
					// Toggle compress/decompress mode
					s.isCompress = !s.isCompress
					s.statusMsg = ""
					s.stats = nil
					s.updateSuggestedDst()
					return s, nil
				}
				if s.focusIndex == 3 && s.isCompress {
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
					return s, s.startOperation()
				}
				// Pressing enter in inputs advances focus
				if s.focusIndex == 1 {
					s.focusIndex = 2
					s.refocus()
					s.updateSuggestedDst()
					return s, nil
				} else if s.focusIndex == 2 {
					if s.isCompress {
						s.focusIndex = 3
					} else {
						s.focusIndex = 4
					}
					s.refocus()
					return s, nil
				}
			}
		}
	}

	// Update text inputs
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

func (s *CompressScreen) updateSuggestedDst() {
	src := CleanPath(s.srcInput.Value())
	s.srcInput.SetValue(src)
	if src == "" {
		return
	}

	if s.isCompress {
		fmtExt := string(s.formats[s.formatIndex])
		s.dstInput.SetValue(strings.TrimSuffix(src, filepath.Ext(src)) + "." + fmtExt)
	} else {
		// Suggest decompress directory
		s.dstInput.SetValue(filepath.Dir(src))
	}
}

func (s *CompressScreen) startOperation() tea.Cmd {
	src := CleanPath(s.srcInput.Value())
	dst := CleanPath(s.dstInput.Value())
	s.srcInput.SetValue(src)
	s.dstInput.SetValue(dst)

	if src == "" || dst == "" {
		s.statusMsg = "Error: Source and destination paths cannot be empty"
		s.statusErr = true
		return nil
	}

	// Verify source exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		s.statusMsg = "Error: Source path does not exist"
		s.statusErr = true
		return nil
	}

	s.running = true
	s.statusMsg = "Processing..."
	s.statusErr = false
	s.currentProg = 0.0
	s.stats = nil
	s.progressChan = make(chan float64, 100)
	s.resultChan = make(chan taskFinishedMsg, 1)

	// Launch worker
	go func() {
		startTime := time.Now()
		var err error

		if s.isCompress {
			fmtChoice := s.formats[s.formatIndex]
			err = compress.Compress([]string{src}, dst, fmtChoice, func(p float64) {
				s.progressChan <- p
			})
		} else {
			err = compress.Decompress(src, dst, func(p float64) {
				s.progressChan <- p
			})
		}
		close(s.progressChan)

		var stats map[string]string
		if err == nil {
			duration := time.Since(startTime)
			if s.isCompress {
				srcInfo, _ := os.Stat(src)
				dstInfo, _ := os.Stat(dst)
				srcSize := srcInfo.Size()
				// For directory, calculate recursive size
				if srcInfo.IsDir() {
					srcSize, _ = compress.GetTotalSize([]string{src})
				}
				dstSize := dstInfo.Size()
				if srcSize == 0 {
					srcSize = 1
				}
				ratio := float64(srcSize-dstSize) / float64(srcSize) * 100
				speed := float64(srcSize) / (1024 * 1024) / duration.Seconds()

				stats = map[string]string{
					"Duration":       fmt.Sprintf("%.2fs", duration.Seconds()),
					"Original Size":  formatBytes(srcSize),
					"Output Size":    formatBytes(dstSize),
					"Ratio":          fmt.Sprintf("%.1f%% space saved", ratio),
					"Average Speed":  fmt.Sprintf("%.2f MB/s", speed),
				}
			} else {
				stats = map[string]string{
					"Duration": fmt.Sprintf("%.2fs", duration.Seconds()),
				}
			}
		}
		s.resultChan <- taskFinishedMsg{err: err, stats: stats}
	}()

	return tea.Batch(
		s.spinner.Tick,
		pollProgress(s.progressChan),
		pollResult(s.resultChan),
	)
}

func pollProgress(ch chan float64) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return nil
		}
		return ProgressMsg(p)
	}
}

func pollResult(ch chan taskFinishedMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (s *CompressScreen) View(width int, height int) string {
	var sections []string

	// Header
	sections = append(sections, TitleStyle.Render("Compression Engine"))
	sections = append(sections, SubtitleStyle.Render("Pack files with zstd, brotli, gzip, lzma, and standard zip/tar archives."))

	if s.running {
		// Loading/Progress display
		progressView := s.progress.ViewAs(s.currentProg)
		spinnerView := s.spinner.View()
		
		runContent := fmt.Sprintf(
			"\n  %s  %s  %.1f%%\n\n  %s\n",
			spinnerView,
			StatusPending.Render("Processing files..."),
			s.currentProg*100,
			progressView,
		)
		sections = append(sections, BoxBorder.Render(runContent))
	} else {
		// Form Fields
		var fields []string

		// 1. Mode Select
		modeLabel := FieldLabel.Render("Operation Mode: ")
		modeVal := " [Compress] "
		if !s.isCompress {
			modeVal = " [Decompress] "
		}
		if s.focused && s.focusIndex == 0 {
			modeVal = HighlightActive.Render(modeVal)
		} else {
			modeVal = TextPrimary.Render(modeVal)
		}
		fields = append(fields, fmt.Sprintf("%s%s\n", modeLabel, modeVal))

		// 2. Source path input
		srcLabel := FieldLabel.Render("Source Path:")
		srcBox := s.srcInput.View()
		if s.focused && s.focusIndex == 1 {
			srcBox = InputFocused.Render(srcBox)
		} else {
			srcBox = InputUnfocused.Render(srcBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", srcLabel, srcBox))

		// 3. Destination path input
		dstLabel := FieldLabel.Render("Destination Path / Folder:")
		dstBox := s.dstInput.View()
		if s.focused && s.focusIndex == 2 {
			dstBox = InputFocused.Render(dstBox)
		} else {
			dstBox = InputUnfocused.Render(dstBox)
		}
		fields = append(fields, fmt.Sprintf("%s\n%s\n", dstLabel, dstBox))

		// 4. Format selection (Compress only)
		if s.isCompress {
			fmtLabel := FieldLabel.Render("Compression Format:")
			var formatOptions string
			for i, fmtOpt := range s.formats {
				optStr := fmt.Sprintf(" %s ", fmtOpt)
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
		}

		// 5. Submit Button
		btnText := "  Run Compression  "
		if !s.isCompress {
			btnText = "  Run Decompression  "
		}
		var btn string
		if s.focused && s.focusIndex == 4 {
			btn = BtnFocused.Render(btnText)
		} else {
			btn = BtnUnfocused.Render(btnText)
		}
		fields = append(fields, btn)

		sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, fields...)))
	}

	// Status and stats footer
	if s.statusMsg != "" {
		if s.statusErr {
			sections = append(sections, StatusError.Render("✖ "+s.statusMsg))
		} else {
			sections = append(sections, StatusSuccess.Render("✔ "+s.statusMsg))
		}
	}

	if s.stats != nil {
		var statsRows []string
		statsRows = append(statsRows, SectionHeader.Render("Metrics & Performance:"))
		for k, v := range s.stats {
			statsRows = append(statsRows, fmt.Sprintf("  • %-16s %s", k+":", TextPrimary.Render(v)))
		}
		sections = append(sections, BoxBorder.Render(lipgloss.JoinVertical(lipgloss.Left, statsRows...)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// formatBytes helper
func formatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper to strip file extensions
func stringsTrimSuffix(s, suffix string) string {
	if len(suffix) > 0 && len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
