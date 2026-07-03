package media

import (
	"bufio"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertImageNatively attempts to convert between standard image formats using Go standard library.
func ConvertImageNatively(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Decode source image
	img, format, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Detect target format based on extension
	ext := strings.ToLower(filepath.Ext(dstPath))
	switch ext {
	case ".png":
		return png.Encode(dstFile, img)
	case ".jpg", ".jpeg":
		return jpeg.Encode(dstFile, img, &jpeg.Options{Quality: 90})
	case ".gif":
		return gif.Encode(dstFile, img, nil)
	default:
		return fmt.Errorf("unsupported native format '%s' (fallback to FFmpeg recommended)", format)
	}
}

// ConvertMedia runs FFmpeg to convert audio/video/images and streams stdout/stderr lines to a channel.
func ConvertMedia(srcPath, dstPath string, logChan chan<- string) error {
	// Execute FFmpeg: -y (overwrite), -i (input)
	// We can add presets or let FFmpeg auto-determine best stream conversion based on output file extension.
	cmd := exec.Command("ffmpeg", "-y", "-i", srcPath, dstPath)
	return runFFmpeg(cmd, logChan)
}

// ExtractAudio extracts the audio track from a video file.
func ExtractAudio(srcPath, dstPath string, format string, logChan chan<- string) error {
	var args []string
	switch strings.ToLower(format) {
	case "mp3":
		args = []string{"-y", "-i", srcPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", dstPath}
	case "wav":
		args = []string{"-y", "-i", srcPath, "-vn", "-acodec", "pcm_s16le", dstPath}
	case "flac":
		args = []string{"-y", "-i", srcPath, "-vn", "-acodec", "flac", dstPath}
	case "aac":
		args = []string{"-y", "-i", srcPath, "-vn", "-acodec", "aac", "-b:a", "192k", dstPath}
	default:
		// Fallback to simple stream copy/auto-transcode
		args = []string{"-y", "-i", srcPath, "-vn", dstPath}
	}

	cmd := exec.Command("ffmpeg", args...)
	return runFFmpeg(cmd, logChan)
}

func runFFmpeg(cmd *exec.Cmd, logChan chan<- string) error {
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Stream logs in a separate goroutine
	go func() {
		defer stderr.Close()
		reader := bufio.NewReader(stderr)
		for {
			line, err := reader.ReadString('\r') // FFmpeg often uses \r for in-place progress updates
			if err != nil {
				if err != io.EOF {
					if logChan != nil {
						logChan <- fmt.Sprintf("[Error reading log]: %s", err)
					}
				}
				break
			}
			line = strings.TrimSpace(line)
			if line != "" {
				if logChan != nil {
					logChan <- line
				}
			}
		}
	}()

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}
