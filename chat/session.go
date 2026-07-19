package chat

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	program          *tea.Program
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

func (s *Session) cleanup() {
	if s.peer != nil {
		s.peer.Close()
	}
}

type incomingMsg struct {
	username string
	text     string
}

type systemMsg struct {
	text string
}

type connStateMsg struct {
	state string
	text  string
}

type inviteResultMsg struct {
	code string
	err  error
}

type model struct {
	session    *Session
	viewport   viewport.Model
	textInput  textinput.Model
	messages   []string
	width      int
	height     int
	ready      bool
	connStatus string
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) addMessage(text string) {
	m.messages = append(m.messages, text)
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
	m.viewport.GotoBottom()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 8
		verticalMargin := headerHeight + 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}
		m.textInput.Width = msg.Width - 2
		if len(m.messages) > 0 {
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
		}
		return m, nil

	case incomingMsg:
		m.addMessage(StylePeerMessage(msg.username, msg.text))
		return m, nil

	case systemMsg:
		m.addMessage(StyleSystemMessage(msg.text))
		return m, nil

	case connStateMsg:
		m.connStatus = msg.state
		if msg.text != "" {
			m.addMessage(StyleSystemMessage(msg.text))
		}
		return m, nil

	case inviteResultMsg:
		if msg.err != nil {
			m.addMessage(StyleErrorMessage("Failed to create invite"))
		} else {
			m.addMessage(StyleSystemMessage(fmt.Sprintf("Invite code: %s", msg.code)))
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.session.cleanup()
			return m, tea.Quit

		case tea.KeyEnter:
			val := strings.TrimSpace(m.textInput.Value())
			m.textInput.Reset()
			if val == "" {
				return m, nil
			}
			switch {
			case val == "/exit":
				m.session.cleanup()
				return m, tea.Quit

			case val == "/help":
				m.addMessage(StyleHelp())
				return m, nil

			case val == "/members":
				m.addMessage(m.session.formatMembers())
				return m, nil

			case val == "/clear":
				m.messages = nil
				m.viewport.SetContent("")
				return m, nil

			case strings.HasPrefix(val, "/invite"):
				go func() {
					inv, err := m.session.cfg.APIClient.CreateInvite(m.session.cfg.RoomName, 1, 24)
					if err != nil {
						m.session.program.Send(inviteResultMsg{err: err})
						return
					}
					m.session.program.Send(inviteResultMsg{code: inv.Code})
				}()
				return m, nil

			case strings.HasPrefix(val, "/"):
				m.addMessage(StyleErrorMessage(fmt.Sprintf("Unknown command: %s", val)))
				return m, nil

			default:
				if err := m.session.peer.SendMessage([]byte(val)); err != nil {
					m.addMessage(StyleErrorMessage("Failed to send message"))
				} else {
					m.addMessage(StyleOwnMessage(val))
				}
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	header := RenderHeader(m.session.cfg.RoomName, m.session.cfg.Username, m.connStatus)
	inputPrompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B8B8B")).Render("❯ ")
	m.textInput.Prompt = inputPrompt
	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), m.textInput.View())
}

func (s *Session) formatMembers() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	usernames := make([]string, 0, len(s.members))
	for _, username := range s.members {
		if username != s.cfg.Username {
			usernames = append(usernames, username)
		}
	}
	return StyleMembers(usernames)
}

func (s *Session) Start() error {
	if !s.cfg.Debug {
		log.SetOutput(io.Discard)
	}

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

	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 0

	m := &model{
		session:    s,
		textInput:  ti,
		connStatus: "🟡 Connecting...",
		messages:   make([]string, 0, 64),
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	s.program = p

	errCh := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errCh <- err
	}()

	peer := rtc.NewPeer(rtc.PeerConfig{
		UserID:     s.cfg.UserID,
		Username:   s.cfg.Username,
		RoomName:   s.cfg.RoomName,
		RoomKey:    s.cfg.RoomKey,
		SigningKey: s.cfg.SigningKey,
		ICEServers: ics,
		WSClient:   wsClient,
		OnMessage: func(senderUsername string, plaintext []byte) {
			s.program.Send(incomingMsg{username: senderUsername, text: string(plaintext)})
		},
		OnPeerJoined: func(userID string) {
			s.loadMemberKeys()
			s.mu.RLock()
			username := s.members[userID]
			s.mu.RUnlock()
			if username != "" && username != s.cfg.Username {
				s.program.Send(systemMsg{text: fmt.Sprintf("%s joined the room", username)})
			}
		},
		OnPeerLeft: func(userID string) {
			s.mu.Lock()
			username := s.members[userID]
			delete(s.members, userID)
			s.mu.Unlock()
			if username != "" {
				s.program.Send(systemMsg{text: fmt.Sprintf("%s left the room", username)})
			}
		},
		OnError: func(err error) {
			if s.cfg.Debug {
				s.program.Send(systemMsg{text: fmt.Sprintf("Error: %v", err)})
			}
		},
		OnConnectionStateChange: func(state rtc.ConnectionState) {
			switch state {
			case rtc.ConnectionStateConnected:
				s.program.Send(connStateMsg{state: "🟢 Connected", text: "🟢 Connected to room"})
			case rtc.ConnectionStateConnecting:
				s.program.Send(connStateMsg{state: "🟡 Connecting...", text: ""})
			case rtc.ConnectionStateDisconnected:
				s.program.Send(connStateMsg{state: "🔴 Disconnected", text: "⚠ Connection lost"})
			case rtc.ConnectionStateFailed:
				s.program.Send(connStateMsg{state: "🔴 Failed", text: "🔴 Disconnected"})
			}
		},
	})

	s.peer = peer
	if err := peer.Start(); err != nil {
		p.Quit()
		return fmt.Errorf("failed to start peer connection: %w", err)
	}

	s.loadMemberKeys()

	return <-errCh
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
