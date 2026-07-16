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

var (
	ErrNoPeerConnection  = fmt.Errorf("peer connection not initialized")
	ErrDataChannelClosed = fmt.Errorf("data channel not open")
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
	log.Printf("[rtc] Start() called for user %s in room %s", p.config.UserID, p.config.RoomName)

	m := webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return fmt.Errorf("failed to register codecs: %w", err)
	}
	webrtcAPI := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

	config := webrtc.Configuration{
		ICEServers: p.config.ICEServers,
	}

	pc, err := webrtcAPI.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	p.pc = pc

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] ICE connection state change: %s", state)
		switch state {
		case webrtc.ICEConnectionStateConnected:
			log.Printf("[rtc] ICE connection established")
			p.mu.Lock()
			p.connected = true
			p.mu.Unlock()
		case webrtc.ICEConnectionStateDisconnected:
			fallthrough
		case webrtc.ICEConnectionStateFailed:
			log.Printf("[rtc] ICE connection lost: %s", state)
			p.mu.Lock()
			p.connected = false
			p.mu.Unlock()
		}
	})

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Printf("[rtc] ICE candidate gathering complete")
			return
		}
		candJSON, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			log.Printf("[rtc] failed to marshal ICE candidate: %v", err)
			return
		}
		log.Printf("[rtc] ICE candidate sent: %s", candidate.ToJSON().Candidate)
		p.config.WSClient.SendMsg(api.SignalMessage{
			Type:    "ice-candidate",
			Room:    p.config.RoomName,
			Payload: candJSON,
		})
	})

	pc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Printf("[rtc] signaling state: %s", state)
	})

	dc, err := pc.CreateDataChannel("chat", nil)
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}
	p.dc = dc

	dc.OnOpen(func() {
		log.Printf("[rtc] DataChannel OnOpen")
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
		log.Printf("[rtc] remote DataChannel received: %s (label=%s)", d.Label(), d.Label())
		p.mu.Lock()
		if p.dc == nil {
			p.dc = d
		}
		p.mu.Unlock()
		d.OnOpen(func() {
			log.Printf("[rtc] remote DataChannel OnOpen: %s", d.Label())
		})
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

	go p.handleSignaling()

	log.Printf("[rtc] Start() complete, waiting for peers...")
	return nil
}

func (p *Peer) handleSignaling() {
	log.Printf("[rtc] signaling handler started")
	for {
		select {
		case msg, ok := <-p.config.WSClient.Recv:
			if !ok {
				log.Printf("[rtc] signaling channel closed")
				return
			}

			switch msg.Type {
			case "offer":
				log.Printf("[rtc] offer received from %s", msg.From)
				var offer webrtc.SessionDescription
				if err := json.Unmarshal(msg.Payload, &offer); err != nil {
					log.Printf("[rtc] failed to unmarshal offer: %v", err)
					continue
				}
				p.mu.Lock()
				err := p.pc.SetRemoteDescription(offer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] SetRemoteDescription(offer) failed: %v", err)
					continue
				}
				log.Printf("[rtc] remote description set from offer, creating answer")
				answer, err := p.pc.CreateAnswer(nil)
				if err != nil {
					log.Printf("[rtc] CreateAnswer failed: %v", err)
					continue
				}
				p.mu.Lock()
				err = p.pc.SetLocalDescription(answer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] SetLocalDescription(answer) failed: %v", err)
					continue
				}
				answerJSON, _ := json.Marshal(answer)
				log.Printf("[rtc] answer sent to %s", msg.From)
				p.config.WSClient.SendMsg(api.SignalMessage{
					Type:    "answer",
					Room:    p.config.RoomName,
					To:      msg.From,
					Payload: answerJSON,
				})

			case "answer":
				log.Printf("[rtc] answer received from %s", msg.From)
				var answer webrtc.SessionDescription
				if err := json.Unmarshal(msg.Payload, &answer); err != nil {
					log.Printf("[rtc] failed to unmarshal answer: %v", err)
					continue
				}
				p.mu.Lock()
				err := p.pc.SetRemoteDescription(answer)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] SetRemoteDescription(answer) failed: %v", err)
				} else {
					log.Printf("[rtc] remote description set from answer")
				}

			case "ice-candidate":
				var candidate webrtc.ICECandidateInit
				if err := json.Unmarshal(msg.Payload, &candidate); err != nil {
					log.Printf("[rtc] failed to unmarshal ICE candidate: %v", err)
					continue
				}
				log.Printf("[rtc] ICE candidate received: %s", candidate.Candidate)
				p.mu.Lock()
				err := p.pc.AddICECandidate(candidate)
				p.mu.Unlock()
				if err != nil {
					log.Printf("[rtc] AddICECandidate failed: %v", err)
				}

			case "peer-joined":
				log.Printf("[rtc] peer-joined notification: %s", msg.From)
				if p.config.OnPeerJoined != nil {
					p.config.OnPeerJoined(msg.From)
				}
				p.createOfferForPeer(msg.From)

			case "peer-left":
				log.Printf("[rtc] peer-left notification: %s", msg.From)
				if p.config.OnPeerLeft != nil {
					p.config.OnPeerLeft(msg.From)
				}

			case "room-peers":
				var peers []map[string]interface{}
				if err := json.Unmarshal(msg.Payload, &peers); err != nil {
					log.Printf("[rtc] failed to unmarshal room-peers: %v", err)
					continue
				}
				log.Printf("[rtc] room-peers received: %d peer(s)", len(peers))
				for _, peer := range peers {
					if peerID, ok := peer["user_id"].(string); ok && peerID != p.config.UserID {
						log.Printf("[rtc] room contains peer: %s", peerID)
						if p.config.OnPeerJoined != nil {
							p.config.OnPeerJoined(peerID)
						}
					}
				}

			default:
				log.Printf("[rtc] unknown message type: %s", msg.Type)
			}
		case <-p.stopCh:
			log.Printf("[rtc] signaling handler stopped")
			return
		}
	}
}

func (p *Peer) createOfferForPeer(targetUserID string) {
	log.Printf("[rtc] CreateOffer for peer %s", targetUserID)

	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		log.Printf("[rtc] CreateOffer failed: %v", err)
		if p.config.OnError != nil {
			p.config.OnError(fmt.Errorf("failed to create offer: %w", err))
		}
		return
	}
	log.Printf("[rtc] offer created, setting local description")

	p.mu.Lock()
	err = p.pc.SetLocalDescription(offer)
	p.mu.Unlock()
	if err != nil {
		log.Printf("[rtc] SetLocalDescription(offer) failed: %v", err)
		if p.config.OnError != nil {
			p.config.OnError(fmt.Errorf("failed to set local description: %w", err))
		}
		return
	}

	offerJSON, err := json.Marshal(offer)
	if err != nil {
		log.Printf("[rtc] failed to marshal offer: %v", err)
		return
	}

	log.Printf("[rtc] offer sent to %s (type: %s)", targetUserID, offer.Type)
	p.config.WSClient.SendMsg(api.SignalMessage{
		Type:    "offer",
		Room:    p.config.RoomName,
		To:      targetUserID,
		Payload: offerJSON,
	})
}

func (p *Peer) SendMessage(plaintext []byte) error {
	data, err := EncodeMessage(plaintext, p.config.RoomKey, p.config.UserID, p.config.Username, p.config.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dc == nil || p.dc.ReadyState() != webrtc.DataChannelStateOpen {
		return ErrDataChannelClosed
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


