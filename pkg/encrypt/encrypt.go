package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	MagicBytes = "UCENC" // Universal Converter Encrypted
	Version    = 1
	ChunkSize  = 64 * 1024 // 64 KB chunks
	TagSize    = 16        // AEAD tag size
)

type Algorithm byte

const (
	AlgAES256GCM       Algorithm = 1
	AlgChaCha20ID      Algorithm = 2
)

// Argon2Parameters contains configuration for key derivation.
type Argon2Parameters struct {
	Time    uint32
	Memory  uint32
	Threads uint8
}

var DefaultArgon2Params = Argon2Parameters{
	Time:    3,
	Memory:  64 * 1024, // 64 MB
	Threads: 4,
}

// DeriveKey derives a 32-byte key from password and salt using Argon2id.
func DeriveKey(password string, salt []byte, params Argon2Parameters) []byte {
	return argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, 32)
}

// EncryptFile encrypts a source file to a destination file using the passphrase and algorithm.
func EncryptFile(srcPath, dstPath, password string, alg Algorithm, onProgress func(float64)) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	totalSize := srcInfo.Size()
	if totalSize == 0 {
		totalSize = 1
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Generate salt and base nonce
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	baseNonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, baseNonce); err != nil {
		return err
	}

	// Write header:
	// Magic (5B) + Version (1B) + Alg (1B) + Salt (16B) + BaseNonce (12B) + Time (4B) + Memory (4B) + Threads (1B)
	// Total Header Size = 44 bytes
	header := make([]byte, 44)
	copy(header[0:5], MagicBytes)
	header[5] = Version
	header[6] = byte(alg)
	copy(header[7:23], salt)
	copy(header[23:35], baseNonce)
	
	params := DefaultArgon2Params
	binary.BigEndian.PutUint32(header[35:39], params.Time)
	binary.BigEndian.PutUint32(header[39:43], params.Memory)
	header[43] = params.Threads

	if _, err := dstFile.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Derive key
	key := DeriveKey(password, salt, params)

	// Create AEAD cipher
	var aead cipher.AEAD
	switch alg {
	case AlgAES256GCM:
		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}
		aead, err = cipher.NewGCM(block)
		if err != nil {
			return err
		}
	case AlgChaCha20ID:
		aead, err = chacha20poly1305.New(key)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported algorithm: %d", alg)
	}

	// Encrypt chunk-by-chunk
	buf := make([]byte, ChunkSize)
	var counter uint32
	var bytesRead int64

	for {
		n, readErr := io.ReadFull(srcFile, buf)
		if n > 0 {
			bytesRead += int64(n)
			isLast := readErr == io.EOF || readErr == io.ErrUnexpectedEOF

			// Derive chunk nonce: 7 bytes baseNonce + 4 bytes counter + 1 byte last-chunk flag
			chunkNonce := make([]byte, 12)
			copy(chunkNonce[0:7], baseNonce[0:7])
			binary.BigEndian.PutUint32(chunkNonce[7:11], counter)
			if isLast {
				chunkNonce[11] = 1
			} else {
				chunkNonce[11] = 0
			}

			ciphertext := aead.Seal(nil, chunkNonce, buf[:n], nil)
			if _, writeErr := dstFile.Write(ciphertext); writeErr != nil {
				return fmt.Errorf("failed to write ciphertext: %w", writeErr)
			}

			counter++
			if onProgress != nil {
				onProgress(float64(bytesRead) / float64(totalSize))
			}

			if isLast {
				break
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read error: %w", readErr)
		}
	}

	return nil
}

// DecryptFile decrypts a source file to a destination file using the passphrase.
func DecryptFile(srcPath, dstPath, password string, onProgress func(float64)) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	totalSize := srcInfo.Size()
	if totalSize < 44 {
		return errors.New("file is too short to be encrypted by this app")
	}

	header := make([]byte, 44)
	if _, err := io.ReadFull(srcFile, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Verify header
	if string(header[0:5]) != MagicBytes {
		return errors.New("invalid file format: missing magic bytes")
	}
	if header[5] != Version {
		return fmt.Errorf("unsupported file version: %d", header[5])
	}

	alg := Algorithm(header[6])
	salt := header[7:23]
	baseNonce := header[23:35]
	
	var params Argon2Parameters
	params.Time = binary.BigEndian.Uint32(header[35:39])
	params.Memory = binary.BigEndian.Uint32(header[39:43])
	params.Threads = header[43]

	// Derive key
	key := DeriveKey(password, salt, params)

	// Create AEAD cipher
	var aead cipher.AEAD
	switch alg {
	case AlgAES256GCM:
		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}
		aead, err = cipher.NewGCM(block)
		if err != nil {
			return err
		}
	case AlgChaCha20ID:
		aead, err = chacha20poly1305.New(key)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported algorithm: %d", alg)
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Decrypt chunk-by-chunk
	ciphertextChunkSize := ChunkSize + TagSize
	buf := make([]byte, ciphertextChunkSize)
	var counter uint32
	var bytesRead int64 = 44 // Start after header

	for {
		n, readErr := io.ReadFull(srcFile, buf)
		if n > 0 {
			bytesRead += int64(n)
			
			// Probe if this is the last chunk: if we reach EOF or less bytes than full chunk size
			// or if the next read returns EOF/0 bytes.
			// The most reliable way: check if we are at EOF now.
			isLast := false
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				isLast = true
			} else {
				// Peek next byte to see if we are at EOF
				nextByte := make([]byte, 1)
				currentOffset, _ := srcFile.Seek(0, io.SeekCurrent)
				peekN, _ := srcFile.Read(nextByte)
				if peekN == 0 {
					isLast = true
				}
				// Seek back
				srcFile.Seek(currentOffset, io.SeekStart)
			}

			// Derive chunk nonce
			chunkNonce := make([]byte, 12)
			copy(chunkNonce[0:7], baseNonce[0:7])
			binary.BigEndian.PutUint32(chunkNonce[7:11], counter)
			if isLast {
				chunkNonce[11] = 1
			} else {
				chunkNonce[11] = 0
			}

			plaintext, err := aead.Open(nil, chunkNonce, buf[:n], nil)
			if err != nil {
				// Clean up destination on failure to avoid leaving partial decrypted files
				dstFile.Close()
				os.Remove(dstPath)
				return errors.New("decryption failed: incorrect password or corrupted file")
			}

			if _, writeErr := dstFile.Write(plaintext); writeErr != nil {
				return fmt.Errorf("failed to write plaintext: %w", writeErr)
			}

			counter++
			if onProgress != nil {
				onProgress(float64(bytesRead) / float64(totalSize))
			}

			if isLast {
				break
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read error: %w", readErr)
		}
	}

	return nil
}
