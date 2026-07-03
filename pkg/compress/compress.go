package compress

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// ProgressWriter tracks bytes written and updates progress.
type ProgressWriter struct {
	Writer     io.Writer
	Total      int64
	Current    int64
	OnProgress func(float64)
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if n > 0 {
		pw.Current += int64(n)
		if pw.Total > 0 && pw.OnProgress != nil {
			pw.OnProgress(float64(pw.Current) / float64(pw.Total))
		}
	}
	return n, err
}

// ProgressReader tracks bytes read and updates progress.
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Current    int64
	OnProgress func(float64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Current += int64(n)
		if pr.Total > 0 && pr.OnProgress != nil {
			pr.OnProgress(float64(pr.Current) / float64(pr.Total))
		}
	}
	return n, err
}

// Format represents supported compression formats.
type Format string

const (
	FormatZip    Format = "zip"
	FormatTar    Format = "tar"
	FormatTarGz  Format = "tar.gz"
	FormatTarXz  Format = "tar.xz"
	FormatTarZst Format = "tar.zst"
	FormatGzip   Format = "gz"
	FormatXz     Format = "xz"
	FormatZstd   Format = "zst"
	FormatBrotli Format = "br"
)

// GetTotalSize calculates the total size of files/folders in srcPaths.
func GetTotalSize(srcPaths []string) (int64, error) {
	var total int64
	for _, path := range srcPaths {
		info, err := os.Stat(path)
		if err != nil {
			return 0, err
		}
		if !info.IsDir() {
			total += info.Size()
			continue
		}
		err = filepath.Walk(path, func(_ string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !fileInfo.IsDir() {
				total += fileInfo.Size()
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

// Compress packages and compresses srcPaths into dstPath using the specified format.
func Compress(srcPaths []string, dstPath string, format Format, onProgress func(float64)) error {
	totalSize, err := GetTotalSize(srcPaths)
	if err != nil {
		return fmt.Errorf("failed to calculate source size: %w", err)
	}
	if totalSize == 0 {
		totalSize = 1 // Avoid division by zero
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// progWriter is omitted in favor of precise source read tracking

	// We wrap progWriter with compression layers. Note that for zip and tar,
	// the progress is tracked at the writer level (which is compressed output).
	// This actually reflects the disk write progress. Let's make it reflect either read or write bytes.
	// Tracking the raw file reading bytes is more accurate for progress because it reaches exactly 100% at the end.
	// Let's modify progress tracking to wrap the input files instead, which guarantees precision!
	
	// Let's update: we will track progress based on the bytes read from the input files.
	var bytesRead int64
	updateReadProgress := func(n int) {
		bytesRead += int64(n)
		if onProgress != nil {
			onProgress(float64(bytesRead) / float64(totalSize))
		}
	}

	switch format {
	case FormatZip:
		return compressZip(srcPaths, dstFile, updateReadProgress)
	case FormatTar, FormatTarGz, FormatTarXz, FormatTarZst:
		return compressTar(srcPaths, dstFile, format, updateReadProgress)
	case FormatGzip, FormatXz, FormatZstd, FormatBrotli:
		if len(srcPaths) != 1 {
			return fmt.Errorf("single file compression formats support exactly 1 source path")
		}
		return compressSingleFile(srcPaths[0], dstFile, format, updateReadProgress)
	default:
		return fmt.Errorf("unsupported compression format: %s", format)
	}
}

// compressZip archives files into a zip.
func compressZip(srcPaths []string, w io.Writer, onRead func(int)) error {
	archive := zip.NewWriter(w)
	defer archive.Close()

	for _, src := range srcPaths {
		if _, err := os.Stat(src); err != nil {
			return err
		}

		baseDir := filepath.Dir(src)

		err := filepath.Walk(src, func(path string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				return err
			}

			header.Name = relPath
			if fileInfo.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if fileInfo.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			buf := make([]byte, 32*1024)
			for {
				n, readErr := file.Read(buf)
				if n > 0 {
					_, writeErr := writer.Write(buf[:n])
					if writeErr != nil {
						return writeErr
					}
					onRead(n)
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					return readErr
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// compressTar archives files into a tar, optionally compressed.
func compressTar(srcPaths []string, w io.Writer, format Format, onRead func(int)) error {
	var cw io.WriteCloser
	var err error

	switch format {
	case FormatTarGz:
		cw = gzip.NewWriter(w)
	case FormatTarXz:
		cw, err = xz.NewWriter(w)
		if err != nil {
			return err
		}
	case FormatTarZst:
		cw, err = zstd.NewWriter(w)
		if err != nil {
			return err
		}
	default:
		// Plain tar, no compression writer wrapper needed.
		cw = nopWriteCloser{w}
	}
	defer cw.Close()

	tw := tar.NewWriter(cw)
	defer tw.Close()

	for _, src := range srcPaths {
		if _, err := os.Stat(src); err != nil {
			return err
		}

		baseDir := filepath.Dir(src)

		err := filepath.Walk(src, func(path string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
			if err != nil {
				return err
			}

			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if fileInfo.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			buf := make([]byte, 32*1024)
			for {
				n, readErr := file.Read(buf)
				if n > 0 {
					_, writeErr := tw.Write(buf[:n])
					if writeErr != nil {
						return writeErr
					}
					onRead(n)
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					return readErr
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// compressSingleFile compresses a single file using gzip, xz, zstd, or brotli.
func compressSingleFile(src string, w io.Writer, format Format, onRead func(int)) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	var cw io.WriteCloser
	switch format {
	case FormatGzip:
		cw = gzip.NewWriter(w)
	case FormatXz:
		cw, err = xz.NewWriter(w)
		if err != nil {
			return err
		}
	case FormatZstd:
		cw, err = zstd.NewWriter(w)
		if err != nil {
			return err
		}
	case FormatBrotli:
		cw = brotli.NewWriterLevel(w, brotli.DefaultCompression)
	default:
		return fmt.Errorf("unsupported single file format: %s", format)
	}
	defer cw.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			_, writeErr := cw.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			onRead(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// Decompress extracts files from srcPath to dstDir.
func Decompress(srcPath string, dstDir string, onProgress func(float64)) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	totalSize := info.Size()
	if totalSize == 0 {
		totalSize = 1
	}

	progressReader := &ProgressReader{
		Reader: srcFile,
		Total:  totalSize,
		OnProgress: func(ratio float64) {
			if onProgress != nil {
				onProgress(ratio)
			}
		},
	}

	// Detect format based on filename
	lowerName := strings.ToLower(srcPath)
	if strings.HasSuffix(lowerName, ".zip") {
		// zip needs random access, so we can't use progressReader directly for streaming zip header.
		// However, we can measure progress based on unpacked bytes or use progressReader for the file creation.
		// Let's implement custom zip extractor.
		return decompressZip(srcPath, dstDir, onProgress)
	} else if strings.HasSuffix(lowerName, ".tar") {
		return decompressTar(progressReader, dstDir)
	} else if strings.HasSuffix(lowerName, ".tar.gz") || strings.HasSuffix(lowerName, ".tgz") {
		gr, err := gzip.NewReader(progressReader)
		if err != nil {
			return err
		}
		defer gr.Close()
		return decompressTar(gr, dstDir)
	} else if strings.HasSuffix(lowerName, ".tar.xz") || strings.HasSuffix(lowerName, ".txz") {
		xr, err := xz.NewReader(progressReader)
		if err != nil {
			return err
		}
		return decompressTar(xr, dstDir)
	} else if strings.HasSuffix(lowerName, ".tar.zst") {
		zr, err := zstd.NewReader(progressReader)
		if err != nil {
			return err
		}
		defer zr.Close()
		return decompressTar(zr, dstDir)
	} else if strings.HasSuffix(lowerName, ".gz") {
		gr, err := gzip.NewReader(progressReader)
		if err != nil {
			return err
		}
		defer gr.Close()
		dstName := strings.TrimSuffix(filepath.Base(srcPath), ".gz")
		return decompressSingleStream(gr, filepath.Join(dstDir, dstName))
	} else if strings.HasSuffix(lowerName, ".xz") {
		xr, err := xz.NewReader(progressReader)
		if err != nil {
			return err
		}
		dstName := strings.TrimSuffix(filepath.Base(srcPath), ".xz")
		return decompressSingleStream(xr, filepath.Join(dstDir, dstName))
	} else if strings.HasSuffix(lowerName, ".zst") {
		zr, err := zstd.NewReader(progressReader)
		if err != nil {
			return err
		}
		defer zr.Close()
		dstName := strings.TrimSuffix(filepath.Base(srcPath), ".zst")
		return decompressSingleStream(zr, filepath.Join(dstDir, dstName))
	} else if strings.HasSuffix(lowerName, ".br") {
		br := brotli.NewReader(progressReader)
		dstName := strings.TrimSuffix(filepath.Base(srcPath), ".br")
		return decompressSingleStream(br, filepath.Join(dstDir, dstName))
	}

	return fmt.Errorf("unknown compression format for file: %s", srcPath)
}

func decompressZip(srcPath, dstDir string, onProgress func(float64)) error {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Calculate total uncompressed size to track extraction progress
	var totalUncompressedBytes int64
	for _, f := range r.File {
		totalUncompressedBytes += int64(f.UncompressedSize64)
	}
	if totalUncompressedBytes == 0 {
		totalUncompressedBytes = 1
	}

	var extractedBytes int64

	for _, f := range r.File {
		path := filepath.Join(dstDir, f.Name)

		// Check for Zip Slip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(dstDir)+string(os.PathSeparator)) && filepath.Clean(dstDir) != "." {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		buf := make([]byte, 32*1024)
		for {
			n, readErr := rc.Read(buf)
			if n > 0 {
				_, writeErr := dstFile.Write(buf[:n])
				if writeErr != nil {
					rc.Close()
					dstFile.Close()
					return writeErr
				}
				extractedBytes += int64(n)
				if onProgress != nil {
					onProgress(float64(extractedBytes) / float64(totalUncompressedBytes))
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				rc.Close()
				dstFile.Close()
				return readErr
			}
		}
		rc.Close()
		dstFile.Close()
	}
	return nil
}

func decompressTar(r io.Reader, dstDir string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dstDir, header.Name)

		// Check for Zip Slip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(dstDir)+string(os.PathSeparator)) && filepath.Clean(dstDir) != "." {
			return fmt.Errorf("illegal file path in tar: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, header.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}

			dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}

			if _, err := io.Copy(dstFile, tr); err != nil {
				dstFile.Close()
				return err
			}
			dstFile.Close()
		}
	}
	return nil
}

func decompressSingleStream(r io.Reader, dstPath string) error {
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, r)
	return err
}

// nopWriteCloser wraps a writer to implement io.WriteCloser with a no-op Close.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
