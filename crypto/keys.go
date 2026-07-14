// Package crypto provides cryptographic primitives for the p2p-chat-cli client.
// This file handles key generation for X25519 (key exchange) and Ed25519 (signing).
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// GenerateX25519Keypair generates a new X25519 key pair for Diffie-Hellman key exchange.
// Returns the 32-byte public key and 32-byte private key, or an error if random
// byte generation fails.
func GenerateX25519Keypair() (publicKey, privateKey []byte, err error) {
	// Generate 32 random bytes for the private key (scalar).
	privateKey = make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, privateKey); err != nil {
		return nil, nil, fmt.Errorf("failed to generate X25519 private key: %w", err)
	}

	// Derive the public key by performing scalar multiplication of the
	// private key with the curve's base point.
	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive X25519 public key: %w", err)
	}

	return publicKey, privateKey, nil
}

// GenerateEd25519Keypair generates a new Ed25519 key pair for digital signatures.
// Returns the public and private key, or an error if key generation fails.
func GenerateEd25519Keypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
	}
	return pub, priv, nil
}
