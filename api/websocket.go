// Package api — WebSocket signaling client for WebRTC peer coordination.
// Uses gorilla/websocket to maintain a persistent connection to the server
// for exchanging SDP offers, answers, and ICE candidates between peers.
package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// SignalMessage is the message format exchanged over the WebSocket signaling channel.
type SignalMessage struct {
	// Type is the message type: "offer", "answer", "ice-candidate", "join", "leave", "peer-joined", "peer-left".
	Type string `json:"type"`
	// Room is the target room name.
	Room string `json:"room,omitempty"`
	// To is the target user ID for directed messages.
	To string `json:"to,omitempty"`
	// From is the sender's user ID (set by the server on incoming messages).
	From string `json:"from,omitempty"`
	// Payload carries the SDP or ICE candidate data as raw JSON.
	Payload json.RawMessage `json:"payload,omitempty"`
}

// WSClient manages a WebSocket connection to the signaling server.
type WSClient struct {
	// conn is the underlying WebSocket connection.
	conn *websocket.Conn
	// Send is the channel for outgoing messages.
	Send chan SignalMessage
	// Recv is the channel for incoming messages.
	Recv chan SignalMessage
	// done signals that the connection has been closed.
	done chan struct{}
	// closeOnce ensures Close() is idempotent.
	closeOnce sync.Once
	// mu protects write operations on the connection.
	mu sync.Mutex
}

// Connect establishes a WebSocket connection to the signaling server.
// The serverURL should be the HTTP base URL (e.g., http://localhost:8080);
// it will be converted to ws:// or wss:// automatically.
func Connect(serverURL, token string) (*WSClient, error) {
	// Parse and convert the server URL to a WebSocket URL.
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL: %w", err)
	}

	// Convert http(s) to ws(s).
	switch parsedURL.Scheme {
	case "https":
		parsedURL.Scheme = "wss"
	default:
		parsedURL.Scheme = "ws"
	}

	// Set the WebSocket path and include the JWT as a query parameter.
	parsedURL.Path = "/ws"
	q := parsedURL.Query()
	q.Set("token", token)
	parsedURL.RawQuery = q.Encode()

	// Dial the WebSocket server.
	conn, _, err := websocket.DefaultDialer.Dial(parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signaling server: %w", err)
	}

	ws := &WSClient{
		conn: conn,
		Send: make(chan SignalMessage, 64),
		Recv: make(chan SignalMessage, 64),
		done: make(chan struct{}),
	}

	// Start the read and write loops in background goroutines.
	go ws.readLoop()
	go ws.writeLoop()

	return ws, nil
}

// SendMsg sends a signaling message through the WebSocket connection.
// This is a convenience method that writes directly rather than using the channel.
func (ws *WSClient) SendMsg(msg SignalMessage) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	select {
	case <-ws.done:
		return fmt.Errorf("websocket connection is closed")
	default:
	}

	return ws.conn.WriteJSON(msg)
}

// readLoop continuously reads messages from the WebSocket and dispatches
// them to the Recv channel. It runs until the connection is closed or an
// error occurs.
func (ws *WSClient) readLoop() {
	defer ws.Close()

	for {
		var msg SignalMessage
		err := ws.conn.ReadJSON(&msg)
		if err != nil {
			// Check if the connection was intentionally closed.
			select {
			case <-ws.done:
				return
			default:
			}

			// Connection error or server disconnect.
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure) {
				// Log-worthy error, but we just close gracefully.
			}
			return
		}

		select {
		case ws.Recv <- msg:
		case <-ws.done:
			return
		}
	}
}

// writeLoop reads messages from the Send channel and writes them to the
// WebSocket connection. It runs until the done channel is closed.
func (ws *WSClient) writeLoop() {
	defer ws.Close()

	for {
		select {
		case msg, ok := <-ws.Send:
			if !ok {
				// Send channel was closed.
				return
			}
			ws.mu.Lock()
			err := ws.conn.WriteJSON(msg)
			ws.mu.Unlock()
			if err != nil {
				return
			}
		case <-ws.done:
			return
		}
	}
}

// Close cleanly shuts down the WebSocket connection and signals all
// goroutines to stop. It is safe to call multiple times.
func (ws *WSClient) Close() {
	ws.closeOnce.Do(func() {
		close(ws.done)
		// Send a close message to the server before closing the connection.
		ws.mu.Lock()
		_ = ws.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		ws.mu.Unlock()
		ws.conn.Close()
	})
}

// Done returns a channel that is closed when the WebSocket connection is terminated.
func (ws *WSClient) Done() <-chan struct{} {
	return ws.done
}
