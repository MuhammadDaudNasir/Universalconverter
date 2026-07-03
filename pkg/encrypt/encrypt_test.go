package encrypt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "crypto_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalData := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Universal TUI Converter app encryption validation.")
	srcPath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(srcPath, originalData, 0644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	passphrase := "super-secure-passphrase-123!"

	algorithms := []Algorithm{AlgAES256GCM, AlgChaCha20ID}

	for _, alg := range algorithms {
		t.Run(string(alg), func(t *testing.T) {
			encPath := filepath.Join(tempDir, string(alg)+".enc")
			decPath := filepath.Join(tempDir, string(alg)+".dec")

			// 1. Encrypt
			err := EncryptFile(srcPath, encPath, passphrase, alg, func(p float64) {})
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// 2. Decrypt
			err = DecryptFile(encPath, decPath, passphrase, func(p float64) {})
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// 3. Verify
			decData, err := os.ReadFile(decPath)
			if err != nil {
				t.Fatalf("failed to read decrypted file: %v", err)
			}

			if !bytes.Equal(originalData, decData) {
				t.Errorf("decrypted data does not match original data")
			}
		})
	}
}
