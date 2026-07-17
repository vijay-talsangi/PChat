package chat

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/pion/webrtc/v3"
	"github.com/vijay-talsangi/PChat/api"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
	"github.com/vijay-talsangi/PChat/rtc"
)

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
	Debug       bool
}

type Session struct {
	cfg              SessionConfig
	peer             *rtc.Peer
	wsClient         *api.WSClient
	members          map[string]string
	provisionedPeers map[string]bool
	mu               sync.RWMutex
	outputMu         sync.Mutex
	done             chan struct{}
}

func NewSession(cfg SessionConfig) *Session {
	return &Session{
		cfg:              cfg,
		members:          make(map[string]string),
		provisionedPeers: make(map[string]bool),
		done:             make(chan struct{}),
	}
}

func (s *Session) prompt() string {
	return fmt.Sprintf("%s ❯ ", s.cfg.RoomName)
}

func (s *Session) safePrint(fn func()) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	ClearLine()
	fn()
	fmt.Print(s.prompt())
}

func (s *Session) cleanup() {
	if s.peer != nil {
		s.peer.Close()
	}
	fmt.Println()
}

func (s *Session) Start() error {
	if !s.cfg.Debug {
		log.SetOutput(io.Discard)
	}

	PrintHeader(s.cfg.RoomName, s.cfg.Username)

	var ics []webrtc.ICEServer
	turnCreds, turnErr := s.cfg.APIClient.GetTurnCredentials(s.cfg.RoomName)
	if turnErr == nil && turnCreds != nil {
		ics = rtc.BuildICEServers(turnCreds)
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
				s.safePrint(func() {
					PrintSystem(fmt.Sprintf("%s joined", username))
				})
			}
		},
		OnPeerLeft: func(userID string) {
			s.mu.Lock()
			username := s.members[userID]
			delete(s.members, userID)
			s.mu.Unlock()
			if username != "" {
				s.safePrint(func() {
					PrintSystem(fmt.Sprintf("%s left", username))
				})
			}
		},
		OnError: func(err error) {
			if s.cfg.Debug {
				s.safePrint(func() {
					PrintError(fmt.Sprintf("%v", err))
				})
			}
		},
		OnConnectionStateChange: func(state rtc.ConnectionState) {
			switch state {
			case rtc.ConnectionStateConnected:
				s.safePrint(func() {
					PrintConnected()
				})
			case rtc.ConnectionStateConnecting:
				s.safePrint(func() {
					PrintConnecting()
				})
			case rtc.ConnectionStateDisconnected:
				s.safePrint(func() {
					PrintWarning("Connection lost")
				})
			case rtc.ConnectionStateFailed:
				s.safePrint(func() {
					PrintDisconnected()
				})
			}
		},
	})

	s.peer = peer

	if err := peer.Start(); err != nil {
		return fmt.Errorf("failed to start peer connection: %w", err)
	}

	s.loadMemberKeys()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(s.prompt())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		<-sigCh
		s.outputMu.Lock()
		ClearLine()
		s.outputMu.Unlock()
		s.cleanup()
		fmt.Println("Bye!")
		os.Exit(0)
	}()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print(s.prompt())
			continue
		}

		switch {
		case line == "/exit":
			s.cleanup()
			fmt.Println("Bye!")
			return nil

		case line == "/help":
			s.safePrint(func() {
				PrintHelp()
			})
			continue

		case line == "/members":
			s.safePrint(func() {
				s.showMembers()
			})
			continue

		case line == "/clear":
			s.outputMu.Lock()
			fmt.Print("\033[2J\033[H")
			PrintHeader(s.cfg.RoomName, s.cfg.Username)
			fmt.Print(s.prompt())
			s.outputMu.Unlock()
			continue

		case strings.HasPrefix(line, "/invite"):
			s.safePrint(func() {
				inv, err := s.cfg.APIClient.CreateInvite(s.cfg.RoomName, 1, 24)
				if err != nil {
					PrintError("Failed to create invite")
					return
				}
				fmt.Printf("Invite code: %s\n", inv.Code)
			})
			continue

		case strings.HasPrefix(line, "/"):
			s.safePrint(func() {
				PrintError(fmt.Sprintf("Unknown command: %s", line))
			})
			continue

		default:
			if err := peer.SendMessage([]byte(line)); err != nil {
				s.safePrint(func() {
					PrintError("Failed to send message")
				})
			} else {
				s.safePrint(func() {
					PrintOwnMessage(line)
				})
			}
		}
	}

	return scanner.Err()
}

func (s *Session) onMessage(senderUsername string, plaintext []byte) {
	s.safePrint(func() {
		PrintMessage(senderUsername, string(plaintext))
	})
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
			continue
		}
		sealed, err := pcrypto.SealRoomKey(s.cfg.RoomKey, pubKey, nil)
		if err != nil {
			continue
		}
		encryptedKey := pcrypto.EncodeBase64(sealed)
		if err := s.cfg.APIClient.UploadRoomKey(s.cfg.RoomName, m.UserID, encryptedKey); err != nil {
			continue
		}
		s.mu.Lock()
		s.provisionedPeers[m.UserID] = true
		s.mu.Unlock()
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
