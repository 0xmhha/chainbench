package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSubscribeCancellationBoundsAnUnansweredAcknowledgement(t *testing.T) {
	connection := make(chan *websocket.Conn, 1)
	url := wsServer(t, func(c *websocket.Conn) { connection <- c; _, _, _ = c.ReadMessage() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		sub, err := Subscribe(ctx, url, "newHeads")
		if sub != nil {
			_ = sub.Close()
		}
		result <- err
	}()
	conn := <-connection
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("cancelled acknowledgement succeeded")
		}
	case <-time.After(300 * time.Millisecond):
		_ = conn.Close()
		<-result
		t.Fatal("cancellation did not interrupt subscription acknowledgement")
	}
}

func TestSubscribeMissingMethodRetainsTypedRPCError(t *testing.T) {
	url := wsServer(t, func(c *websocket.Conn) {
		_ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "error": map[string]any{"code": -32601, "message": "method not found"}})
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := Subscribe(ctx, url, "newHeads")
	var server *ServerError
	if !errors.As(err, &server) || server.Method != "eth_subscribe" || server.Code != -32601 {
		t.Fatalf("WS error lost its method/code: %v", err)
	}
}

func TestSubscribeRefusesAnEmptyAcknowledgementID(t *testing.T) {
	url := wsServer(t, func(c *websocket.Conn) { _ = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "result": ""}) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub, err := Subscribe(ctx, url, "newHeads")
	if sub != nil {
		_ = sub.Close()
	}
	if err == nil {
		t.Fatal("empty subscription ID was accepted")
	}
}
