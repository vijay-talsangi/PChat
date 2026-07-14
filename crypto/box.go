// Package crypto — asymmetric room key encryption using X25519 ECDH + AES-256-GCM.
//
// SealRoomKey encrypts a room's symmetric key to a specific recipient using an
// ephemeral X25519 key pair. The shared secret is derived via ECDH and hashed
// with SHA-256 to produce the AES key.
//
// Wire format: ephemeral_pub(32) || nonce(12) || ciphertext || tag(16)
package crypto

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

const (
	// ephemeralPubKeySize is the size of an X25519 public key.
	ephemeralPubKeySize = 32
)

// SealRoomKey encrypts roomKey for a specific recipient identified by their
// X25519 public key. An ephemeral X25519 key pair is generated for each
// invocation to ensure forward secrecy.
//
// The senderPrivKey parameter is accepted for API consistency but is not used
// in this ephemeral-key scheme — each call generates a fresh ephemeral keypair.
//
// Output format: ephemeral_pub(32) || nonce(12) || ciphertext || tag(16)
func SealRoomKey(roomKey, recipientPubKey, senderPrivKey []byte) ([]byte, error) {
	if len(recipientPubKey) != ephemeralPubKeySize {
		return nil, fmt.Errorf("invalid recipient public key size: expected %d, got %d",
			ephemeralPubKeySize, len(recipientPubKey))
	}

	// Generate an ephemeral X25519 key pair for this sealing operation.
	ephPub, ephPriv, err := GenerateX25519Keypair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral keypair: %w", err)
	}

	// Compute the shared secret via X25519 ECDH: scalar_mult(ephemeral_priv, recipient_pub).
	sharedSecret, err := curve25519.X25519(ephPriv, recipientPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Derive a 32-byte AES key from the shared secret using SHA-256.
	aesKey := sha256.Sum256(sharedSecret)

	// Encrypt the room key using AES-256-GCM with the derived key.
	// Encrypt returns: nonce(12) || ciphertext || tag(16)
	encrypted, err := Encrypt(roomKey, aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt room key: %w", err)
	}

	// Prepend the ephemeral public key so the recipient can derive the same shared secret.
	// Final format: ephemeral_pub(32) || nonce(12) || ciphertext || tag(16)
	sealed := make([]byte, 0, ephemeralPubKeySize+len(encrypted))
	sealed = append(sealed, ephPub...)
	sealed = append(sealed, encrypted...)

	return sealed, nil
}

// OpenRoomKey decrypts a sealed room key using the recipient's X25519 private key.
// It extracts the ephemeral public key from the sealed data, computes the shared
// secret, and decrypts the room key.
//
// Expected input format: ephemeral_pub(32) || nonce(12) || ciphertext || tag(16)
func OpenRoomKey(sealed, recipientPrivKey []byte) ([]byte, error) {
	// Minimum length: ephemeral pub key + nonce + GCM tag (no plaintext = empty room key check).
	minLen := ephemeralPubKeySize + gcmNonceSize + 16 // 16 is GCM overhead
	if len(sealed) < minLen {
		return nil, fmt.Errorf("sealed data too short: need at least %d bytes, got %d",
			minLen, len(sealed))
	}

	// Extract the ephemeral public key from the first 32 bytes.
	ephPub := sealed[:ephemeralPubKeySize]
	encrypted := sealed[ephemeralPubKeySize:]

	// Compute the same shared secret: scalar_mult(recipient_priv, ephemeral_pub).
	sharedSecret, err := curve25519.X25519(recipientPrivKey, ephPub)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Derive the same AES key from the shared secret.
	aesKey := sha256.Sum256(sharedSecret)

	// Decrypt the room key.
	roomKey, err := Decrypt(encrypted, aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt room key: %w", err)
	}

	return roomKey, nil
}
