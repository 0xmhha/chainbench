package rpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSubscribeRejectsUncorrelatedAcknowledgements(t *testing.T) {
	for _, c := range []struct {
		name, version string
		id            any
	}{
		{"wrong ID", "2.0", 2},
		{"missing ID", "2.0", nil},
		{"string ID", "2.0", "1"},
		{"wrong version", "1.0", 1},
		{"missing version", "", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			url := wsServer(t, func(conn *websocket.Conn) {
				_ = conn.WriteJSON(map[string]any{"jsonrpc": c.version, "id": c.id, "result": "0xsub"})
			})
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			sub, err := Subscribe(ctx, url, "newHeads")
			if sub != nil {
				_ = sub.Close()
			}
			if err == nil {
				t.Fatal("uncorrelated acknowledgement reported successful subscription")
			}
		})
	}
}

func TestSubscribeOnlyDeliversItsOwnProtocolNotifications(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "0xours"})
		for _, c := range []struct{ version, subscription, number string }{
			{"2.0", "0xother", "0xbad"},
			{"1.0", "0xours", "0xbad"},
			{"2.0", "", "0xbad"},
			{"2.0", "0xours", "0x2a"},
		} {
			_ = conn.WriteJSON(map[string]any{"jsonrpc": c.version, "method": "eth_subscription", "params": map[string]any{"subscription": c.subscription, "result": map[string]any{"number": c.number}}})
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	sub, err := Subscribe(ctx, url, "newHeads")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	select {
	case raw := <-sub.Notifications():
		var head struct{ Number string }
		if err := json.Unmarshal(raw, &head); err != nil || head.Number != "0x2a" {
			t.Fatalf("unrelated or invalid protocol message reached this subscription: %s (%v)", raw, err)
		}
	case <-ctx.Done():
		t.Fatal("valid notification did not arrive")
	}
}
