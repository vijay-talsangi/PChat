package rtc

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"

	"github.com/vijay-talsangi/PChat/api"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
)

type PeerConfig struct {
	UserID       string
	Username     string
	RoomName     string
	RoomKey      []byte
	SigningKey   []byte
	ICEServers   []webrtc.ICEServer
	WSClient     *api.WSClient
	OnMessage    func(senderUsername string, plaintext []byte)
	OnPeerJoined func(userID string)
	OnPeerLeft   func(userID string)
	OnError      func(err error)
}

type Peer struct {
	config       PeerConfig
	pc           *webrtc.PeerConnection
	dc           *webrtc.DataChannel
	mu           sync.Mutex
	connected    bool
	nonceTracker *pcrypto.NonceTracker
	signingKeys  map[string]ed25519.PublicKey
	stopCh       chan struct{}
}

func NewPeer(cfg PeerConfig) *Peer {
	return &Peer{
		config:       cfg,
		nonceTracker: pcrypto.NewNonceTracker(),
		signingKeys:  make(map[string]ed25519.PublicKey),
		stopCh:       make(chan struct{}),
	}
}

func (p *Peer) Start() error {
	m := webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return fmt.Errorf("failed to register codecs: %w", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

	config := webrtc.Configuration{
		ICEServers: p.config.ICEServers,
	}

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	p.pc = pc

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] ICE state: %s", state)
		switch state {
		case webrtc.ICEConnectionStateConnected:
			p.mu.Lock()
			p.connected = true
			p.mu.Unlock()
		case webrtc.ICEConnectionStateDisconnected:
			fallthrough
		case webrtc.ICEConnectionStateFailed:
			p.mu.Lock()
			p.connected = false
			p.mu.Unlock()
		}
	})

	dc, err := pc.CreateDataChannel("chat", nil)
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}
	p.dc = dc

	dc.OnOpen(func() {
		log.Printf("[rtc] DataChannel open")
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		sender, plaintext, err := DecodeMessage(msg.Data, p.config.RoomKey, p.nonceTracker, p.getSigningKeys())
		if err != nil {
			if p.config.OnError != nil {
				p.config.OnError(fmt.Errorf("failed to decode message: %w", err))
			}
			return
		}
		if p.config.OnMessage != nil {
			p.config.OnMessage(sender, plaintext)
		}
	})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("[rtc] remote DataChannel: %s", d.Label())
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			sender, plaintext, err := DecodeMessage(msg.Data, p.config.RoomKey, p.nonceTracker, p.getSigningKeys())
			if err != nil {
				if p.config.OnError != nil {
					p.config.OnError(fmt.Errorf("failed to decode message: %w", err))
				}
				return
			}
			if p.config.OnMessage != nil {
				p.config.OnMessage(sender, plaintext)
			}
		})
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	offerJSON, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("failed to marshal offer: %w", err)
	}
	p.config.WSClient.SendMsg(api.SignalMessage{
		Type:    "offer",
		Room:    p.config.RoomName,
		Payload: offerJSON,
	})

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candJSON, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			return
		}
		p.config.WSClient.SendMsg(api.SignalMessage{
			Type:    "ice-candidate",
			Room:    p.config.RoomName,
			Payload: candJSON,
		})
	})

	go p.handleSignaling()

	return nil
}

func (p *Peer) handleSignaling() {
	for {
		select {
		case msg, ok := <-p.config.WSClient.Recv:
			if !ok {
				return
			}
			switch msg.Type {
			case "answer":
				var answer webrtc.SessionDescription
				if err := json.Unmarshal(msg.Payload, &answer); err != nil {
					log.Printf("[rtc] failed to unmarshal answer: %v", err)
					continue
				}
				p.mu.Lock()
				err := p.pc.SetRemoteDescription(answer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] failed to set remote description: %v", err)
				}

			case "offer":
				var offer webrtc.SessionDescription
				if err := json.Unmarshal(msg.Payload, &offer); err != nil {
					log.Printf("[rtc] failed to unmarshal offer: %v", err)
					continue
				}
				p.mu.Lock()
				err := p.pc.SetRemoteDescription(offer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] failed to set remote description: %v", err)
					continue
				}
				answer, err := p.pc.CreateAnswer(nil)
				if err != nil {
					log.Printf("[rtc] failed to create answer: %v", err)
					continue
				}
				p.mu.Lock()
				err = p.pc.SetLocalDescription(answer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] failed to set local description: %v", err)
					continue
				}
				answerJSON, _ := json.Marshal(answer)
				p.config.WSClient.SendMsg(api.SignalMessage{
					Type:    "answer",
					Room:    p.config.RoomName,
					To:      msg.From,
					Payload: answerJSON,
				})

			case "ice-candidate":
				var candidate webrtc.ICECandidateInit
				if err := json.Unmarshal(msg.Payload, &candidate); err != nil {
					log.Printf("[rtc] failed to unmarshal ICE candidate: %v", err)
					continue
				}
				p.mu.Lock()
				err := p.pc.AddICECandidate(candidate)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] failed to add ICE candidate: %v", err)
				}

			case "peer-joined":
				if p.config.OnPeerJoined != nil {
					p.config.OnPeerJoined(msg.From)
				}

			case "peer-left":
				if p.config.OnPeerLeft != nil {
					p.config.OnPeerLeft(msg.From)
				}

			case "room-peers":
				var peers []map[string]interface{}
				if err := json.Unmarshal(msg.Payload, &peers); err == nil {
					for _, peer := range peers {
						if peerID, ok := peer["user_id"].(string); ok && peerID != p.config.UserID {
							if p.config.OnPeerJoined != nil {
								p.config.OnPeerJoined(peerID)
							}
						}
					}
				}
			}
		case <-p.stopCh:
			return
		}
	}
}

func (p *Peer) SendMessage(plaintext []byte) error {
	data, err := EncodeMessage(plaintext, p.config.RoomKey, p.config.UserID, p.config.Username, p.config.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dc == nil || p.dc.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("data channel not open")
	}
	return p.dc.Send(data)
}

func (p *Peer) AddSigningKey(userID string, signingPubKey []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.signingKeys[userID] = ed25519.PublicKey(signingPubKey)
}

func (p *Peer) getSigningKeys() map[string]ed25519.PublicKey {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make(map[string]ed25519.PublicKey, len(p.signingKeys))
	for k, v := range p.signingKeys {
		keys[k] = v
	}
	return keys
}

func (p *Peer) Close() {
	close(p.stopCh)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dc != nil {
		p.dc.Close()
	}
	if p.pc != nil {
		p.pc.Close()
	}
}

func (p *Peer) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connected
}

func BuildICEServers(turnCreds *api.TurnCreds) []webrtc.ICEServer {
	var servers []webrtc.ICEServer
	for _, url := range turnCreds.URLs {
		servers = append(servers, webrtc.ICEServer{
			URLs:           []string{url},
			Username:       turnCreds.Username,
			Credential:     turnCreds.Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		})
	}
	servers = append(servers, webrtc.ICEServer{
		URLs: []string{"stun:stun.l.google.com:19302"},
	})
	return servers
}


