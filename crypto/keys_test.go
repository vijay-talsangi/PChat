package crypto

import (
	"encoding/base64"
	"testing"
)

func TestGenerateX25519Keypair(t *testing.T) {
	pub, priv, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair failed: %v", err)
	}
	if len(pub) != 32 {
		t.Fatalf("expected 32-byte public key, got %d", len(pub))
	}
	if len(priv) != 32 {
		t.Fatalf("expected 32-byte private key, got %d", len(priv))
	}
	if string(pub) == string(priv) {
		t.Fatal("public and private keys should differ")
	}
}

func TestGenerateEd25519Keypair(t *testing.T) {
	pub, priv, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair failed: %v", err)
	}
	if len(pub) != 32 {
		t.Fatalf("expected 32-byte public key, got %d", len(pub))
	}
	if len(priv) != 64 {
		t.Fatalf("expected 64-byte private key, got %d", len(priv))
	}
}

func TestEncodeDecodeBase64(t *testing.T) {
	original := []byte("hello world test data 12345")
	encoded := EncodeBase64(original)
	decoded, err := DecodeBase64(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64 failed: %v", err)
	}
	if string(original) != string(decoded) {
		t.Fatalf("round-trip base64 mismatch: got %q, expected %q", string(decoded), string(original))
	}
}

func TestDecodeBase64Invalid(t *testing.T) {
	_, err := DecodeBase64("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error decoding invalid base64, got nil")
	}
}

func TestGenerateX25519KeypairUnique(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		pub, _, err := GenerateX25519Keypair()
		if err != nil {
			t.Fatal(err)
		}
		s := base64.StdEncoding.EncodeToString(pub)
		if keys[s] {
			t.Fatal("duplicate public key generated")
		}
		keys[s] = true
	}
}
