package crypto

import (
	"crypto/rand"
	"sync"
	"testing"
)

func TestNonceTracker(t *testing.T) {
	nt := NewNonceTracker()

	nonce := []byte("0123456789abcdef")
	peerID := "peer-1"

	if nt.IsDuplicate(peerID, nonce) {
		t.Fatal("nonce should not be duplicate before recording")
	}

	nt.Record(peerID, nonce)

	if !nt.IsDuplicate(peerID, nonce) {
		t.Fatal("nonce should be duplicate after recording")
	}
}

func TestNonceTrackerDifferentPeers(t *testing.T) {
	nt := NewNonceTracker()

	nonce := []byte("same-nonce-value!!")
	nt.Record("peer-1", nonce)

	if nt.IsDuplicate("peer-2", nonce) {
		t.Fatal("same nonce from different peer should not be duplicate")
	}
}

func TestNonceTrackerDifferentNonces(t *testing.T) {
	nt := NewNonceTracker()

	nt.Record("peer-1", []byte("nonce-one-------"))
	if nt.IsDuplicate("peer-1", []byte("nonce-two-------")) {
		t.Fatal("different nonce from same peer should not be duplicate")
	}
}

func TestNonceTrackerEmptyPeer(t *testing.T) {
	nt := NewNonceTracker()

	if nt.IsDuplicate("", []byte("nonce")) {
		t.Fatal("empty peer should not have duplicates")
	}

	nt.Record("", []byte("nonce"))
	if !nt.IsDuplicate("", []byte("nonce")) {
		t.Fatal("empty peer should have duplicate after recording")
	}
}

func TestNonceTrackerConcurrency(t *testing.T) {
	nt := NewNonceTracker()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nonce := make([]byte, 16)
			rand.Read(nonce)
			nt.Record("peer-concurrent", nonce)
		}(i)
	}

	wg.Wait()
}
