package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsServer starts an httptest WebSocket server that speaks the eth_subscribe
// protocol, calling handle after the subscribe request is read.
func wsServer(t *testing.T, handle func(c *websocket.Conn)) string {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		var req struct {
			Method string `json:"method"`
		}
		if err := c.ReadJSON(&req); err != nil || req.Method != "eth_subscribe" {
			_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "error": map[string]any{"message": "bad request"}})
			return
		}
		handle(c)
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestSubscribe_StreamsNotifications(t *testing.T) {
	url := wsServer(t, func(c *websocket.Conn) {
		_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "0xsub1"})
		_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription",
			"params": map[string]any{"subscription": "0xsub1", "result": map[string]any{"number": "0x2a"}}})
		time.Sleep(100 * time.Millisecond)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sub, err := Subscribe(ctx, url, "newHeads")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()
	if sub.ID() != "0xsub1" {
		t.Errorf("id = %q, want 0xsub1", sub.ID())
	}
	select {
	case n := <-sub.Notifications():
		var head struct {
			Number string `json:"number"`
		}
		if err := json.Unmarshal(n, &head); err != nil || head.Number != "0x2a" {
			t.Errorf("notification = %s (%v)", n, err)
		}
	case <-ctx.Done():
		t.Fatal("no notification received")
	}
}

func TestSubscribe_Rejected(t *testing.T) {
	url := wsServer(t, func(c *websocket.Conn) {
		_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1,
			"error": map[string]any{"message": "notifications not supported"}})
		time.Sleep(50 * time.Millisecond)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := Subscribe(ctx, url, "newHeads"); err == nil ||
		!strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected a rejection error, got %v", err)
	}
}
