package schema

import (
	"path/filepath"
	"testing"
)

func TestNetworkSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"network-local.json",
		"network-remote.json",
		"network-hybrid.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("network", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestNetworkSchema_RejectsMissingChainType(t *testing.T) {
	doc := []byte(`{
		"name": "no-chain-type",
		"chain_id": 1,
		"nodes": [{"id":"n1","provider":"local","http":"http://127.0.0.1:8545"}]
	}`)
	if err := ValidateBytes("network", doc); err == nil {
		t.Fatal("expected validation error for missing chain_type")
	}
}

func TestCommandSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"command-network-load.json",
		"command-node-rpc.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("command", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestCommandSchema_RejectsUnknownCommand(t *testing.T) {
	doc := []byte(`{"command":"not.a.real.command","args":{}}`)
	if err := ValidateBytes("command", doc); err == nil {
		t.Fatal("expected validation error for unknown command name")
	}
}

func TestCommandSchema_RejectsMissingArgs(t *testing.T) {
	doc := []byte(`{"command":"network.load"}`)
	if err := ValidateBytes("command", doc); err == nil {
		t.Fatal("expected validation error for missing args")
	}
}

func TestEventSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"event-node-started.json",
		"event-chain-block.json",
		"event-progress.json",
		"event-result-ok.json",
		"event-result-error.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("event", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestEventSchema_RejectsUnknownEventName(t *testing.T) {
	doc := []byte(`{"type":"event","name":"not.a.real.event","ts":"2026-04-20T10:00:00Z"}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for unknown event name")
	}
}

func TestEventSchema_RejectsResultWithoutOk(t *testing.T) {
	doc := []byte(`{"type":"result"}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for result without ok field")
	}
}

func TestEventSchema_RejectsResultOkTrueWithError(t *testing.T) {
	doc := []byte(`{"type":"result","ok":true,"error":{"code":"INTERNAL","message":"contradictory"}}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for ok:true result carrying an error body")
	}
}

func TestEventSchema_RejectsResultOkFalseWithoutError(t *testing.T) {
	doc := []byte(`{"type":"result","ok":false}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for ok:false result without error body")
	}
}
