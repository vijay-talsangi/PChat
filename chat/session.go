package chat

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/pion/webrtc/v3"

	"github.com/vijay-talsangi/PChat/api"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
	"github.com/vijay-talsangi/PChat/rtc"
)

func roomKeyFingerprint(key []byte) string {
	if len(key) == 0 {
		return "empty"
	}
	h := sha256.Sum256(key)
	return hex.EncodeToString(h[:])
}

type Member struct {
	UserID           string
	Username         string
	PublicKey        string
	SigningPublicKey []byte
}

type MembersFunc func(roomName, token string) ([]Member, error)

type SessionConfig struct {
	RoomName    string
	UserID      string
	Username    string
	ServerURL   string
	Token       string
	RoomKey     []byte
	SigningKey  []byte
	APIClient   *api.Client
	MembersFunc MembersFunc
}

type Session struct {
	cfg              SessionConfig
	peer             *rtc.Peer
	wsClient         *api.WSClient
	members          map[string]string
	provisionedPeers map[string]bool
	mu               sync.RWMutex
	done             chan struct{}
	prompt           string
}

func NewSession(cfg SessionConfig) *Session {
	return &Session{
		cfg:              cfg,
		members:          make(map[string]string),
		provisionedPeers: make(map[string]bool),
		done:             make(chan struct{}),
	}
}

func (s *Session) Start() error {
	log.Printf("[session] Starting session: room=%s user=%s roomKey_hash=%s",
		s.cfg.RoomName, s.cfg.UserID, roomKeyFingerprint(s.cfg.RoomKey))
	var ics []webrtc.ICEServer
	turnCreds, turnErr := s.cfg.APIClient.GetTurnCredentials(s.cfg.RoomName)
	if turnErr == nil && turnCreds != nil {
		ics = rtc.BuildICEServers(turnCreds)
	} else if turnErr != nil {
		log.Printf("[session] TURN credentials unavailable (non-fatal): %v", turnErr)
	}

	wsClient, err := api.Connect(s.cfg.ServerURL, s.cfg.Token, s.cfg.RoomName)
	if err != nil {
		return fmt.Errorf("failed to connect to signaling server: %w", err)
	}
	s.wsClient = wsClient
	defer wsClient.Close()

	peer := rtc.NewPeer(rtc.PeerConfig{
		UserID:     s.cfg.UserID,
		Username:   s.cfg.Username,
		RoomName:   s.cfg.RoomName,
		RoomKey:    s.cfg.RoomKey,
		SigningKey: s.cfg.SigningKey,
		ICEServers: ics,
		WSClient:   wsClient,
		OnMessage:  s.onMessage,
		OnPeerJoined: func(userID string) {
			s.loadMemberKeys()
			s.mu.RLock()
			username := s.members[userID]
			s.mu.RUnlock()
			if username != "" && username != s.cfg.Username {
				ClearLine()
				PrintSystem(fmt.Sprintf("Peer joined: %s", username))
				fmt.Print(s.prompt)
			}
		},
		OnPeerLeft: func(userID string) {
			s.mu.Lock()
			username := s.members[userID]
			delete(s.members, userID)
			s.mu.Unlock()
			if username != "" {
				ClearLine()
				PrintSystem(fmt.Sprintf("Peer left: %s", username))
				fmt.Print(s.prompt)
			}
		},
		OnError: func(err error) {
			ClearLine()
			PrintError(fmt.Sprintf("RTC error: %v", err))
			fmt.Print(s.prompt)
		},
	})

	s.peer = peer

	if err := peer.Start(); err != nil {
		return fmt.Errorf("failed to start peer connection: %w", err)
	}

	s.loadMemberKeys()

	scanner := bufio.NewScanner(os.Stdin)
	s.prompt = fmt.Sprintf("%s> ", s.cfg.RoomName)
	fmt.Print(s.prompt)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print(s.prompt)
			continue
		}

		switch {
		case line == "/exit":
			PrintSystem("Leaving room...")
			peer.Close()
			return nil

		case line == "/help":
			PrintHelp()
			fmt.Print(s.prompt)
			continue

		case line == "/members":
			s.showMembers()
			fmt.Print(s.prompt)
			continue

		case strings.HasPrefix(line, "/"):
			ClearLine()
			PrintError(fmt.Sprintf("Unknown command: %s", line))
			fmt.Print(s.prompt)
			continue

		default:
			if err := peer.SendMessage([]byte(line)); err != nil {
				ClearLine()
				PrintError(fmt.Sprintf("Failed to send: %v", err))
			} else {
				ClearLine()
				PrintOwnMessage(line)
			}
			fmt.Print(s.prompt)
		}
	}

	return scanner.Err()
}

func (s *Session) onMessage(senderUsername string, plaintext []byte) {
	ClearLine()
	PrintMessage(senderUsername, string(plaintext))
	fmt.Print(s.prompt)
}

func (s *Session) loadMemberKeys() {
	members, err := s.cfg.MembersFunc(s.cfg.RoomName, s.cfg.Token)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.members = make(map[string]string)
	for _, m := range members {
		s.members[m.UserID] = m.Username
		if m.UserID != s.cfg.UserID && len(m.SigningPublicKey) > 0 {
			s.peer.AddSigningKey(m.UserID, m.SigningPublicKey)
		}
	}
	s.mu.Unlock()
	s.provisionRoomKeysForMissingMembers()
}

func (s *Session) provisionRoomKeysForMissingMembers() {
	if len(s.cfg.RoomKey) == 0 {
		return
	}
	missingMembers, err := s.cfg.APIClient.GetMembersWithoutKeys(s.cfg.RoomName)
	if err != nil {
		log.Printf("[session] failed to fetch members without keys: %v", err)
		return
	}
	for _, m := range missingMembers {
		if m.UserID == s.cfg.UserID {
			continue
		}
		s.mu.RLock()
		alreadyProvisioned := s.provisionedPeers[m.UserID]
		s.mu.RUnlock()
		if alreadyProvisioned {
			continue
		}
		pubKey, err := pcrypto.DecodeBase64(m.PublicKey)
		if err != nil {
			log.Printf("[session] invalid public key for %s: %v", m.UserID, err)
			continue
		}
		sealed, err := pcrypto.SealRoomKey(s.cfg.RoomKey, pubKey, nil)
		if err != nil {
			log.Printf("[session] failed to seal room key for %s: %v", m.UserID, err)
			continue
		}
		encryptedKey := pcrypto.EncodeBase64(sealed)
		if err := s.cfg.APIClient.UploadRoomKey(s.cfg.RoomName, m.UserID, encryptedKey); err != nil {
			log.Printf("[session] failed to upload room key for %s: %v", m.UserID, err)
			continue
		}
		s.mu.Lock()
		s.provisionedPeers[m.UserID] = true
		s.mu.Unlock()
		log.Printf("[session] provisioned room key for %s (%s) key_hash=%s", m.Username, m.UserID, roomKeyFingerprint(s.cfg.RoomKey))
	}
}

func (s *Session) showMembers() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	usernames := make([]string, 0, len(s.members))
	for _, username := range s.members {
		if username != s.cfg.Username {
			usernames = append(usernames, username)
		}
	}
	PrintMembers(usernames)
}
