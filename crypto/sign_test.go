package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestSignVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("this is a test message")
	sig := Sign(message, priv)

	if !Verify(message, sig, pub) {
		t.Fatal("signature verification failed")
	}
}

func TestVerifyWrongMessage(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("original message")
	sig := Sign(message, priv)

	if Verify([]byte("tampered message"), sig, pub) {
		t.Fatal("signature should NOT verify for tampered message")
	}
}

func TestVerifyWrongKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("test")
	sig := Sign(message, priv)

	if Verify(message, sig, pub2) {
		t.Fatal("signature should NOT verify with different key")
	}
}

func TestVerifyInvalidPublicKeySize(t *testing.T) {
	if Verify([]byte("msg"), []byte("sig"), ed25519.PublicKey{}) {
		t.Fatal("Verify should return false for empty key")
	}
}

func TestVerifyInvalidSignatureSize(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	if Verify([]byte("msg"), []byte("short"), pub) {
		t.Fatal("Verify should return false for short signature")
	}
}
