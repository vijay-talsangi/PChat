package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
)

type testDataMessage struct {
	SenderID       string `json:"sender_id"`
	SenderUsername string `json:"sender_username"`
	Nonce          string `json:"nonce"`
	Ciphertext     string `json:"ciphertext"`
	Signature      string `json:"signature"`
	Timestamp      int64  `json:"timestamp"`
}

func TestFullPipeline(t *testing.T) {
	roomKey := make([]byte, 32)
	rand.Read(roomKey)

	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	senderID := "test-user-1"
	senderUsername := "alice"
	plaintext := []byte("hi")

	// Step 1: Generate replay nonce (16 bytes)
	nonce := make([]byte, 16)
	rand.Read(nonce)

	// Step 2: Encrypt plaintext with AES-256-GCM
	ciphertext, err := Encrypt(plaintext, roomKey)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	t.Logf("fingerprint(first 8 bytes of sha256): %x", sha256Hash(roomKey)[:8])
	t.Logf("full sha256 of roomKey: %x", sha256Hash(roomKey))
	t.Logf("roomKey length: %d", len(roomKey))
	t.Logf("ciphertext length: %d", len(ciphertext))
	t.Logf("ciphertext hex: %x", ciphertext)
	t.Logf("gcm nonce (first 12 bytes): %x", ciphertext[:12])
	t.Logf("encrypted data + tag (remaining %d bytes): %x", len(ciphertext)-12, ciphertext[12:])

	// Step 3: Sign
	signedData := make([]byte, 0, len(nonce)+len(ciphertext))
	signedData = append(signedData, nonce...)
	signedData = append(signedData, ciphertext...)
	signature := Sign(signedData, signingKey)

	// Step 4: Build message
	msg := testDataMessage{
		SenderID:       senderID,
		SenderUsername: senderUsername,
		Nonce:          base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:     base64.StdEncoding.EncodeToString(ciphertext),
		Signature:      base64.StdEncoding.EncodeToString(signature),
		Timestamp:      1234567890,
	}

	wireData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	t.Logf("wire length: %d", len(wireData))

	// Step 5: Simulate receiving the message
	var decoded testDataMessage
	if err := json.Unmarshal(wireData, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	decodedNonce, err := base64.StdEncoding.DecodeString(decoded.Nonce)
	if err != nil {
		t.Fatalf("decode nonce failed: %v", err)
	}
	decodedCiphertext, err := base64.StdEncoding.DecodeString(decoded.Ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext failed: %v", err)
	}
	decodedSignature, err := base64.StdEncoding.DecodeString(decoded.Signature)
	if err != nil {
		t.Fatalf("decode signature failed: %v", err)
	}

	// Verify lengths match
	if len(decodedCiphertext) != len(ciphertext) {
		t.Fatalf("ciphertext length mismatch: decode %d vs original %d",
			len(decodedCiphertext), len(ciphertext))
	}

	// Verify ciphertext bytes are identical
	for i := range ciphertext {
		if decodedCiphertext[i] != ciphertext[i] {
			t.Fatalf("ciphertext byte %d mismatch: decoded %x vs original %x",
				i, decodedCiphertext[i], ciphertext[i])
		}
	}

	// Verify signature
	signedDataVerif := make([]byte, 0, len(decodedNonce)+len(decodedCiphertext))
	signedDataVerif = append(signedDataVerif, decodedNonce...)
	signedDataVerif = append(signedDataVerif, decodedCiphertext...)
	if !Verify(signedDataVerif, decodedSignature, signingKey.Public().(ed25519.PublicKey)) {
		t.Fatal("signature verification FAILED")
	}
	t.Log("signature verification PASSED")

	// Verify decryption with SAME key
	result, err := Decrypt(decodedCiphertext, roomKey)
	if err != nil {
		t.Fatalf("Decrypt with same key FAILED: %v", err)
	}
	if string(result) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %q, expected %q", string(result), string(plaintext))
	}
	t.Logf("Decrypt with same key PASSED: %q", string(result))

	// Verify decryption with DIFFERENT key fails
	differentKey := make([]byte, 32)
	rand.Read(differentKey)
	_, err = Decrypt(decodedCiphertext, differentKey)
	if err == nil {
		t.Fatal("Decrypt with wrong key should have failed but succeeded")
	}
	t.Logf("Decrypt with different key correctly FAILED: %v", err)

	// Test with single-byte corruption of ciphertext
	corruptedCiphertext := make([]byte, len(decodedCiphertext))
	copy(corruptedCiphertext, decodedCiphertext)
	corruptedCiphertext[12] ^= 0x01 // flip one bit in the encrypted data (not nonce)
	_, err = Decrypt(corruptedCiphertext, roomKey)
	if err == nil {
		t.Fatal("Decrypt with corrupted ciphertext should have failed but succeeded")
	}
	t.Logf("Decrypt with corrupted ciphertext correctly FAILED: %v", err)
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
