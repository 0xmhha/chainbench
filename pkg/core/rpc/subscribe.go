package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

// Subscription is an active eth_subscribe stream over a WebSocket. Notifications
// yields the result of each subscription notification; the caller must Close it.
type Subscription struct {
	conn          *websocket.Conn
	id            string
	notifications chan json.RawMessage
}

// ID returns the server-assigned subscription id.
func (s *Subscription) ID() string { return s.id }

// Notifications is the channel of notification results (closed on error/Close).
func (s *Subscription) Notifications() <-chan json.RawMessage { return s.notifications }

// Subscribe opens a WebSocket to wsURL, sends eth_subscribe with params (e.g.
// "newHeads", or "logs" and a filter object), and returns a Subscription that
// streams each notification's result. The caller must Close it; cancelling ctx
// also ends the stream.
func Subscribe(ctx context.Context, wsURL string, params ...any) (*Subscription, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rpc: dial %s: %w", wsURL, err)
	}
	req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "eth_subscribe", "params": params}
	if err := conn.WriteJSON(req); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rpc: send eth_subscribe: %w", err)
	}
	var resp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := conn.ReadJSON(&resp); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rpc: read subscribe response: %w", err)
	}
	if resp.Error != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rpc: eth_subscribe rejected: %s", resp.Error.Message)
	}

	sub := &Subscription{conn: conn, id: resp.Result, notifications: make(chan json.RawMessage, 16)}
	// Cancelling ctx closes the conn, which unblocks the blocking read below.
	go func() { <-ctx.Done(); _ = conn.Close() }()
	go sub.readLoop()
	return sub, nil
}

func (s *Subscription) readLoop() {
	defer close(s.notifications)
	for {
		var msg struct {
			Method string `json:"method"`
			Params struct {
				Result json.RawMessage `json:"result"`
			} `json:"params"`
		}
		if err := s.conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.Method == "eth_subscription" && len(msg.Params.Result) > 0 {
			s.notifications <- msg.Params.Result
		}
	}
}

// Close ends the subscription and closes the WebSocket.
func (s *Subscription) Close() error { return s.conn.Close() }
