package crypto

import (
	"testing"
)

func TestSealAndOpenRoomKey(t *testing.T) {
	roomKey, err := GenerateRoomKey()
	if err != nil {
		t.Fatal(err)
	}

	alicePub, alicePriv, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := SealRoomKey(roomKey, alicePub, nil)
	if err != nil {
		t.Fatalf("SealRoomKey failed: %v", err)
	}

	opened, err := OpenRoomKey(sealed, alicePriv)
	if err != nil {
		t.Fatalf("OpenRoomKey failed: %v", err)
	}

	if string(opened) != string(roomKey) {
		t.Fatalf("opened key mismatch: got %x, expected %x", opened, roomKey)
	}
}

func TestSealAndOpenDifferentRecipient(t *testing.T) {
	roomKey, _ := GenerateRoomKey()
	alicePub, _, _ := GenerateX25519Keypair()
	_, bobPriv, _ := GenerateX25519Keypair()

	sealed, err := SealRoomKey(roomKey, alicePub, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = OpenRoomKey(sealed, bobPriv)
	if err == nil {
		t.Fatal("expected error opening with wrong private key, got nil")
	}
}

func TestSealRoomKeyInvalidRecipientKey(t *testing.T) {
	roomKey, _ := GenerateRoomKey()
	_, err := SealRoomKey(roomKey, []byte("too-short"), nil)
	if err == nil {
		t.Fatal("expected error for invalid recipient key size, got nil")
	}
}

func TestOpenRoomKeyTooShort(t *testing.T) {
	_, err := OpenRoomKey([]byte("short"), make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for too-short sealed data, got nil")
	}
}

func TestSealAndOpenRoundTripMultiple(t *testing.T) {
	for i := 0; i < 5; i++ {
		roomKey, _ := GenerateRoomKey()
		alicePub, alicePriv, _ := GenerateX25519Keypair()

		sealed, err := SealRoomKey(roomKey, alicePub, nil)
		if err != nil {
			t.Fatalf("iteration %d: SealRoomKey failed: %v", i, err)
		}

		opened, err := OpenRoomKey(sealed, alicePriv)
		if err != nil {
			t.Fatalf("iteration %d: OpenRoomKey failed: %v", i, err)
		}

		if string(opened) != string(roomKey) {
			t.Fatalf("iteration %d: key mismatch", i)
		}
	}
}
