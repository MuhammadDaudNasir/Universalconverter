package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
)

// HashType represents the supported hashing algorithms.
type HashType string

const (
	HashMD5     HashType = "MD5"
	HashSHA1    HashType = "SHA-1"
	HashSHA256  HashType = "SHA-256"
	HashSHA512  HashType = "SHA-512"
	HashSHA3256 HashType = "SHA3-256"
	HashSHA3512 HashType = "SHA3-512"
)

// GenerateHash computes the hash of the input string in hexadecimal format.
func GenerateHash(input string, hashType HashType) string {
	data := []byte(input)
	switch hashType {
	case HashMD5:
		h := md5.Sum(data)
		return hex.EncodeToString(h[:])
	case HashSHA1:
		h := sha1.Sum(data)
		return hex.EncodeToString(h[:])
	case HashSHA256:
		h := sha256.Sum256(data)
		return hex.EncodeToString(h[:])
	case HashSHA512:
		h := sha512.Sum512(data)
		return hex.EncodeToString(h[:])
	case HashSHA3256:
		h := sha3.Sum256(data)
		return hex.EncodeToString(h[:])
	case HashSHA3512:
		h := sha3.Sum512(data)
		return hex.EncodeToString(h[:])
	default:
		return ""
	}
}

// EncodingType represents supported text representations.
type EncodingType string

const (
	EncHex       EncodingType = "Hex"
	EncBase64    EncodingType = "Base64"
	EncBase64URL EncodingType = "Base64 URL"
	EncURL       EncodingType = "URL Encode"
	EncBinary    EncodingType = "Binary"
)

// EncodeString encodes string to target format.
func EncodeString(input string, enc EncodingType) string {
	data := []byte(input)
	switch enc {
	case EncHex:
		return hex.EncodeToString(data)
	case EncBase64:
		return base64.StdEncoding.EncodeToString(data)
	case EncBase64URL:
		return base64.URLEncoding.EncodeToString(data)
	case EncURL:
		return url.QueryEscape(input)
	case EncBinary:
		var sb strings.Builder
		for _, b := range data {
			sb.WriteString(fmt.Sprintf("%08b ", b))
		}
		return strings.TrimSpace(sb.String())
	default:
		return ""
	}
}

// DecodeString decodes target format back to plain text.
func DecodeString(input string, enc EncodingType) (string, error) {
	switch enc {
	case EncHex:
		decoded, err := hex.DecodeString(strings.TrimSpace(input))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	case EncBase64:
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	case EncBase64URL:
		decoded, err := base64.URLEncoding.DecodeString(strings.TrimSpace(input))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	case EncURL:
		return url.QueryUnescape(input)
	case EncBinary:
		parts := strings.Fields(input)
		var decoded []byte
		for _, part := range parts {
			val, err := strconv.ParseUint(part, 2, 8)
			if err != nil {
				return "", fmt.Errorf("invalid binary element '%s': %w", part, err)
			}
			decoded = append(decoded, byte(val))
		}
		return string(decoded), nil
	default:
		return "", fmt.Errorf("unsupported decoding: %s", enc)
	}
}
