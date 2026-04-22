package wire

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeCommand_ValidEnvelope(t *testing.T) {
	input := []byte(`{"command":"network.load","args":{"name":"my-local"}}`)
	cmd, err := DecodeCommand(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cmd == nil {
		t.Fatal("cmd is nil")
	}
	if string(cmd.Command) != "network.load" {
		t.Errorf("command: got %q, want %q", cmd.Command, "network.load")
	}
	if cmd.Args == nil {
		t.Fatal("args is nil")
	}
}

func TestDecodeCommand_MalformedJSON(t *testing.T) {
	input := []byte(`{"command":"network.load"`) // truncated
	if _, err := DecodeCommand(bytes.NewReader(input)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecodeCommand_UnknownField(t *testing.T) {
	input := []byte(`{"command":"network.load","args":{},"bogus":1}`)
	_, err := DecodeCommand(bytes.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
	if !strings.Contains(err.Error(), "unknown field") && !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention unknown field: %v", err)
	}
}
