// Package crypto — AES-256-GCM symmetric encryption for room message confidentiality.
// Wire format: nonce(12 bytes) || ciphertext || GCM tag (16 bytes, appended by GCM).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// aesKeySize is the required key length for AES-256.
	aesKeySize = 32
	// gcmNonceSize is the standard nonce size for AES-GCM (96 bits).
	gcmNonceSize = 12
)

// GenerateRoomKey generates a cryptographically random 32-byte AES-256 key
// suitable for encrypting room messages.
func GenerateRoomKey() ([]byte, error) {
	key := make([]byte, aesKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate room key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the given 32-byte key.
// A random 12-byte nonce is generated and prepended to the output.
// Output format: nonce(12) || ciphertext || tag(16).
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != aesKeySize {
		return nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", aesKeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce for this encryption operation.
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext and GCM tag to the nonce prefix.
	// Result: nonce || ciphertext || tag
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data encrypted by Encrypt using AES-256-GCM.
// Expects input format: nonce(12) || ciphertext || tag(16).
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != aesKeySize {
		return nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", aesKeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Validate minimum ciphertext length: nonce + at least the GCM tag.
	if len(ciphertext) < gcmNonceSize+gcm.Overhead() {
		return nil, fmt.Errorf("ciphertext too short: need at least %d bytes, got %d",
			gcmNonceSize+gcm.Overhead(), len(ciphertext))
	}

	// Split the nonce from the ciphertext+tag portion.
	nonce := ciphertext[:gcmNonceSize]
	encrypted := ciphertext[gcmNonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
