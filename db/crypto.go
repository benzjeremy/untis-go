package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrInvalidCiphertext = errors.New("invalid or corrupted ciphertext")
	appCryptoSalt        = []byte("untis-go-secure-salt-v1.3.1-desktop-production")
	legacyCryptoSalt     = []byte("untis-go-secure-salt-v1.3.0-desktop-production")
)

// getMachineID retrieves the unique machine-id from standard Linux locations
func getMachineID() string {
	paths := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}

	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				return trimmed
			}
		}
	}

	// Fallback to hostname if machine-id is not accessible
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "untis-go-default-machine-key"
}

// deriveKeyWithSalt generates a 32-byte key for AES-256 using PBKDF2 with the specified salt
func deriveKeyWithSalt(salt []byte) ([]byte, error) {
	machID := getMachineID()

	username := os.Getenv("USER")
	if username == "" {
		if u, err := user.Current(); err == nil && u.Username != "" {
			username = u.Username
		} else {
			username = "untis-user"
		}
	}

	homeDir, _ := os.UserHomeDir()

	// Use PBKDF2 with SHA256, 100000 iterations, and 32-byte key length
	// This is much more resistant to brute-force attacks than plain SHA256
	var data []byte
	data = append(data, salt...)
	data = append(data, []byte(machID)...)
	data = append(data, []byte(username)...)
	data = append(data, []byte(homeDir)...)

	key := pbkdf2.Key(data, salt, 100000, 32, sha256.New)
	return key, nil
}

// DeriveKey generates a 32-byte key for AES-256 from machine ID and user environment
func DeriveKey() ([]byte, error) {
	return deriveKeyWithSalt(appCryptoSalt)
}

// EncryptPassword encrypts a plaintext password using AES-256-GCM
// It returns a base64-encoded string containing [12-byte nonce][ciphertext + 16-byte tag]
func EncryptPassword(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := DeriveKey()
	if err != nil {
		return "", fmt.Errorf("failed to derive encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	// Seal appends ciphertext and authentication tag to nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptWithKey attempts to decrypt ciphertext using a given key
func decryptWithKey(key []byte, raw []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize+gcm.Overhead() {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DecryptPassword decrypts a base64-encoded AES-256-GCM ciphertext
func DecryptPassword(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}

	key, err := DeriveKey()
	if err != nil {
		return "", fmt.Errorf("failed to derive decryption key: %w", err)
	}

	// Try standard key
	if pt, err := decryptWithKey(key, raw); err == nil {
		return pt, nil
	}

	// Try legacy salt key fallback (for credentials stored before v1.3 salt rotation)
	if legacyKey, err := deriveKeyWithSalt(legacyCryptoSalt); err == nil {
		if pt, err := decryptWithKey(legacyKey, raw); err == nil {
			return pt, nil
		}
	}

	return "", fmt.Errorf("decryption failed (wrong key or corrupted data)")
}
