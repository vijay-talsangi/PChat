package chat

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
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

type msgType int

const (
	msgTypePeer msgType = iota
	msgTypeOwn
	msgTypeSystem
	msgTypeWarning
	msgTypeError
)

type chatMessage struct {
	timestamp time.Time
	username  string
	text      string
	msgType   msgType
}

type incomingMsg struct {
	username string
	text     string
}

type systemMsg struct {
	text string
}

type connStateMsg struct {
	state connState
	text  string
}

type inviteResultMsg struct {
	code string
	err  error
}

type model struct {
	session      *Session
	viewport     viewport.Model
	textArea     textarea.Model
	chatMsgs     []chatMessage
	styledLns    []string
	width        int
	height       int
	ready        bool
	connState    connState
	unreadCnt    int
	lastPeerWid  int
}

func (m *model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) inputHeight() int {
	return m.textArea.Height() + 2
}

func (m *model) updateVpHeight() {
	headerH := 6
	indicatorH := 0
	if m.unreadCnt > 0 {
		indicatorH = 1
	}
	m.viewport.Height = m.height - headerH - m.inputHeight() - indicatorH
}

func (m *model) buildVpContent() string {
	return strings.Join(m.styledLns, "\n")
}

func (m *model) refreshVp() {
	m.viewport.SetContent(m.buildVpContent())
}

func (m *model) styleMessage(msg chatMessage, prevMsg *chatMessage) string {
	ts := formatTimestamp(msg.timestamp)
	vpWidth := m.width
	if vpWidth <= 0 {
		vpWidth = 80
	}

	switch msg.msgType {
	case msgTypePeer:
		showSender := true
		if prevMsg != nil && prevMsg.msgType == msgTypePeer && prevMsg.username == msg.username &&
			msg.timestamp.Sub(prevMsg.timestamp) <= 2*time.Minute {
			showSender = false
		}
		if showSender {
			prefix := lipgloss.NewStyle().Bold(true).Foreground(hashColor(msg.username)).Render("[" + msg.username + "]")
			prefixWidth := lipgloss.Width("[" + msg.username + "]  ")
			m.lastPeerWid = prefixWidth

			textWidth := vpWidth - prefixWidth
			if textWidth < 10 {
				textWidth = 10
			}
			wrapped := wrapText(msg.text, textWidth)
			lines := strings.Split(wrapped, "\n")

			indent := strings.Repeat(" ", prefixWidth)
			tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)

			var b strings.Builder
			for i, line := range lines {
				if i == 0 {
					b.WriteString(prefix)
					b.WriteString("  ")
					b.WriteString(line)
				} else {
					b.WriteString(indent)
					b.WriteString(line)
				}
				if i < len(lines)-1 {
					b.WriteString("\n")
				}
			}
			b.WriteString(tsStyled)
			return b.String()
		}

		indent := strings.Repeat(" ", m.lastPeerWid)
		textWidth := vpWidth - m.lastPeerWid
		if textWidth < 10 {
			textWidth = 10
		}
		wrapped := wrapText(msg.text, textWidth)
		lines := strings.Split(wrapped, "\n")
		tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)

		var b strings.Builder
		for i, line := range lines {
			b.WriteString(indent)
			b.WriteString(line)
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString(tsStyled)
		return b.String()

	case msgTypeOwn:
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#20C997")).Render("You >")
		prefixWidth := lipgloss.Width("You >  ")

		textWidth := vpWidth - prefixWidth
		if textWidth < 10 {
			textWidth = 10
		}
		wrapped := wrapText(msg.text, textWidth)
		lines := strings.Split(wrapped, "\n")

		indent := strings.Repeat(" ", prefixWidth)
		tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)

		var b strings.Builder
		for i, line := range lines {
			if i == 0 {
				b.WriteString(prefix)
				b.WriteString("  ")
				b.WriteString(line)
			} else {
				b.WriteString(indent)
				b.WriteString(line)
			}
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString(tsStyled)
		return b.String()

	case msgTypeSystem:
		text := "✓ " + msg.text
		tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)
		wrapped := wrapText(text, vpWidth-lipgloss.Width("  "+ts)-2)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B8B8B")).Render(wrapped) + tsStyled

	case msgTypeWarning:
		text := "⚠ " + msg.text
		tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)
		wrapped := wrapText(text, vpWidth-lipgloss.Width("  "+ts)-2)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render(wrapped) + tsStyled

	case msgTypeError:
		text := "✗ " + msg.text
		tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + ts)
		wrapped := wrapText(text, vpWidth-lipgloss.Width("  "+ts)-2)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(wrapped) + tsStyled
	}
	return ""
}

func (m *model) restyleAllMessages() {
	m.styledLns = make([]string, 0, len(m.chatMsgs))
	m.lastPeerWid = 0
	for i, msg := range m.chatMsgs {
		var prev *chatMessage
		if i > 0 {
			prev = &m.chatMsgs[i-1]
		}
		m.styledLns = append(m.styledLns, m.styleMessage(msg, prev))
	}
	m.refreshVp()
}

func (m *model) addMessage(msg chatMessage) {
	msg.timestamp = time.Now()
	var prev *chatMessage
	if len(m.chatMsgs) > 0 {
		prev = &m.chatMsgs[len(m.chatMsgs)-1]
	}
	styled := m.styleMessage(msg, prev)
	m.chatMsgs = append(m.chatMsgs, msg)
	m.styledLns = append(m.styledLns, styled)
	wasBottom := m.viewport.AtBottom()
	if wasBottom {
		m.refreshVp()
		m.viewport.GotoBottom()
	} else {
		m.unreadCnt++
		m.updateVpHeight()
		m.refreshVp()
	}
}

func (m *model) clearUnread() {
	if m.unreadCnt == 0 {
		return
	}
	m.unreadCnt = 0
	m.updateVpHeight()
	m.refreshVp()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
  case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 6
		indicatorH := 0
		if m.unreadCnt > 0 {
			indicatorH = 1
		}
		m.textArea.SetWidth(msg.Width - 6)
		vpHeight := m.height - headerH - m.inputHeight() - indicatorH
		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.YPosition = headerH
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		if len(m.chatMsgs) > 0 {
			m.restyleAllMessages()
		}
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		if m.viewport.AtBottom() {
			m.clearUnread()
		}
		return m, cmd

	case incomingMsg:
		m.addMessage(chatMessage{msgType: msgTypePeer, username: msg.username, text: msg.text})
		return m, nil

	case systemMsg:
		m.addMessage(chatMessage{msgType: msgTypeSystem, text: msg.text})
		return m, nil

	case connStateMsg:
		m.connState = msg.state
		if msg.text != "" {
			m.addMessage(chatMessage{msgType: msgTypeSystem, text: msg.text})
		}
		return m, nil

	case inviteResultMsg:
		if msg.err != nil {
			m.addMessage(chatMessage{msgType: msgTypeError, text: "Failed to create invite"})
		} else {
			m.addMessage(chatMessage{msgType: msgTypeSystem, text: fmt.Sprintf("Invite code: %s", msg.code)})
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			if m.viewport.AtBottom() {
				m.clearUnread()
			}
			return m, cmd

		case tea.KeyCtrlC:
			m.session.cleanup()
			return m, tea.Quit

		case tea.KeyEnter:
			if msg.Alt {
				var cmd tea.Cmd
				m.textArea, cmd = m.textArea.Update(msg)
				m.updateVpHeight()
				return m, cmd
			}
			val := strings.TrimSpace(m.textArea.Value())
			m.textArea.Reset()
			m.updateVpHeight()
			if val == "" {
				return m, nil
			}
			switch {
			case val == "/exit":
				m.session.cleanup()
				return m, tea.Quit

			case val == "/help":
				m.styledLns = append(m.styledLns, StyleHelp())
				m.refreshVp()
				return m, nil

			case val == "/members":
				m.styledLns = append(m.styledLns, m.session.formatMembers())
				m.refreshVp()
				return m, nil

			case val == "/clear":
				m.chatMsgs = nil
				m.styledLns = nil
				m.unreadCnt = 0
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
				m.addMessage(chatMessage{msgType: msgTypeError, text: fmt.Sprintf("Unknown command: %s", val)})
				return m, nil

			default:
				if err := m.session.peer.SendMessage([]byte(val)); err != nil {
					m.addMessage(chatMessage{msgType: msgTypeError, text: "Failed to send message"})
				} else {
					m.addMessage(chatMessage{msgType: msgTypeOwn, text: val})
				}
				return m, nil
			}

		case tea.KeyCtrlJ:
			// Ctrl+J (\n): insert newline via synthetic KeyEnter (textarea handles "enter" key)
			var cmd tea.Cmd
			m.textArea, cmd = m.textArea.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.updateVpHeight()
			return m, cmd

		default:
			var cmd tea.Cmd
			m.textArea, cmd = m.textArea.Update(msg)
			m.updateVpHeight()
			return m, cmd
		}

	default:
		return m, nil
	}
}

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	header := RenderHeader(m.width, m.session.cfg.RoomName, m.session.cfg.Username, m.connState)
	indicator := ""
	if m.unreadCnt > 0 {
		indicator = RenderUnreadIndicator(m.unreadCnt) + "\n"
	}
	inputView := m.textArea.View()
	renderedInput := RenderInput(inputView, m.width, m.textArea.Focused())
	return header + m.viewport.View() + "\n" + indicator + renderedInput
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

	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.MaxHeight = 6
	ta.Focus()

	m := &model{
		session:   s,
		textArea:  ta,
		connState: StateConnecting,
		chatMsgs:  make([]chatMessage, 0, 64),
		styledLns: make([]string, 0, 64),
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
				s.program.Send(connStateMsg{state: StateConnected, text: "Connected to room"})
			case rtc.ConnectionStateConnecting:
				s.program.Send(connStateMsg{state: StateConnecting, text: ""})
			case rtc.ConnectionStateDisconnected:
				s.program.Send(connStateMsg{state: StateDisconnected, text: "Connection lost"})
			case rtc.ConnectionStateFailed:
				s.program.Send(connStateMsg{state: StateFailed, text: "Disconnected"})
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
