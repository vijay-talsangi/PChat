package crypto

import (
	"testing"
)

func TestGenerateRoomKey(t *testing.T) {
	key, err := GenerateRoomKey()
	if err != nil {
		t.Fatalf("GenerateRoomKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateRoomKey()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("hello p2p chat!")
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %q, expected %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	_, err := Encrypt([]byte("hello"), []byte("too-short"))
	if err == nil {
		t.Fatal("expected error for invalid key size, got nil")
	}
}

func TestDecryptInvalidKeySize(t *testing.T) {
	_, err := Decrypt(make([]byte, 28), []byte("too-short"))
	if err == nil {
		t.Fatal("expected error for invalid key size, got nil")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := GenerateRoomKey()
	key2, _ := GenerateRoomKey()
	ciphertext, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key, got nil")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	key, _ := GenerateRoomKey()
	ciphertext, err := Encrypt([]byte("data"), key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[13] ^= 0xFF
	_, err = Decrypt(ciphertext, key)
	if err == nil {
		t.Fatal("expected error decrypting corrupted data, got nil")
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := Decrypt([]byte("short"), make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for too-short ciphertext, got nil")
	}
}

func TestEncryptUniqueNonces(t *testing.T) {
	key, _ := GenerateRoomKey()
	plaintext := []byte("same message")
	nonces := make(map[string]bool)
	for i := 0; i < 5; i++ {
		ciphertext, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatal(err)
		}
		nonce := string(ciphertext[:12])
		if nonces[nonce] {
			t.Fatal("duplicate nonce detected - nonces should be unique")
		}
		nonces[nonce] = true
	}
}

func TestGenerateRoomKeyUnique(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key, err := GenerateRoomKey()
		if err != nil {
			t.Fatal(err)
		}
		s := string(key)
		if keys[s] {
			t.Fatal("duplicate room key generated")
		}
		keys[s] = true
	}
}
