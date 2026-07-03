package compress

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCompressDecompressRoundtrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "compress_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source folder layout
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "nested"), 0755); err != nil {
		t.Fatalf("failed to create src folder structure: %v", err)
	}

	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "nested", "file2.txt")

	data1 := []byte("Universal Converter Compression Engine file 1 content")
	data2 := []byte("Nested file 2 content.")

	if err := os.WriteFile(file1, data1, 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, data2, 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	formats := []Format{FormatZip, FormatTarGz}

	for _, fmtChoice := range formats {
		t.Run(string(fmtChoice), func(t *testing.T) {
			archivePath := filepath.Join(tempDir, "archive."+string(fmtChoice))
			destDir := filepath.Join(tempDir, "out_"+string(fmtChoice))
			if err := os.MkdirAll(destDir, 0755); err != nil {
				t.Fatalf("failed to create dest folder: %v", err)
			}

			// 1. Compress
			err := Compress([]string{srcDir}, archivePath, fmtChoice, func(p float64) {})
			if err != nil {
				t.Fatalf("compression failed: %v", err)
			}

			// 2. Decompress
			err = Decompress(archivePath, destDir, func(p float64) {})
			if err != nil {
				t.Fatalf("decompression failed: %v", err)
			}

			// 3. Verify
			// File 1 path in extracted folder is destDir / "src" / "file1.txt"
			extractedFile1 := filepath.Join(destDir, "src", "file1.txt")
			extractedFile2 := filepath.Join(destDir, "src", "nested", "file2.txt")

			dec1, err := os.ReadFile(extractedFile1)
			if err != nil {
				t.Fatalf("failed to read extracted file1: %v", err)
			}
			dec2, err := os.ReadFile(extractedFile2)
			if err != nil {
				t.Fatalf("failed to read extracted file2: %v", err)
			}

			if !bytes.Equal(data1, dec1) {
				t.Errorf("extracted file1 mismatch")
			}
			if !bytes.Equal(data2, dec2) {
				t.Errorf("extracted file2 mismatch")
			}
		})
	}
}
