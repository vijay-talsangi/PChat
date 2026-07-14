// Package crypto — nonce tracking for replay attack prevention.
// Each peer maintains a set of seen nonce hex strings, and any duplicate
// nonce from the same peer is rejected to prevent message replay.
package crypto

import (
	"encoding/hex"
	"sync"
)

// NonceTracker keeps a record of nonces seen from each peer to detect and
// prevent replay attacks on the data channel.
type NonceTracker struct {
	mu sync.Mutex
	// seen maps peerID -> set of hex-encoded nonces already processed.
	seen map[string]map[string]bool
}

// NewNonceTracker creates and returns an initialized NonceTracker.
func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		seen: make(map[string]map[string]bool),
	}
}

// IsDuplicate checks whether a nonce has already been seen from the given peer.
// This method is thread-safe.
func (nt *NonceTracker) IsDuplicate(peerID string, nonce []byte) bool {
	nonceHex := hex.EncodeToString(nonce)

	nt.mu.Lock()
	defer nt.mu.Unlock()

	peerNonces, exists := nt.seen[peerID]
	if !exists {
		return false
	}
	return peerNonces[nonceHex]
}

// Record stores a nonce as seen for the given peer. Future calls to
// IsDuplicate with the same peerID and nonce will return true.
// This method is thread-safe.
func (nt *NonceTracker) Record(peerID string, nonce []byte) {
	nonceHex := hex.EncodeToString(nonce)

	nt.mu.Lock()
	defer nt.mu.Unlock()

	if _, exists := nt.seen[peerID]; !exists {
		nt.seen[peerID] = make(map[string]bool)
	}
	nt.seen[peerID][nonceHex] = true
}
