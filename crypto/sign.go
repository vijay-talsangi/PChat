// Package crypto — Ed25519 digital signatures for message authentication.
package crypto

import (
	"crypto/ed25519"
)

// Sign produces an Ed25519 signature over the given message using the provided
// private key. The signature is 64 bytes and is deterministic (same message +
// key always produces the same signature).
func Sign(message []byte, privateKey ed25519.PrivateKey) []byte {
	return ed25519.Sign(privateKey, message)
}

// Verify checks an Ed25519 signature against the given message and public key.
// Returns true if the signature is valid, false otherwise.
func Verify(message, signature []byte, publicKey ed25519.PublicKey) bool {
	// Guard against invalid key sizes to avoid panics.
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	if len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}
