// Package rtc — DataChannel message encoding/decoding.
// Defines the wire format for encrypted, signed chat messages exchanged
// over WebRTC data channels between peers.
//
// Message JSON format:
//
//	{
//	  "sender_id": "...",
//	  "sender_username": "...",
//	  "nonce": "<base64>",
//	  "ciphertext": "<base64>",
//	  "signature": "<base64>",
//	  "timestamp": 1234567890
//	}
package rtc

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	pcrypto "github.com/vijay-talsangi/PChat/crypto"
)

// DataMessage is the wire format for chat messages sent over WebRTC data channels.
type DataMessage struct {
	// SenderID is the unique user ID of the message sender.
	SenderID string `json:"sender_id"`
	// SenderUsername is the display name of the sender.
	SenderUsername string `json:"sender_username"`
	// Nonce is a base64-encoded random nonce for replay prevention.
	Nonce string `json:"nonce"`
	// Ciphertext is the base64-encoded AES-256-GCM encrypted message body.
	Ciphertext string `json:"ciphertext"`
	// Signature is the base64-encoded Ed25519 signature over (nonce || ciphertext).
	Signature string `json:"signature"`
	// Timestamp is the Unix timestamp (seconds) when the message was created.
	Timestamp int64 `json:"timestamp"`
}

// EncodeMessage encrypts and signs a plaintext message for transmission over
// a data channel. The message is encrypted with the room's AES-256 key and
// signed with the sender's Ed25519 private key.
//
// Returns the JSON-encoded DataMessage bytes.
func EncodeMessage(
	plaintext []byte,
	roomKey []byte,
	senderID, senderUsername string,
	signingKey ed25519.PrivateKey,
) ([]byte, error) {
	// Generate a random 16-byte nonce for replay attack prevention.
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate message nonce: %w", err)
	}

	// Encrypt the plaintext with the shared room key.
	ciphertext, err := pcrypto.Encrypt(plaintext, roomKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Sign the concatenation of nonce and ciphertext for integrity and authenticity.
	signedData := make([]byte, 0, len(nonce)+len(ciphertext))
	signedData = append(signedData, nonce...)
	signedData = append(signedData, ciphertext...)
	signature := pcrypto.Sign(signedData, signingKey)

	msg := DataMessage{
		SenderID:       senderID,
		SenderUsername: senderUsername,
		Nonce:          base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:     base64.StdEncoding.EncodeToString(ciphertext),
		Signature:      base64.StdEncoding.EncodeToString(signature),
		Timestamp:      time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	return data, nil
}

// DecodeMessage verifies, decrypts, and extracts a chat message from the
// wire format. It performs the following checks:
//  1. JSON decoding
//  2. Base64 decoding of all fields
//  3. Replay detection via the nonce tracker
//  4. Signature verification using the sender's Ed25519 public key
//  5. AES-256-GCM decryption
//
// Returns the sender's username and the decrypted plaintext.
func DecodeMessage(
	data []byte,
	roomKey []byte,
	nonceTracker *pcrypto.NonceTracker,
	signingKeys map[string]ed25519.PublicKey,
) (senderUsername string, plaintext []byte, err error) {
	var msg DataMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Decode base64 fields.
	nonce, err := base64.StdEncoding.DecodeString(msg.Nonce)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(msg.Ciphertext)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Check for replay attacks using the nonce tracker.
	if nonceTracker != nil {
		if nonceTracker.IsDuplicate(msg.SenderID, nonce) {
			return "", nil, fmt.Errorf("duplicate nonce detected from %s (replay attack?)", msg.SenderID)
		}
		nonceTracker.Record(msg.SenderID, nonce)
	}

	// Verify the message signature if we have the sender's public key.
	if pubKey, ok := signingKeys[msg.SenderID]; ok {
		signedData := make([]byte, 0, len(nonce)+len(ciphertext))
		signedData = append(signedData, nonce...)
		signedData = append(signedData, ciphertext...)
		if !pcrypto.Verify(signedData, signature, pubKey) {
			return "", nil, fmt.Errorf("invalid signature from %s", msg.SenderID)
		}
	}

	// Decrypt the message using the room's AES key.
	plaintext, err = pcrypto.Decrypt(ciphertext, roomKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	return msg.SenderUsername, plaintext, nil
}
