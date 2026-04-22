package wire

import (
	"strings"
	"testing"
)

func TestDecodeMessage_Event(t *testing.T) {
	line := []byte(`{"type":"event","name":"node.started","data":{"node_id":"node1"},"ts":"2026-04-22T10:00:00Z"}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	em, ok := msg.(EventMessage)
	if !ok {
		t.Fatalf("type: got %T, want EventMessage", msg)
	}
	if string(em.Name) != "node.started" {
		t.Errorf("name: got %q", em.Name)
	}
}

func TestDecodeMessage_Progress(t *testing.T) {
	line := []byte(`{"type":"progress","step":"init","done":2,"total":4}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	pm, ok := msg.(ProgressMessage)
	if !ok {
		t.Fatalf("type: got %T, want ProgressMessage", msg)
	}
	if pm.Step != "init" {
		t.Errorf("step: got %q", pm.Step)
	}
}

func TestDecodeMessage_ResultOK(t *testing.T) {
	line := []byte(`{"type":"result","ok":true,"data":{"blockNumber":"0x2a"}}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rm, ok := msg.(ResultMessage)
	if !ok {
		t.Fatalf("type: got %T, want ResultMessage", msg)
	}
	if rm.Ok != true {
		t.Errorf("ok: got %v", rm.Ok)
	}
}

func TestDecodeMessage_ResultError(t *testing.T) {
	line := []byte(`{"type":"result","ok":false,"error":{"code":"NOT_SUPPORTED","message":"no"}}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rm, ok := msg.(ResultMessage)
	if !ok {
		t.Fatalf("type: got %T, want ResultMessage", msg)
	}
	if rm.Ok != false {
		t.Errorf("ok: got %v", rm.Ok)
	}
	if rm.Error == nil || string(rm.Error.Code) != "NOT_SUPPORTED" {
		t.Errorf("error: got %+v", rm.Error)
	}
}

func TestDecodeMessage_UnknownType(t *testing.T) {
	line := []byte(`{"type":"bogus"}`)
	_, err := DecodeMessage(line)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "bogus") && !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error should mention unknown type: %v", err)
	}
}

func TestDecodeMessage_MalformedJSON(t *testing.T) {
	line := []byte(`{"type":"event"`) // truncated
	if _, err := DecodeMessage(line); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecodeMessage_RoundtripViaEmitter(t *testing.T) {
	// Emit via Emitter → decode via DecodeMessage → verify fields survive.
	var buf = new(stringBuf)
	e := NewEmitter(buf)
	if err := e.EmitEvent("chain.block", map[string]any{"height": 42}); err != nil {
		t.Fatal(err)
	}
	if err := e.EmitProgress("step1", 1, 3); err != nil {
		t.Fatal(err)
	}
	if err := e.EmitResult(true, map[string]any{"done": true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		msg, err := DecodeMessage([]byte(line))
		if err != nil {
			t.Fatalf("line %d decode: %v (%q)", i, err, line)
		}
		switch i {
		case 0:
			if _, ok := msg.(EventMessage); !ok {
				t.Errorf("line 0: got %T, want EventMessage", msg)
			}
		case 1:
			if _, ok := msg.(ProgressMessage); !ok {
				t.Errorf("line 1: got %T, want ProgressMessage", msg)
			}
		case 2:
			if _, ok := msg.(ResultMessage); !ok {
				t.Errorf("line 2: got %T, want ResultMessage", msg)
			}
		}
	}
}

// stringBuf is an io.Writer that captures output as a string,
// kept local to this test to avoid dependency on bytes package choices.
type stringBuf struct{ b strings.Builder }

func (s *stringBuf) Write(p []byte) (int, error) { return s.b.Write(p) }
func (s *stringBuf) String() string              { return s.b.String() }
